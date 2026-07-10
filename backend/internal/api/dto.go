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

type GitStateDTO struct {
	Branch string `json:"branch"`
	Dirty  bool   `json:"dirty"`
	Ahead  int    `json:"ahead"`
	Behind int    `json:"behind"`
}

type HealthStateDTO struct {
	Status string `json:"status"`
}

type ActionSummaryDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WorkerSummaryDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	PID          int    `json:"pid,omitempty"`
	LastError    string `json:"lastError,omitempty"`
	LastExitCode *int   `json:"lastExitCode,omitempty"`
	AutoStart    bool   `json:"autoStart"`
}

func toServiceDTO(cfg config.ServiceConfig, state runtime.ServiceState) ServiceDTO {
	actions := make([]ActionSummaryDTO, 0, len(cfg.Actions))
	for _, a := range cfg.Actions {
		actions = append(actions, ActionSummaryDTO{ID: a.ID, Name: a.Name})
	}

	autoStartByID := make(map[string]bool, len(cfg.Workers))
	for _, w := range cfg.Workers {
		autoStartByID[w.ID] = w.AutoStart
	}

	workers := make([]WorkerSummaryDTO, 0, len(state.Workers))
	for _, w := range state.Workers {
		workers = append(workers, WorkerSummaryDTO{
			ID:           w.ID,
			Name:         w.Name,
			Status:       string(w.Status),
			PID:          w.PID,
			LastError:    w.LastError,
			LastExitCode: w.LastExitCode,
			AutoStart:    autoStartByID[w.ID],
		})
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
			Git: GitStateDTO{
				Branch: state.Git.Branch,
				Dirty:  state.Git.Dirty,
				Ahead:  state.Git.Ahead,
				Behind: state.Git.Behind,
			},
			Health: HealthStateDTO{Status: state.Health.Status},
		},
		Actions: actions,
		Workers: workers,
	}
}
