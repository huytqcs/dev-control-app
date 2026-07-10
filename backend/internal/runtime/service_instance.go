package runtime

import (
	"sync"

	"devctl/internal/config"
)

// serviceRuntime is the in-memory instance of one configured service: its
// static config, current state, (if running) its process handle, and its
// workers' own runtime instances.
type serviceRuntime struct {
	mu        sync.RWMutex
	config    config.ServiceConfig
	state     ServiceState
	proc      *RunningProcess
	workerIDs []string
	workers   map[string]*workerRuntime
}

func newServiceRuntime(cfg config.ServiceConfig) *serviceRuntime {
	workerIDs := make([]string, 0, len(cfg.Workers))
	workers := make(map[string]*workerRuntime, len(cfg.Workers))
	for _, w := range cfg.Workers {
		workerIDs = append(workerIDs, w.ID)
		workers[w.ID] = newWorkerRuntime(w)
	}

	return &serviceRuntime{
		config: cfg,
		state: ServiceState{
			ID:     cfg.ID,
			Name:   cfg.Name,
			Status: ServiceStopped,
			Port:   cfg.Port,
			Git:    GitState{},
			Health: HealthState{Status: "unknown"},
		},
		workerIDs: workerIDs,
		workers:   workers,
	}
}

// snapshot returns a point-in-time copy of the service's state, with Workers
// assembled live from each worker's own runtime state rather than stored
// statically on ServiceState.
func (sr *serviceRuntime) snapshot() ServiceState {
	sr.mu.RLock()
	state := sr.state
	sr.mu.RUnlock()

	state.Workers = make([]WorkerState, 0, len(sr.workerIDs))
	for _, id := range sr.workerIDs {
		state.Workers = append(state.Workers, sr.workers[id].snapshot())
	}
	return state
}

func (sr *serviceRuntime) getWorker(workerID string) (*workerRuntime, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	wr, ok := sr.workers[workerID]
	return wr, ok
}
