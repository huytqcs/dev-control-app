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
