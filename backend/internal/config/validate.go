package config

import "fmt"

// Validate checks a parsed WorkspaceConfig against the rules in SPEC.md §8.2
// and returns every violation found (not just the first), so a user fixing
// their config sees the whole list in one pass.
func Validate(cfg *WorkspaceConfig) []error {
	var errs []error

	if cfg.Name == "" {
		errs = append(errs, fmt.Errorf("workspace name must not be empty"))
	}

	serviceIDs := map[string]bool{}
	for _, svc := range cfg.Services {
		if serviceIDs[svc.ID] {
			errs = append(errs, fmt.Errorf("duplicate service id %q", svc.ID))
		}
		serviceIDs[svc.ID] = true

		if len(svc.StartCommand) == 0 {
			errs = append(errs, fmt.Errorf("service %q: startCommand must not be empty", svc.ID))
		}

		workerIDs := map[string]bool{}
		for _, w := range svc.Workers {
			if workerIDs[w.ID] {
				errs = append(errs, fmt.Errorf("service %q: duplicate worker id %q", svc.ID, w.ID))
			}
			workerIDs[w.ID] = true
			if len(w.StartCommand) == 0 {
				errs = append(errs, fmt.Errorf("service %q worker %q: startCommand must not be empty", svc.ID, w.ID))
			}
		}

		for _, a := range svc.Actions {
			if len(a.Command) == 0 {
				errs = append(errs, fmt.Errorf("service %q action %q: command must not be empty", svc.ID, a.ID))
			}
		}

		for _, hc := range svc.HealthChecks {
			if !supportedHealthCheckTypes[hc.Type] {
				errs = append(errs, fmt.Errorf("service %q: unsupported health check type %q", svc.ID, hc.Type))
			}
		}
	}

	for _, svc := range cfg.Services {
		for _, dep := range svc.DependsOn {
			if !serviceIDs[dep] {
				errs = append(errs, fmt.Errorf("service %q: dependsOn references missing service %q", svc.ID, dep))
			}
		}
	}

	presetIDs := map[string]bool{}
	for _, preset := range cfg.Presets {
		if presetIDs[preset.ID] {
			errs = append(errs, fmt.Errorf("duplicate preset id %q", preset.ID))
		}
		presetIDs[preset.ID] = true

		for _, svcID := range preset.Services {
			if !serviceIDs[svcID] {
				errs = append(errs, fmt.Errorf("preset %q references missing service %q", preset.ID, svcID))
			}
		}
	}

	return errs
}
