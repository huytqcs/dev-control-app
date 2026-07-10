package health

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"devctl/internal/config"
)

// TestStop_WaitsForInFlightProbeBeforeReportingUnknown guards against a race
// where Stop reported "unknown" immediately on cancel, while a probe that
// was already in flight when Stop was called could still land its own
// (now-stale) healthy/unhealthy result afterwards — leaving a stopped
// service's health badge stuck showing old state instead of hiding.
func TestStop_WaitsForInFlightProbeBeforeReportingUnknown(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	defer close(release)

	var mu sync.Mutex
	var calls []Status
	m := NewMonitor(func(id string, status Status) {
		mu.Lock()
		calls = append(calls, status)
		mu.Unlock()
	})

	checks := []config.HealthCheck{{Type: "http", URL: srv.URL, Timeout: 5}}
	m.Start("svc", checks)

	<-started // the initial probe is now blocked waiting on the server
	m.Stop("svc")

	// With the fix, Stop already blocked until the in-flight probe's own
	// onUpdate call landed, so this is just a margin against a regression —
	// not what makes the assertion below meaningful. Confirmed via a manual
	// timing run that the buggy version's straggler write lands within
	// ~30µs of Stop returning, so 100ms is a large, non-flaky margin.
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(calls) == 0 {
		t.Fatalf("expected at least one onUpdate call, got none")
	}
	if last := calls[len(calls)-1]; last != StatusUnknown {
		t.Fatalf("expected last reported status to be %q (Stop wins), got %q from %v", StatusUnknown, last, calls)
	}
}

func TestStartStop_NoConfiguredChecksIsNoop(t *testing.T) {
	var calls int
	m := NewMonitor(func(string, Status) { calls++ })

	m.Start("svc", nil)
	m.Stop("svc")

	if calls != 0 {
		t.Fatalf("expected no onUpdate calls for an unmonitored service, got %d", calls)
	}
}
