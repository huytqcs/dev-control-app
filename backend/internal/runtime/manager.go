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

	"devctl/internal/logs"
	"devctl/internal/workspace"
)

var (
	ErrServiceNotFound       = errors.New("service not found")
	ErrServiceAlreadyRunning = errors.New("service already running")
	ErrServiceNotRunning     = errors.New("service not running")
)

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
		services:    make(map[string]*serviceRuntime),
	}
	for _, svc := range ws.ListServices() {
		m.services[svc.ID] = newServiceRuntime(svc)
		m.order = append(m.order, svc.ID)
	}
	return m
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

	return nil
}

// StopService signals the running process and waits for it to exit. It
// deliberately does not flip the state to "stopped" itself — watchExit is the
// single place that finalizes exit state, whether the exit was requested
// (this method) or a crash, so there's exactly one writer for that
// transition.
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
	sr.mu.Unlock()
	m.emit(EventServiceUpdated, id, sr.snapshot())

	if proc == nil {
		return nil
	}
	if err := m.runner.Stop(proc, m.stopTimeout); err != nil {
		return fmt.Errorf("stop service %q: %w", id, err)
	}
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
	sr.mu.Unlock()
	m.emit(EventServiceUpdated, id, sr.snapshot())
}
