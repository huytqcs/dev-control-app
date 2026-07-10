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
type Monitor struct {
	onUpdate OnUpdate

	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

func NewMonitor(onUpdate OnUpdate) *Monitor {
	return &Monitor{onUpdate: onUpdate, cancels: make(map[string]context.CancelFunc)}
}

// Start begins monitoring serviceID. A no-op if there are no configured
// checks or the service is already being monitored.
func (m *Monitor) Start(serviceID string, checks []config.HealthCheck) {
	if len(checks) == 0 {
		return
	}

	m.mu.Lock()
	if _, exists := m.cancels[serviceID]; exists {
		m.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancels[serviceID] = cancel
	m.mu.Unlock()

	interval := defaultInterval
	if checks[0].Interval > 0 {
		interval = time.Duration(checks[0].Interval) * time.Second
	}

	go m.loop(ctx, serviceID, checks, interval)
}

// Stop halts monitoring for serviceID and reports it back to "unknown" —
// health only means anything while the process is running.
func (m *Monitor) Stop(serviceID string) {
	m.mu.Lock()
	cancel, exists := m.cancels[serviceID]
	if exists {
		delete(m.cancels, serviceID)
	}
	m.mu.Unlock()

	if !exists {
		return
	}
	cancel()
	m.onUpdate(serviceID, StatusUnknown)
}

func (m *Monitor) loop(ctx context.Context, serviceID string, checks []config.HealthCheck, interval time.Duration) {
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
