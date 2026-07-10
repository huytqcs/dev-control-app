// Package config defines the workspace YAML schema and loads/validates it.
package config

// WorkspaceConfig is the root of a devctl.yaml file.
type WorkspaceConfig struct {
	Name     string          `yaml:"name"`
	Services []ServiceConfig `yaml:"services"`
	Presets  []PresetConfig  `yaml:"presets"`
}

type ServiceConfig struct {
	ID           string            `yaml:"id"`
	Name         string            `yaml:"name"`
	Type         string            `yaml:"type"`
	Path         string            `yaml:"path"`
	StartCommand []string          `yaml:"startCommand"`
	Env          map[string]string `yaml:"env"`
	Port         int               `yaml:"port"`
	OpenURLs     []string          `yaml:"openUrls"`
	DependsOn    []string          `yaml:"dependsOn"`
	HealthChecks []HealthCheck     `yaml:"healthChecks"`
	Workers      []WorkerConfig    `yaml:"workers"`
	Actions      []ActionConfig    `yaml:"actions"`
	Tags         []string          `yaml:"tags"`
}

type WorkerConfig struct {
	ID           string            `yaml:"id"`
	Name         string            `yaml:"name"`
	StartCommand []string          `yaml:"startCommand"`
	Env          map[string]string `yaml:"env"`
	// AutoStart, when true, starts this worker whenever its parent service
	// starts (and stops it whenever the service stops or crashes) — e.g. a
	// Sidekiq worker that should always run alongside its Rails server.
	// Default false: workers are independently controllable
	// (ARCHITECTURE.md §12.2) unless opted into this.
	AutoStart bool `yaml:"autoStart"`
}

type PresetConfig struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Services []string `yaml:"services"`
}

type ActionConfig struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Command     []string          `yaml:"command"`
	Env         map[string]string `yaml:"env"`
	Description string            `yaml:"description"`
}

// HealthCheck.Type is "tcp" or "http".
type HealthCheck struct {
	Type     string `yaml:"type"`
	Port     int    `yaml:"port,omitempty"`
	URL      string `yaml:"url,omitempty"`
	Interval int    `yaml:"interval,omitempty"`
	Timeout  int    `yaml:"timeout,omitempty"`
}

var supportedHealthCheckTypes = map[string]bool{
	"tcp":  true,
	"http": true,
}
