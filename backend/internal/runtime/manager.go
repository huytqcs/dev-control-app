package runtime

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"devctl/internal/applog"
	"devctl/internal/config"
	"devctl/internal/logs"
	"devctl/internal/workspace"
)

var (
	ErrServiceNotFound       = errors.New("service not found")
	ErrServiceAlreadyRunning = errors.New("service already running")
	ErrServiceNotRunning     = errors.New("service not running")

	ErrWorkerNotFound       = errors.New("worker not found")
	ErrWorkerAlreadyRunning = errors.New("worker already running")
	ErrWorkerNotRunning     = errors.New("worker not running")
)

// HealthMonitor starts/stops per-service health probing. Implemented by
// internal/health.Monitor; defined here (structurally, not by import) so the
// runtime package doesn't need to depend on the health package's own
// dependencies — only app.go, the composition root, wires the two together.
type HealthMonitor interface {
	Start(serviceID string, checks []config.HealthCheck)
	Stop(serviceID string)
}

type noopHealthMonitor struct{}

func (noopHealthMonitor) Start(string, []config.HealthCheck) {}
func (noopHealthMonitor) Stop(string)                        {}

// GitProbe reads git status for a repo path. Implemented by
// runtime.GitAdapter wrapping internal/git.Service.
type GitProbe interface {
	Status(ctx context.Context, repoPath string) (GitState, error)
}

type noopGitProbe struct{}

func (noopGitProbe) Status(context.Context, string) (GitState, error) { return GitState{}, nil }

// DefaultStopTimeout is how long StopService waits for SIGTERM to take
// effect before escalating to SIGKILL.
const DefaultStopTimeout = 5 * time.Second

// Manager owns the lifecycle of every configured service: starting/stopping
// processes, tracking state transitions, and forwarding their stdout/stderr
// into the log manager. It is the single source of truth for runtime state
// (ARCHITECTURE.md §6.1).
type Manager struct {
	runner      ProcessRunner
	logs        *logs.Manager
	publisher   EventPublisher
	stopTimeout time.Duration
	health      HealthMonitor
	git         GitProbe

	mu       sync.RWMutex
	order    []string
	services map[string]*serviceRuntime
}

func NewManager(ws *workspace.Service, runner ProcessRunner, logMgr *logs.Manager, publisher EventPublisher) *Manager {
	if publisher == nil {
		publisher = NoopPublisher{}
	}
	m := &Manager{
		runner:      runner,
		logs:        logMgr,
		publisher:   publisher,
		stopTimeout: DefaultStopTimeout,
		health:      noopHealthMonitor{},
		git:         noopGitProbe{},
		services:    make(map[string]*serviceRuntime),
	}
	for _, svc := range ws.ListServices() {
		m.services[svc.ID] = newServiceRuntime(svc)
		m.order = append(m.order, svc.ID)
	}
	return m
}

// SetHealthMonitor wires in the real health checker (app.go composition
// root). Must be called before any service starts if health checks should
// take effect for it.
func (m *Manager) SetHealthMonitor(hm HealthMonitor) {
	if hm != nil {
		m.health = hm
	}
}

// SetGitProbe wires in the real git status reader (app.go composition root).
func (m *Manager) SetGitProbe(gp GitProbe) {
	if gp != nil {
		m.git = gp
	}
}

func (m *Manager) ListStates() []ServiceState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	states := make([]ServiceState, 0, len(m.order))
	for _, id := range m.order {
		states = append(states, m.services[id].snapshot())
	}
	return states
}

func (m *Manager) GetState(id string) (ServiceState, bool) {
	sr, err := m.getRuntime(id)
	if err != nil {
		return ServiceState{}, false
	}
	return sr.snapshot(), true
}

func (m *Manager) getRuntime(id string) (*serviceRuntime, error) {
	m.mu.RLock()
	sr, ok := m.services[id]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrServiceNotFound, id)
	}
	return sr, nil
}

func (m *Manager) emit(evtType EventType, serviceID string, payload interface{}) {
	m.publisher.Publish(AppEvent{
		Type:      evtType,
		ServiceID: serviceID,
		Payload:   payload,
		Time:      time.Now(),
	})
}

// StartService starts the configured process for id. It is a no-op error
// (ErrServiceAlreadyRunning) if the service is already running or starting;
// individual-service start deliberately does not cascade to dependencies
// (SPEC.md §25 / ARCHITECTURE.md §20).
func (m *Manager) StartService(ctx context.Context, id string) error {
	sr, err := m.getRuntime(id)
	if err != nil {
		return err
	}

	sr.mu.Lock()
	if sr.state.Status == ServiceRunning || sr.state.Status == ServiceStarting {
		sr.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrServiceAlreadyRunning, id)
	}
	sr.state.Status = ServiceStarting
	sr.state.LastError = ""
	sr.state.LastExitCode = nil
	cfg := sr.config
	sr.mu.Unlock()
	m.emit(EventServiceUpdated, id, sr.snapshot())

	// Checked explicitly because a missing cwd otherwise surfaces as a
	// misleading "fork/exec <shell>: no such file or directory" (a Go
	// stdlib quirk: a failed chdir combined with Setpgid gets mislabeled
	// as an exec failure on the shell binary instead of a chdir failure).
	if info, statErr := os.Stat(cfg.Path); statErr != nil || !info.IsDir() {
		startErr := fmt.Errorf("service path does not exist or is not a directory: %s", cfg.Path)
		sr.mu.Lock()
		sr.state.Status = ServiceFailed
		sr.state.LastError = startErr.Error()
		sr.mu.Unlock()
		m.emit(EventServiceUpdated, id, sr.snapshot())
		return fmt.Errorf("start service %q: %w", id, startErr)
	}

	proc, err := m.runner.Start(ctx, ProcessOptions{
		ID:      cfg.ID,
		Command: cfg.StartCommand,
		Dir:     cfg.Path,
		Env:     cfg.Env,
	})
	if err != nil {
		sr.mu.Lock()
		sr.state.Status = ServiceFailed
		sr.state.LastError = err.Error()
		sr.mu.Unlock()
		m.emit(EventServiceUpdated, id, sr.snapshot())
		return fmt.Errorf("start service %q: %w", id, err)
	}

	now := time.Now()
	sr.mu.Lock()
	sr.proc = proc
	sr.state.Status = ServiceRunning
	sr.state.PID = proc.PID
	sr.state.StartedAt = &now
	sr.mu.Unlock()
	m.emit(EventServiceUpdated, id, sr.snapshot())

	streamKey := logs.ServiceStreamKey(id)
	go m.pipeLog(id, streamKey, "stdout", proc.Stdout)
	go m.pipeLog(id, streamKey, "stderr", proc.Stderr)
	go m.watchExit(sr, proc)

	m.health.Start(id, cfg.HealthChecks)
	m.startAutoWorkers(ctx, id, cfg)

	return nil
}

// startAutoWorkers starts every worker configured with autoStart: true
// (e.g. Sidekiq alongside its Rails server). Failures are logged, not
// returned — one misconfigured worker shouldn't fail the service start that
// already succeeded.
func (m *Manager) startAutoWorkers(ctx context.Context, serviceID string, cfg config.ServiceConfig) {
	for _, w := range cfg.Workers {
		if !w.AutoStart {
			continue
		}
		if err := m.StartWorker(ctx, serviceID, w.ID); err != nil && !errors.Is(err, ErrWorkerAlreadyRunning) {
			applog.Error("runtime", "autoStart worker %q for service %q: %v", w.ID, serviceID, err)
		}
	}
}

// stopAutoWorkers stops every currently-running autoStart worker for a
// service — the symmetric half of startAutoWorkers, called once the service
// itself has stopped or crashed (watchExit) so a Sidekiq tied to a Rails
// server doesn't keep running (or looks orphaned) after its parent exits.
func (m *Manager) stopAutoWorkers(serviceID string, cfg config.ServiceConfig) {
	for _, w := range cfg.Workers {
		if !w.AutoStart {
			continue
		}
		if err := m.StopWorker(context.Background(), serviceID, w.ID); err != nil && !errors.Is(err, ErrWorkerNotRunning) {
			applog.Error("runtime", "stop autoStart worker %q for service %q: %v", w.ID, serviceID, err)
		}
	}
}

// StopService signals the running process and waits for it to exit. It
// deliberately does not flip the state to "stopped" itself — watchExit is the
// single place that finalizes exit state, whether the exit was requested
// (this method) or a crash, so there's exactly one writer for that
// transition.
//
// If there's no in-memory process handle (proc == nil) but the service isn't
// "stopped" either, this is an orphan adopted by ReconcileOrphans at startup
// (BETA_PLAN: Setsid'd services survive a devctl backend restart, so a fresh
// process has no handle for something a previous instance started). There's
// nothing to send SIGTERM to in that case, so fall back to killing whatever
// holds the configured port — the same mechanism as the manual force-kill
// escape hatch.
func (m *Manager) StopService(ctx context.Context, id string) error {
	sr, err := m.getRuntime(id)
	if err != nil {
		return err
	}

	sr.mu.Lock()
	if sr.state.Status == ServiceStopped {
		sr.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrServiceNotRunning, id)
	}
	sr.state.Status = ServiceStopping
	proc := sr.proc
	port := sr.config.Port
	sr.mu.Unlock()
	m.emit(EventServiceUpdated, id, sr.snapshot())

	if proc == nil {
		if err := ForceKillPort(port); err != nil {
			sr.mu.Lock()
			sr.state.Status = ServiceFailed
			sr.state.LastError = err.Error()
			sr.mu.Unlock()
			m.emit(EventServiceUpdated, id, sr.snapshot())
			return fmt.Errorf("stop service %q: %w", id, err)
		}
		sr.mu.Lock()
		sr.state.Status = ServiceStopped
		sr.state.PID = 0
		sr.mu.Unlock()
		m.health.Stop(id)
		m.emit(EventServiceUpdated, id, sr.snapshot())
		return nil
	}
	if err := m.runner.Stop(proc, m.stopTimeout); err != nil {
		return fmt.Errorf("stop service %q: %w", id, err)
	}
	return nil
}

// ForceKillService kills whatever is listening on the service's configured
// port regardless of whether this backend instance owns that process — the
// manual escape hatch for when ReconcileOrphans guesses wrong (BETA_PLAN
// option 3).
func (m *Manager) ForceKillService(ctx context.Context, id string) error {
	sr, err := m.getRuntime(id)
	if err != nil {
		return err
	}
	sr.mu.RLock()
	port := sr.config.Port
	sr.mu.RUnlock()

	if err := ForceKillPort(port); err != nil {
		return fmt.Errorf("force-kill service %q: %w", id, err)
	}

	sr.mu.Lock()
	sr.proc = nil
	sr.state.Status = ServiceStopped
	sr.state.PID = 0
	sr.mu.Unlock()
	m.health.Stop(id)
	m.emit(EventServiceUpdated, id, sr.snapshot())
	return nil
}

// RestartService stops the service (if running) and starts it again.
func (m *Manager) RestartService(ctx context.Context, id string) error {
	sr, err := m.getRuntime(id)
	if err != nil {
		return err
	}

	if sr.snapshot().Status != ServiceStopped {
		if err := m.StopService(ctx, id); err != nil {
			return fmt.Errorf("restart service %q: stop failed: %w", id, err)
		}
	}
	if err := m.StartService(ctx, id); err != nil {
		return fmt.Errorf("restart service %q: start failed: %w", id, err)
	}
	return nil
}

// dependsOnMap returns each configured service's DependsOn list, keyed by
// service ID, for topoSortServices.
func (m *Manager) dependsOnMap() map[string][]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string][]string, len(m.services))
	for id, sr := range m.services {
		out[id] = sr.config.DependsOn
	}
	return out
}

// StartPreset starts every service in ids in dependency order (T-051). It
// keeps going on individual failures so one bad service doesn't block the
// rest of the preset, collecting every error encountered. Already-running
// services are skipped silently rather than reported as errors.
func (m *Manager) StartPreset(ctx context.Context, ids []string) []error {
	order, hadCycle := topoSortServices(ids, m.dependsOnMap())
	if hadCycle {
		applog.Warn("runtime", "preset: dependency cycle detected among %v; falling back to config order", ids)
	}

	var errs []error
	for _, id := range order {
		if err := m.StartService(ctx, id); err != nil && !errors.Is(err, ErrServiceAlreadyRunning) {
			errs = append(errs, err)
		}
	}
	return errs
}

// StopPreset stops every service in ids in reverse dependency order.
func (m *Manager) StopPreset(ctx context.Context, ids []string) []error {
	order, hadCycle := topoSortServices(ids, m.dependsOnMap())
	if hadCycle {
		applog.Warn("runtime", "preset: dependency cycle detected among %v; falling back to config order", ids)
	}

	var errs []error
	for i := len(order) - 1; i >= 0; i-- {
		id := order[i]
		if err := m.StopService(ctx, id); err != nil && !errors.Is(err, ErrServiceNotRunning) {
			errs = append(errs, err)
		}
	}
	return errs
}

// StopAll stops every currently running worker and service. This is the
// deliberate "Stop All" UI action (GAMMA_PLAN.md T-080 decision) — not an
// implicit stop-on-exit default, which would fight ReconcileOrphans'
// purpose of letting services survive a devctl backend restart. Workers are
// stopped first (autoStart ones would otherwise race with their parent
// service's own stopAutoWorkers call): independently-controlled workers
// aren't tied to their service's lifecycle, so StopService alone wouldn't
// reach them.
func (m *Manager) StopAll(ctx context.Context) []error {
	m.mu.RLock()
	ids := append([]string(nil), m.order...)
	m.mu.RUnlock()

	var errs []error
	for _, id := range ids {
		sr, err := m.getRuntime(id)
		if err != nil {
			continue
		}
		for _, w := range sr.snapshot().Workers {
			if w.Status == WorkerStopped {
				continue
			}
			if err := m.StopWorker(ctx, id, w.ID); err != nil && !errors.Is(err, ErrWorkerNotRunning) {
				errs = append(errs, err)
			}
		}
	}

	errs = append(errs, m.StopPreset(ctx, ids)...)
	return errs
}

func (m *Manager) pipeLog(serviceID, streamKey, source string, r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		entry := m.logs.Append(streamKey, source, scanner.Text(), time.Now())
		m.emit(EventLogAppended, serviceID, LogAppendedPayload{Entry: entry})
	}
}

// watchExit is the single writer for the "process has exited" transition.
// It ignores exits from a superseded generation (e.g. a rapid restart start
// a new process before this one's exit was observed).
func (m *Manager) watchExit(sr *serviceRuntime, proc *RunningProcess) {
	<-proc.Done()

	sr.mu.Lock()
	if sr.proc != proc {
		sr.mu.Unlock()
		return
	}
	exitCode := proc.ExitCode()
	wasStopping := sr.state.Status == ServiceStopping
	sr.state.PID = 0
	sr.state.LastExitCode = &exitCode
	if wasStopping || exitCode == 0 {
		sr.state.Status = ServiceStopped
	} else {
		sr.state.Status = ServiceFailed
		if err := proc.ExitErr(); err != nil {
			sr.state.LastError = err.Error()
		}
	}
	id := sr.config.ID
	cfg := sr.config
	sr.mu.Unlock()
	m.health.Stop(id)
	m.emit(EventServiceUpdated, id, sr.snapshot())
	go m.stopAutoWorkers(id, cfg)
}

// SetServiceHealth records a new health status for id and emits
// health.updated. Called back by the HealthMonitor implementation whenever a
// probe result changes.
func (m *Manager) SetServiceHealth(id string, status string) {
	sr, err := m.getRuntime(id)
	if err != nil {
		return
	}
	sr.mu.Lock()
	sr.state.Health.Status = status
	health := sr.state.Health
	sr.mu.Unlock()
	m.emit(EventHealthUpdated, id, HealthUpdatedPayload{Health: health})
}

// RefreshGitState re-reads git status for id's repo path and stores it,
// emitting health.updated's sibling event git.updated. Called once per
// service at startup and explicitly after any git action completes (T-058) —
// never polled.
func (m *Manager) RefreshGitState(ctx context.Context, id string) (GitState, error) {
	sr, err := m.getRuntime(id)
	if err != nil {
		return GitState{}, err
	}
	sr.mu.RLock()
	path := sr.config.Path
	sr.mu.RUnlock()

	state, err := m.git.Status(ctx, path)
	if err != nil {
		return GitState{}, fmt.Errorf("refresh git state %q: %w", id, err)
	}

	sr.mu.Lock()
	sr.state.Git = state
	sr.mu.Unlock()
	m.emit(EventGitUpdated, id, GitUpdatedPayload{Git: state})
	return state, nil
}

// RefreshAllGitStates refreshes git status for every configured service
// concurrently. Intended for a one-time call at startup (T-058 "on service
// load"); errors are logged, not returned, since one repo's git failure
// shouldn't block the rest.
func (m *Manager) RefreshAllGitStates(ctx context.Context) {
	m.mu.RLock()
	ids := append([]string(nil), m.order...)
	m.mu.RUnlock()

	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if _, err := m.RefreshGitState(ctx, id); err != nil {
				applog.Error("runtime", "git: initial refresh for %q: %v", id, err)
			}
		}(id)
	}
	wg.Wait()
}

// StartWorker starts the configured process for a worker attached to a
// service. Workers are independently controllable from their parent service
// (ARCHITECTURE.md §12.2).
func (m *Manager) StartWorker(ctx context.Context, serviceID, workerID string) error {
	sr, err := m.getRuntime(serviceID)
	if err != nil {
		return err
	}
	wr, ok := sr.getWorker(workerID)
	if !ok {
		return fmt.Errorf("%w: %s/%s", ErrWorkerNotFound, serviceID, workerID)
	}

	wr.mu.Lock()
	if wr.state.Status == WorkerRunning || wr.state.Status == WorkerStarting {
		wr.mu.Unlock()
		return fmt.Errorf("%w: %s/%s", ErrWorkerAlreadyRunning, serviceID, workerID)
	}
	wr.state.Status = WorkerStarting
	wr.state.LastError = ""
	wr.state.LastExitCode = nil
	cfg := wr.config
	wr.mu.Unlock()
	m.emitWorker(serviceID, wr)

	sr.mu.RLock()
	svcPath := sr.config.Path
	sr.mu.RUnlock()

	proc, err := m.runner.Start(ctx, ProcessOptions{
		ID:      serviceID + ":" + cfg.ID,
		Command: cfg.StartCommand,
		Dir:     svcPath,
		Env:     cfg.Env,
	})
	if err != nil {
		wr.mu.Lock()
		wr.state.Status = WorkerFailed
		wr.state.LastError = err.Error()
		wr.mu.Unlock()
		m.emitWorker(serviceID, wr)
		return fmt.Errorf("start worker %q: %w", cfg.ID, err)
	}

	wr.mu.Lock()
	wr.proc = proc
	wr.state.Status = WorkerRunning
	wr.state.PID = proc.PID
	wr.mu.Unlock()
	m.emitWorker(serviceID, wr)

	streamKey := logs.WorkerStreamKey(serviceID, workerID)
	go m.pipeLog(serviceID, streamKey, "stdout", proc.Stdout)
	go m.pipeLog(serviceID, streamKey, "stderr", proc.Stderr)
	go m.watchWorkerExit(serviceID, wr, proc)

	return nil
}

// StopWorker signals a running worker process and waits for it to exit.
func (m *Manager) StopWorker(ctx context.Context, serviceID, workerID string) error {
	sr, err := m.getRuntime(serviceID)
	if err != nil {
		return err
	}
	wr, ok := sr.getWorker(workerID)
	if !ok {
		return fmt.Errorf("%w: %s/%s", ErrWorkerNotFound, serviceID, workerID)
	}

	wr.mu.Lock()
	if wr.state.Status == WorkerStopped {
		wr.mu.Unlock()
		return fmt.Errorf("%w: %s/%s", ErrWorkerNotRunning, serviceID, workerID)
	}
	wr.state.Status = WorkerStopping
	proc := wr.proc
	wr.mu.Unlock()
	m.emitWorker(serviceID, wr)

	if proc == nil {
		return nil
	}
	if err := m.runner.Stop(proc, m.stopTimeout); err != nil {
		return fmt.Errorf("stop worker %q: %w", workerID, err)
	}
	return nil
}

// watchWorkerExit is the single writer for a worker's "process has exited"
// transition, mirroring watchExit for services: a deliberate stop (SIGTERM)
// is reported as stopped even though the OS reports a non-zero/signal exit
// code, since that's expected there, not a crash.
func (m *Manager) watchWorkerExit(serviceID string, wr *workerRuntime, proc *RunningProcess) {
	<-proc.Done()

	wr.mu.Lock()
	if wr.proc != proc {
		wr.mu.Unlock()
		return
	}
	exitCode := proc.ExitCode()
	wasStopping := wr.state.Status == WorkerStopping
	wr.state.PID = 0
	wr.state.LastExitCode = &exitCode
	if wasStopping || exitCode == 0 {
		wr.state.Status = WorkerStopped
	} else {
		wr.state.Status = WorkerFailed
		if err := proc.ExitErr(); err != nil {
			wr.state.LastError = err.Error()
		}
	}
	wr.mu.Unlock()
	m.emitWorker(serviceID, wr)
}

func (m *Manager) emitWorker(serviceID string, wr *workerRuntime) {
	m.emit(EventWorkerUpdated, serviceID, WorkerUpdatedPayload{Worker: wr.snapshot()})
}
