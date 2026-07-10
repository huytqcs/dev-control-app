package config

import "testing"

// A service that omits openUrls/dependsOn in YAML must still marshal those
// fields as [] rather than null — the frontend calls .length on them
// unconditionally (caught via manual browser smoke test, see ALPHA_PLAN.md).
func TestApplyDefaults_NilSlicesBecomeEmpty(t *testing.T) {
	cfg := &WorkspaceConfig{
		Name: "test",
		Services: []ServiceConfig{
			{ID: "svc", StartCommand: []string{"x"}},
		},
		Presets: []PresetConfig{
			{ID: "preset"},
		},
	}

	applyDefaults(cfg)

	svc := cfg.Services[0]
	if svc.OpenURLs == nil {
		t.Error("expected OpenURLs to be non-nil after applyDefaults")
	}
	if svc.DependsOn == nil {
		t.Error("expected DependsOn to be non-nil after applyDefaults")
	}
	if cfg.Presets[0].Services == nil {
		t.Error("expected preset Services to be non-nil after applyDefaults")
	}
}
