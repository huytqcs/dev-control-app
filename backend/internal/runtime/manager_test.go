package runtime

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"devctl/internal/config"
	"devctl/internal/logs"
	"devctl/internal/workspace"
)

func testManager(t *testing.T, services ...config.ServiceConfig) (*Manager, *logs.Manager) {
	t.Helper()
	ws := workspace.New(&config.WorkspaceConfig{Name: "test", Services: services})
	logMgr := logs.NewManager()
	m := NewManager(ws, NewOSProcessRunner(), logMgr, NoopPublisher{})
	m.stopTimeout = 2 * time.Second
	return m, logMgr
}

func tickingService(id string) config.ServiceConfig {
	return config.ServiceConfig{
		ID:           id,
		Name:         id,
		Path:         os.TempDir(),
		StartCommand: []string{"sh", "-c", `trap 'exit 0' TERM; while true; do echo tick; sleep 0.05; done`},
	}
}

func tickingServiceWithAutoWorker(id, workerID string) config.ServiceConfig {
	svc := tickingService(id)
	svc.Workers = []config.WorkerConfig{
		{
			ID:           workerID,
			Name:         workerID,
			StartCommand: []string{"sh", "-c", `trap 'exit 0' TERM; while true; do echo tick; sleep 0.05; done`},
			AutoStart:    true,
		},
	}
	return svc
}

func crashingService(id string, code int) config.ServiceConfig {
	return config.ServiceConfig{
		ID:           id,
		Name:         id,
		Path:         os.TempDir(),
		StartCommand: []string{"sh", "-c", "exit " + itoa(code)},
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

func pollUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !cond() {
		t.Fatalf("condition not met within %s", timeout)
	}
}

func TestManager_StartStopService(t *testing.T) {
	m, logMgr := testManager(t, tickingService("svc"))
	ctx := context.Background()

	if err := m.StartService(ctx, "svc"); err != nil {
		t.Fatalf("StartService: %v", err)
	}

	pollUntil(t, 2*time.Second, func() bool {
		s, _ := m.GetState("svc")
		return s.Status == ServiceRunning && s.PID != 0
	})

	pollUntil(t, 2*time.Second, func() bool {
		entries := logMgr.Recent(logs.ServiceStreamKey("svc"), 0)
		for _, e := range entries {
			if strings.Contains(e.Line, "tick") {
				return true
			}
		}
		return false
	})

	if err := m.StopService(ctx, "svc"); err != nil {
		t.Fatalf("StopService: %v", err)
	}

	pollUntil(t, 3*time.Second, func() bool {
		s, _ := m.GetState("svc")
		return s.Status == ServiceStopped && s.PID == 0
	})
}

// delayedRunner wraps the real OS runner but sleeps after the process is
// genuinely alive and before returning it to the caller — modeling the real
// window (however brief in practice) between a process existing and
// StartService recording its handle in sr.proc.
type delayedRunner struct {
	*OSProcessRunner
	startDelay time.Duration
}

func (r *delayedRunner) Start(ctx context.Context, opts ProcessOptions) (*RunningProcess, error) {
	proc, err := r.OSProcessRunner.Start(ctx, opts)
	if err != nil {
		return nil, err
	}
	time.Sleep(r.startDelay)
	return proc, nil
}

// TestManager_StopDuringStart_DoesNotOrphanProcess guards against a real
// bug: a quick double-click of a service's Start/Stop button (or any other
// back-to-back Start-then-Stop) can land the Stop call while the service is
// still "starting" — sr.proc isn't set yet even though the real process is
// already alive. StopService used to treat a nil proc as "orphan adopted
// from a previous devctl instance" and fall back to ForceKillPort, which
// finds nothing yet (the new process may not be listening on its port
// yet either) and reports success. Status flips to "stopped" — but the
// in-flight StartService call then completes moments later and overwrites
// sr.proc/status back to "running", leaving a real process alive and
// untracked as "stopped" (which is exactly what shows up as a health badge
// stuck healthy long after a service was supposedly stopped).
func TestManager_StopDuringStart_DoesNotOrphanProcess(t *testing.T) {
	svc := tickingService("svc")
	// A configured port the ticking shell script never actually binds —
	// this is what makes ForceKillPort's "nothing listening" case return
	// nil (success) instead of an error, which is what lets the race
	// silently flip status to "stopped" instead of surfacing a failure.
	svc.Port = 19998
	ws := workspace.New(&config.WorkspaceConfig{Services: []config.ServiceConfig{svc}})
	logMgr := logs.NewManager()
	m := NewManager(ws, &delayedRunner{OSProcessRunner: NewOSProcessRunner(), startDelay: 300 * time.Millisecond}, logMgr, NoopPublisher{})
	m.stopTimeout = 2 * time.Second
	ctx := context.Background()

	startErrCh := make(chan error, 1)
	go func() { startErrCh <- m.StartService(ctx, "svc") }()

	// Give StartService time to pass its own guard and set status to
	// "starting", but land well before the injected 300ms start delay
	// elapses — this is the window a real double-click races into.
	time.Sleep(50 * time.Millisecond)
	if s, _ := m.GetState("svc"); s.Status != ServiceStarting {
		t.Fatalf("expected status starting when Stop races in, got %v", s.Status)
	}

	stopErr := m.StopService(ctx, "svc")

	if err := <-startErrCh; err != nil {
		t.Fatalf("StartService: %v", err)
	}

	// Whatever StopService decided (reject, or actually wait and stop the
	// real process), the end state must never be a live, untracked process
	// masquerading as anything but genuinely running-and-tracked.
	pollUntil(t, 3*time.Second, func() bool {
		s, _ := m.GetState("svc")
		return s.Status == ServiceStopped || s.Status == ServiceRunning
	})
	final, _ := m.GetState("svc")

	if final.Status == ServiceStopped {
		if final.PID != 0 {
			t.Fatalf("status is stopped but PID is still %d — process wasn't actually reaped", final.PID)
		}
		return
	}

	// If it settled on "running" instead, that's fine too (e.g. Stop was
	// rejected while starting) — but then the caller must have been told
	// the stop didn't happen, not silently lied to.
	if stopErr == nil {
		t.Fatalf("service ended up running but StopService reported success — caller has no idea the stop didn't take effect")
	}
	if !errors.Is(stopErr, ErrServiceStarting) {
		t.Fatalf("expected ErrServiceStarting, got %v", stopErr)
	}
}

func stubbornService(id string) config.ServiceConfig {
	return config.ServiceConfig{
		ID:           id,
		Name:         id,
		Path:         os.TempDir(),
		StartCommand: []string{"sh", "-c", `trap '' TERM; while true; do echo tick; sleep 0.05; done`},
	}
}

// TestManager_StartDuringStop_Rejected guards the symmetric case of
// TestManager_StopDuringStart_DoesNotOrphanProcess: clicking Start again
// while a previous Stop is still in its SIGTERM-grace teardown (a real
// double-click scenario) must not be allowed to race a second process onto
// the same port while the first is still dying.
func TestManager_StartDuringStop_Rejected(t *testing.T) {
	m, _ := testManager(t, stubbornService("svc"))
	ctx := context.Background()

	if err := m.StartService(ctx, "svc"); err != nil {
		t.Fatalf("StartService: %v", err)
	}
	pollUntil(t, 2*time.Second, func() bool {
		s, _ := m.GetState("svc")
		return s.Status == ServiceRunning && s.PID != 0
	})
	// Give the shell time to actually execute its `trap '' TERM` builtin —
	// "running" flips true the instant the process is forked/exec'd, which
	// can race ahead of the shell reaching its first statement. Without
	// this, SIGTERM can arrive before the trap is registered and the
	// default disposition (terminate) applies instead, killing it
	// immediately — a test-only race, not the thing under test here.
	time.Sleep(200 * time.Millisecond)

	stopErrCh := make(chan error, 1)
	go func() { stopErrCh <- m.StopService(ctx, "svc") }()

	// The stubborn process ignores SIGTERM, so it sits in "stopping" for the
	// full 2s stopTimeout before SIGKILL — land a Start well inside that
	// window, same as a real quick double-click would.
	time.Sleep(100 * time.Millisecond)
	if s, _ := m.GetState("svc"); s.Status != ServiceStopping {
		t.Fatalf("expected status stopping when Start races in, got %v (pid=%d lastErr=%q lastExit=%v)", s.Status, s.PID, s.LastError, s.LastExitCode)
	}

	startErr := m.StartService(ctx, "svc")
	if !errors.Is(startErr, ErrServiceStopping) {
		t.Fatalf("expected ErrServiceStopping, got %v", startErr)
	}

	if err := <-stopErrCh; err != nil {
		t.Fatalf("StopService: %v", err)
	}
	pollUntil(t, 3*time.Second, func() bool {
		s, _ := m.GetState("svc")
		return s.Status == ServiceStopped && s.PID == 0
	})
}

func TestManager_StartUnknownServiceFails(t *testing.T) {
	m, _ := testManager(t)
	err := m.StartService(context.Background(), "ghost")
	if !errors.Is(err, ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", err)
	}
}

func TestManager_StartTwiceReturnsAlreadyRunning(t *testing.T) {
	m, _ := testManager(t, tickingService("svc"))
	ctx := context.Background()

	if err := m.StartService(ctx, "svc"); err != nil {
		t.Fatalf("StartService: %v", err)
	}
	t.Cleanup(func() { _ = m.StopService(ctx, "svc") })

	err := m.StartService(ctx, "svc")
	if !errors.Is(err, ErrServiceAlreadyRunning) {
		t.Fatalf("expected ErrServiceAlreadyRunning, got %v", err)
	}
}

func TestManager_StopWhenNotRunningFails(t *testing.T) {
	m, _ := testManager(t, tickingService("svc"))
	err := m.StopService(context.Background(), "svc")
	if !errors.Is(err, ErrServiceNotRunning) {
		t.Fatalf("expected ErrServiceNotRunning, got %v", err)
	}
}

func TestManager_CrashingProcessMarkedFailed(t *testing.T) {
	m, _ := testManager(t, crashingService("svc", 7))
	ctx := context.Background()

	if err := m.StartService(ctx, "svc"); err != nil {
		t.Fatalf("StartService: %v", err)
	}

	pollUntil(t, 2*time.Second, func() bool {
		s, _ := m.GetState("svc")
		return s.Status == ServiceFailed
	})

	s, _ := m.GetState("svc")
	if s.LastExitCode == nil || *s.LastExitCode != 7 {
		t.Fatalf("expected exit code 7, got %v", s.LastExitCode)
	}
}

func TestManager_RestartService(t *testing.T) {
	m, _ := testManager(t, tickingService("svc"))
	ctx := context.Background()

	if err := m.StartService(ctx, "svc"); err != nil {
		t.Fatalf("StartService: %v", err)
	}
	pollUntil(t, 2*time.Second, func() bool {
		s, _ := m.GetState("svc")
		return s.Status == ServiceRunning
	})
	first, _ := m.GetState("svc")

	if err := m.RestartService(ctx, "svc"); err != nil {
		t.Fatalf("RestartService: %v", err)
	}
	pollUntil(t, 3*time.Second, func() bool {
		s, _ := m.GetState("svc")
		return s.Status == ServiceRunning && s.PID != first.PID
	})

	t.Cleanup(func() { _ = m.StopService(ctx, "svc") })
}

func TestManager_AutoStartWorker_StartsAndStopsWithService(t *testing.T) {
	m, _ := testManager(t, tickingServiceWithAutoWorker("svc", "worker"))
	ctx := context.Background()

	if err := m.StartService(ctx, "svc"); err != nil {
		t.Fatalf("StartService: %v", err)
	}

	pollUntil(t, 2*time.Second, func() bool {
		s, _ := m.GetState("svc")
		for _, w := range s.Workers {
			if w.ID == "worker" && w.Status == WorkerRunning {
				return true
			}
		}
		return false
	})

	if err := m.StopService(ctx, "svc"); err != nil {
		t.Fatalf("StopService: %v", err)
	}

	pollUntil(t, 3*time.Second, func() bool {
		s, _ := m.GetState("svc")
		if s.Status != ServiceStopped {
			return false
		}
		for _, w := range s.Workers {
			if w.ID == "worker" && w.Status != WorkerStopped {
				return false
			}
		}
		return true
	})
}
