package health

import (
	"context"
	"sync"
	"time"

	"devctl/internal/config"
)

const defaultInterval = 3 * time.Second

// OnUpdate is called whenever a monitored service's health status changes
// (including the initial check and the transition back to "unknown" on
// Stop). Implemented by runtime.Manager.SetServiceHealth.
type OnUpdate func(serviceID string, status Status)

// Monitor runs one health-check loop per running service and reports
// results via OnUpdate. It implements runtime.HealthMonitor structurally.
// Only running services are ever monitored (ARCHITECTURE.md §22.1).
type monitorHandle struct {
	cancel context.CancelFunc
	done   chan struct{}
}

type Monitor struct {
	onUpdate OnUpdate

	mu       sync.Mutex
	monitors map[string]monitorHandle
}

func NewMonitor(onUpdate OnUpdate) *Monitor {
	return &Monitor{onUpdate: onUpdate, monitors: make(map[string]monitorHandle)}
}

// Start begins monitoring serviceID. A no-op if there are no configured
// checks or the service is already being monitored.
func (m *Monitor) Start(serviceID string, checks []config.HealthCheck) {
	if len(checks) == 0 {
		return
	}

	m.mu.Lock()
	if _, exists := m.monitors[serviceID]; exists {
		m.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	m.monitors[serviceID] = monitorHandle{cancel: cancel, done: done}
	m.mu.Unlock()

	interval := defaultInterval
	if checks[0].Interval > 0 {
		interval = time.Duration(checks[0].Interval) * time.Second
	}

	go m.loop(ctx, serviceID, checks, interval, done)
}

// Stop halts monitoring for serviceID and reports it back to "unknown" —
// health only means anything while the process is running. It waits for the
// loop goroutine to actually exit before reporting "unknown" so an in-flight
// probe that was already running when Stop was called can't race its own
// (now-stale) healthy/unhealthy result in afterwards and leave the badge
// stuck showing it.
func (m *Monitor) Stop(serviceID string) {
	m.mu.Lock()
	handle, exists := m.monitors[serviceID]
	if exists {
		delete(m.monitors, serviceID)
	}
	m.mu.Unlock()

	if !exists {
		return
	}
	handle.cancel()
	<-handle.done
	m.onUpdate(serviceID, StatusUnknown)
}

func (m *Monitor) loop(ctx context.Context, serviceID string, checks []config.HealthCheck, interval time.Duration, done chan struct{}) {
	defer close(done)

	check := func() {
		m.onUpdate(serviceID, Probe(ctx, checks))
	}

	check()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}
