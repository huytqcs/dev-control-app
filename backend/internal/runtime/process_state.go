package runtime

import "time"

type ServiceStatus string

const (
	ServiceStopped  ServiceStatus = "stopped"
	ServiceStarting ServiceStatus = "starting"
	ServiceRunning  ServiceStatus = "running"
	ServiceFailed   ServiceStatus = "failed"
	ServiceStopping ServiceStatus = "stopping"
)

type WorkerStatus string

const (
	WorkerStopped  WorkerStatus = "stopped"
	WorkerStarting WorkerStatus = "starting"
	WorkerRunning  WorkerStatus = "running"
	WorkerFailed   WorkerStatus = "failed"
)

type WorkerState struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Status       WorkerStatus `json:"status"`
	PID          int          `json:"pid,omitempty"`
	LastError    string       `json:"lastError,omitempty"`
	LastExitCode *int         `json:"lastExitCode,omitempty"`
}

// GitState and HealthState are alpha stubs. The git/health modules land in
// beta and populate these for real; until then every ServiceState reports an
// empty branch and "unknown" health so the API response shape doesn't change
// out from under the frontend later.
type GitState struct {
	Branch string `json:"branch"`
	Dirty  bool   `json:"dirty"`
}

type HealthState struct {
	Status string `json:"status"`
}

type ServiceState struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Status       ServiceStatus `json:"status"`
	PID          int           `json:"pid,omitempty"`
	Port         int           `json:"port,omitempty"`
	StartedAt    *time.Time    `json:"startedAt,omitempty"`
	LastError    string        `json:"lastError,omitempty"`
	LastExitCode *int          `json:"lastExitCode,omitempty"`
	Git          GitState      `json:"git"`
	Health       HealthState   `json:"health"`
	Workers      []WorkerState `json:"workers"`
}
