package runtime

import (
	"sync"

	"devctl/internal/config"
)

// workerRuntime is the in-memory instance of one configured worker: its
// static config, current state, and (if running) its process handle. Mirrors
// serviceRuntime, but workers are independently start/stoppable from their
// parent service (ARCHITECTURE.md §12.2).
type workerRuntime struct {
	mu     sync.RWMutex
	config config.WorkerConfig
	state  WorkerState
	proc   *RunningProcess
}

func newWorkerRuntime(cfg config.WorkerConfig) *workerRuntime {
	return &workerRuntime{
		config: cfg,
		state: WorkerState{
			ID:     cfg.ID,
			Name:   cfg.Name,
			Status: WorkerStopped,
		},
	}
}

func (wr *workerRuntime) snapshot() WorkerState {
	wr.mu.RLock()
	defer wr.mu.RUnlock()
	return wr.state
}
