// Package workspace provides read-only access to config-defined workspace
// data (services, presets). It owns no runtime state — see internal/runtime
// for that.
package workspace

import "devctl/internal/config"

type Service struct {
	cfg *config.WorkspaceConfig
}

func New(cfg *config.WorkspaceConfig) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) GetWorkspace() *config.WorkspaceConfig {
	return s.cfg
}

func (s *Service) ListServices() []config.ServiceConfig {
	return s.cfg.Services
}

func (s *Service) GetService(id string) (config.ServiceConfig, bool) {
	for _, svc := range s.cfg.Services {
		if svc.ID == id {
			return svc, true
		}
	}
	return config.ServiceConfig{}, false
}

func (s *Service) ListPresets() []config.PresetConfig {
	return s.cfg.Presets
}

func (s *Service) GetPreset(id string) (config.PresetConfig, bool) {
	for _, p := range s.cfg.Presets {
		if p.ID == id {
			return p, true
		}
	}
	return config.PresetConfig{}, false
}
