package config

import "testing"

func validConfig() *WorkspaceConfig {
	return &WorkspaceConfig{
		Name: "test",
		Services: []ServiceConfig{
			{ID: "be", StartCommand: []string{"rails", "s"}},
			{ID: "app", StartCommand: []string{"npm", "run", "local"}, DependsOn: []string{"be"}},
		},
		Presets: []PresetConfig{
			{ID: "core", Services: []string{"be", "app"}},
		},
	}
}

func TestValidate_ValidConfigHasNoErrors(t *testing.T) {
	if errs := Validate(validConfig()); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidate_EmptyWorkspaceName(t *testing.T) {
	cfg := validConfig()
	cfg.Name = ""
	assertHasError(t, cfg, "workspace name must not be empty")
}

func TestValidate_DuplicateServiceID(t *testing.T) {
	cfg := validConfig()
	cfg.Services = append(cfg.Services, ServiceConfig{ID: "be", StartCommand: []string{"x"}})
	assertHasError(t, cfg, `duplicate service id "be"`)
}

func TestValidate_DuplicatePresetID(t *testing.T) {
	cfg := validConfig()
	cfg.Presets = append(cfg.Presets, PresetConfig{ID: "core", Services: []string{"be"}})
	assertHasError(t, cfg, `duplicate preset id "core"`)
}

func TestValidate_PresetReferencesMissingService(t *testing.T) {
	cfg := validConfig()
	cfg.Presets[0].Services = append(cfg.Presets[0].Services, "ghost")
	assertHasError(t, cfg, `preset "core" references missing service "ghost"`)
}

func TestValidate_DependsOnMissingService(t *testing.T) {
	cfg := validConfig()
	cfg.Services[1].DependsOn = append(cfg.Services[1].DependsOn, "ghost")
	assertHasError(t, cfg, `service "app": dependsOn references missing service "ghost"`)
}

func TestValidate_EmptyStartCommand(t *testing.T) {
	cfg := validConfig()
	cfg.Services[0].StartCommand = nil
	assertHasError(t, cfg, `service "be": startCommand must not be empty`)
}

func TestValidate_DuplicateWorkerID(t *testing.T) {
	cfg := validConfig()
	cfg.Services[0].Workers = []WorkerConfig{
		{ID: "sidekiq", StartCommand: []string{"sidekiq"}},
		{ID: "sidekiq", StartCommand: []string{"sidekiq"}},
	}
	assertHasError(t, cfg, `service "be": duplicate worker id "sidekiq"`)
}

func TestValidate_EmptyActionCommand(t *testing.T) {
	cfg := validConfig()
	cfg.Services[0].Actions = []ActionConfig{{ID: "migrate"}}
	assertHasError(t, cfg, `service "be" action "migrate": command must not be empty`)
}

func TestValidate_UnsupportedHealthCheckType(t *testing.T) {
	cfg := validConfig()
	cfg.Services[0].HealthChecks = []HealthCheck{{Type: "ping"}}
	assertHasError(t, cfg, `service "be": unsupported health check type "ping"`)
}

func assertHasError(t *testing.T, cfg *WorkspaceConfig, want string) {
	t.Helper()
	errs := Validate(cfg)
	for _, e := range errs {
		if e.Error() == want {
			return
		}
	}
	t.Fatalf("expected error %q, got %v", want, errs)
}
