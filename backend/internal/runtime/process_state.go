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

// GitState is populated by internal/git via the runtime.GitProbe interface.
type GitState struct {
	Branch string `json:"branch"`
	Dirty  bool   `json:"dirty"`
	Ahead  int    `json:"ahead"`
	Behind int    `json:"behind"`
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
