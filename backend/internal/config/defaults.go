package config

// DefaultConfigPath is used when --config is not passed.
const DefaultConfigPath = "~/.config/devctl/devctl.yaml"

// applyDefaults fills in zero-value fields with sane defaults. Called after
// parsing, before validation.
func applyDefaults(cfg *WorkspaceConfig) {
	for i := range cfg.Services {
		svc := &cfg.Services[i]
		if svc.Env == nil {
			svc.Env = map[string]string{}
		}
		// Nil slices marshal to JSON null, not []; the frontend always
		// expects an array here (e.g. service.dependsOn.length).
		if svc.OpenURLs == nil {
			svc.OpenURLs = []string{}
		}
		if svc.DependsOn == nil {
			svc.DependsOn = []string{}
		}
		for j := range svc.Workers {
			if svc.Workers[j].Env == nil {
				svc.Workers[j].Env = map[string]string{}
			}
		}
		for j := range svc.Actions {
			if svc.Actions[j].Env == nil {
				svc.Actions[j].Env = map[string]string{}
			}
		}
	}

	for i := range cfg.Presets {
		if cfg.Presets[i].Services == nil {
			cfg.Presets[i].Services = []string{}
		}
	}
}
