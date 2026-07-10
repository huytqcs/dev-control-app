package runtime

import (
	"sync"

	"devctl/internal/config"
)

// serviceRuntime is the in-memory instance of one configured service: its
// static config, current state, and (if running) its process handle.
type serviceRuntime struct {
	mu     sync.RWMutex
	config config.ServiceConfig
	state  ServiceState
	proc   *RunningProcess
}

func newServiceRuntime(cfg config.ServiceConfig) *serviceRuntime {
	return &serviceRuntime{
		config: cfg,
		state: ServiceState{
			ID:      cfg.ID,
			Name:    cfg.Name,
			Status:  ServiceStopped,
			Port:    cfg.Port,
			Git:     GitState{},
			Health:  HealthState{Status: "unknown"},
			Workers: []WorkerState{},
		},
	}
}

func (sr *serviceRuntime) snapshot() ServiceState {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.state
}
