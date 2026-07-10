package actions

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"devctl/internal/config"
	"devctl/internal/logs"
	"devctl/internal/runtime"
)

// fakePublisher is a no-op-except-recording runtime.EventPublisher, mirroring
// the polling-based async assertion style used in
// internal/runtime/manager_test.go (pollUntil), since our completion event is
// also delivered asynchronously from a goroutine.
type fakePublisher struct {
	mu     sync.Mutex
	events []runtime.AppEvent
}

func (p *fakePublisher) Publish(evt runtime.AppEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, evt)
}

func (p *fakePublisher) snapshot() []runtime.AppEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]runtime.AppEvent(nil), p.events...)
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

func findCompleted(events []runtime.AppEvent, runID string) (runtime.ActionCompletedPayload, bool) {
	for _, e := range events {
		if e.Type != runtime.EventActionCompleted {
			continue
		}
		p, ok := e.Payload.(runtime.ActionCompletedPayload)
		if ok && p.RunID == runID {
			return p, true
		}
	}
	return runtime.ActionCompletedPayload{}, false
}

func TestService_Run_Success(t *testing.T) {
	logMgr := logs.NewManager()
	pub := &fakePublisher{}
	s := NewService(logMgr, pub)

	action := config.ActionConfig{ID: "greet", Name: "greet", Command: []string{"echo", "hello-from-action"}}

	runID, err := s.Run(context.Background(), "svc", action, os.TempDir())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if runID == "" {
		t.Fatalf("expected non-empty run ID")
	}

	streamKey := logs.ActionStreamKey("svc", "greet", runID)
	pollUntil(t, 2*time.Second, func() bool {
		entries := logMgr.Recent(streamKey, 0)
		for _, e := range entries {
			if strings.Contains(e.Line, "hello-from-action") {
				return true
			}
		}
		return false
	})

	pollUntil(t, 2*time.Second, func() bool {
		p, ok := findCompleted(pub.snapshot(), runID)
		return ok && p.Success && p.ExitCode == 0
	})

	p, _ := findCompleted(pub.snapshot(), runID)
	if p.ActionID != "greet" {
		t.Fatalf("expected ActionID %q, got %q", "greet", p.ActionID)
	}
	if p.Error != "" {
		t.Fatalf("expected empty Error on success, got %q", p.Error)
	}
}

func TestService_Run_Failure(t *testing.T) {
	logMgr := logs.NewManager()
	pub := &fakePublisher{}
	s := NewService(logMgr, pub)

	action := config.ActionConfig{ID: "fail", Name: "fail", Command: []string{"sh", "-c", "exit 3"}}

	runID, err := s.Run(context.Background(), "svc", action, os.TempDir())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	pollUntil(t, 2*time.Second, func() bool {
		p, ok := findCompleted(pub.snapshot(), runID)
		return ok && !p.Success && p.ExitCode != 0
	})

	p, _ := findCompleted(pub.snapshot(), runID)
	if p.ExitCode != 3 {
		t.Fatalf("expected exit code 3, got %d", p.ExitCode)
	}
}

func TestService_Run_GuardsAgainstConcurrentRunsForSamePair(t *testing.T) {
	logMgr := logs.NewManager()
	pub := &fakePublisher{}
	s := NewService(logMgr, pub)

	action := config.ActionConfig{ID: "slow", Name: "slow", Command: []string{"sleep", "0.3"}}

	runID1, err := s.Run(context.Background(), "svc", action, os.TempDir())
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}

	_, err = s.Run(context.Background(), "svc", action, os.TempDir())
	if !errors.Is(err, ErrActionAlreadyRunning) {
		t.Fatalf("expected ErrActionAlreadyRunning, got %v", err)
	}

	pollUntil(t, 2*time.Second, func() bool {
		_, ok := findCompleted(pub.snapshot(), runID1)
		return ok
	})

	// A third call for the same pair, after the first has completed, must
	// succeed again.
	runID3, err := s.Run(context.Background(), "svc", action, os.TempDir())
	if err != nil {
		t.Fatalf("third Run after completion: %v", err)
	}
	if runID3 == runID1 {
		t.Fatalf("expected a fresh run ID, got the same one: %s", runID3)
	}

	pollUntil(t, 2*time.Second, func() bool {
		_, ok := findCompleted(pub.snapshot(), runID3)
		return ok
	})
}

// TestService_Run_SurvivesCallerContextCancellation guards against a real
// bug caught by manual integration testing: Run is called with an HTTP
// request's context (it returns a run ID immediately while the process
// keeps running in the background), and net/http cancels that context the
// moment the handler returns — almost immediately, long before a
// several-hundred-millisecond action finishes. If Run ties the actual OS
// process to that context (e.g. via exec.CommandContext(ctx, ...) instead of
// a plain exec.Command), the process gets SIGKILLed right after starting:
// no output is ever produced and it "completes" with exitCode -1 / "signal:
// killed" instead of running to its real, deliberate exit code.
func TestService_Run_SurvivesCallerContextCancellation(t *testing.T) {
	logMgr := logs.NewManager()
	pub := &fakePublisher{}
	s := NewService(logMgr, pub)

	action := config.ActionConfig{
		ID:      "slow-echo",
		Name:    "slow-echo",
		Command: []string{"sh", "-c", "sleep 0.3; echo still-alive; exit 7"},
	}

	callerCtx, cancel := context.WithCancel(context.Background())
	runID, err := s.Run(callerCtx, "svc", action, os.TempDir())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Simulate the HTTP handler returning right after Run hands back a run
	// ID — this is exactly what net/http itself does to a request context.
	cancel()

	pollUntil(t, 2*time.Second, func() bool {
		p, ok := findCompleted(pub.snapshot(), runID)
		return ok && p.ExitCode == 7
	})

	p, _ := findCompleted(pub.snapshot(), runID)
	if strings.Contains(p.Error, "killed") {
		t.Fatalf("action was killed instead of running to its real exit code: Error=%q", p.Error)
	}

	streamKey := logs.ActionStreamKey("svc", "slow-echo", runID)
	entries := logMgr.Recent(streamKey, 0)
	found := false
	for _, e := range entries {
		if strings.Contains(e.Line, "still-alive") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected output produced after the caller's context was cancelled, got entries: %+v", entries)
	}
}

func TestService_Run_DifferentActionsSameServiceRunConcurrently(t *testing.T) {
	logMgr := logs.NewManager()
	pub := &fakePublisher{}
	s := NewService(logMgr, pub)

	actionA := config.ActionConfig{ID: "a", Name: "a", Command: []string{"sleep", "0.2"}}
	actionB := config.ActionConfig{ID: "b", Name: "b", Command: []string{"sleep", "0.2"}}

	if _, err := s.Run(context.Background(), "svc", actionA, os.TempDir()); err != nil {
		t.Fatalf("Run a: %v", err)
	}
	if _, err := s.Run(context.Background(), "svc", actionB, os.TempDir()); err != nil {
		t.Fatalf("Run b (different action, should not be guarded): %v", err)
	}
}
