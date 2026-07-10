package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads, parses, defaults, and validates a workspace config file at path.
// A leading "~" in path (and in any service.path) is expanded to the user's
// home directory.
func Load(path string) (*WorkspaceConfig, error) {
	expanded, err := expandHome(path)
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}

	data, err := os.ReadFile(expanded)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", expanded, err)
	}

	var cfg WorkspaceConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", expanded, err)
	}

	for i := range cfg.Services {
		expandedPath, err := expandHome(cfg.Services[i].Path)
		if err != nil {
			return nil, fmt.Errorf("resolve path for service %q: %w", cfg.Services[i].ID, err)
		}
		cfg.Services[i].Path = expandedPath
	}

	applyDefaults(&cfg)

	if errs := Validate(&cfg); len(errs) > 0 {
		return nil, fmt.Errorf("invalid config %s:\n%s", expanded, joinErrors(errs))
	}

	warnMissingPaths(&cfg)

	return &cfg, nil
}

// warnMissingPaths logs (but does not fail on) services whose configured
// path doesn't exist on disk yet — a common state while a devctl.yaml is
// still being filled in with real repo locations. Failing to start a
// specific service later gives a much clearer error than this, but seeing
// every misconfigured path at startup, all at once, saves a lot of
// trial-and-error clicking.
func warnMissingPaths(cfg *WorkspaceConfig) {
	for _, svc := range cfg.Services {
		if info, err := os.Stat(svc.Path); err != nil || !info.IsDir() {
			log.Printf("warning: service %q: path does not exist or is not a directory: %s", svc.ID, svc.Path)
		}
	}
}

func expandHome(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
}

func joinErrors(errs []error) string {
	var b strings.Builder
	for _, e := range errs {
		b.WriteString("  - ")
		b.WriteString(e.Error())
		b.WriteString("\n")
	}
	return b.String()
}
