package api

import (
	"time"

	"devctl/internal/config"
	"devctl/internal/runtime"
)

type WorkspaceDTO struct {
	Name    string      `json:"name"`
	Presets []PresetDTO `json:"presets"`
}

type PresetDTO struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Services []string `json:"services"`
}

type ServiceDTO struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Type      string             `json:"type"`
	Path      string             `json:"path"`
	Port      int                `json:"port"`
	OpenURLs  []string           `json:"openUrls"`
	DependsOn []string           `json:"dependsOn"`
	State     ServiceStateDTO    `json:"state"`
	Actions   []ActionSummaryDTO `json:"actions"`
	Workers   []WorkerSummaryDTO `json:"workers"`
}

type ServiceStateDTO struct {
	Status       string         `json:"status"`
	PID          int            `json:"pid,omitempty"`
	StartedAt    *time.Time     `json:"startedAt,omitempty"`
	LastError    string         `json:"lastError,omitempty"`
	LastExitCode *int           `json:"lastExitCode,omitempty"`
	Git          GitStateDTO    `json:"git"`
	Health       HealthStateDTO `json:"health"`
}

// GitStateDTO/HealthStateDTO are alpha stubs: the git/health backend modules
// don't exist yet (beta), so these always report an empty branch and
// "unknown" health. Keeping the fields on the wire now means beta's modules
// just start populating real data without changing the response shape.
type GitStateDTO struct {
	Branch string `json:"branch"`
	Dirty  bool   `json:"dirty"`
}

type HealthStateDTO struct {
	Status string `json:"status"`
}

type ActionSummaryDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WorkerSummaryDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func toServiceDTO(cfg config.ServiceConfig, state runtime.ServiceState) ServiceDTO {
	actions := make([]ActionSummaryDTO, 0, len(cfg.Actions))
	for _, a := range cfg.Actions {
		actions = append(actions, ActionSummaryDTO{ID: a.ID, Name: a.Name})
	}

	// No worker runtime exists in alpha, so every configured worker is
	// reported "stopped" — honest given nothing can start it yet.
	workers := make([]WorkerSummaryDTO, 0, len(cfg.Workers))
	for _, w := range cfg.Workers {
		workers = append(workers, WorkerSummaryDTO{ID: w.ID, Name: w.Name, Status: "stopped"})
	}

	return ServiceDTO{
		ID:        cfg.ID,
		Name:      cfg.Name,
		Type:      cfg.Type,
		Path:      cfg.Path,
		Port:      cfg.Port,
		OpenURLs:  cfg.OpenURLs,
		DependsOn: cfg.DependsOn,
		State: ServiceStateDTO{
			Status:       string(state.Status),
			PID:          state.PID,
			StartedAt:    state.StartedAt,
			LastError:    state.LastError,
			LastExitCode: state.LastExitCode,
			Git:          GitStateDTO{Branch: state.Git.Branch, Dirty: state.Git.Dirty},
			Health:       HealthStateDTO{Status: state.Health.Status},
		},
		Actions: actions,
		Workers: workers,
	}
}
