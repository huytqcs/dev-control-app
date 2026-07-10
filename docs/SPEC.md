# Dev Control App V2 — SPEC.md

## 1) Purpose

This document turns **Plan.md** into an implementation-level spec for a lightweight local developer control app with:

- **Backend:** Go
- **Frontend:** React + TypeScript
- **Runtime model:** direct process ownership by the app
- **UI model:** browser-based local dashboard first
- **Target OS:** macOS first

The app is intended to replace `appctl2.py` with a cleaner architecture and a more modern UI while staying reasonably lightweight.

---

# 2) Non-goals

The first version should **not** try to solve everything.

## Explicit non-goals for v1
- Full tmux-based runtime management
- Embedded terminal emulator
- Team sync / multi-user collaboration
- Cloud account / auth system
- Full persistence / history database
- Docker / Kubernetes orchestration
- Plugin marketplace / extension system
- Visual config editor

---

# 3) System overview

```txt
React UI (browser)
    |
    | HTTP + WebSocket
    v
Go Local Server
    |
    +-- Config Loader
    +-- Workspace Service
    +-- Runtime Manager
    +-- Log Manager
    +-- Git Manager
    +-- Health Manager
    +-- Action Runner
```

---

# 4) Runtime principles

## 4.1 Source of truth
The Go backend is the source of truth for:

- service status
- worker status
- current process PID
- log buffers
- health status
- action execution state

## 4.2 Process ownership
The app should start and own service processes directly using `os/exec`.

Do **not** treat tmux as the main process owner.

## 4.3 Event-driven updates
Use WebSocket events for:
- service state changes
- log appends
- health changes
- git state refreshes
- action output

Avoid high-frequency polling loops in the frontend.

## 4.4 Config-driven app
The app must load services, presets, actions, workers, URLs, and health checks from YAML config.

---

# 5) Tech stack

# 5.1 Backend
- Go 1.23+ (or latest stable)
- `chi` router
- `gorilla/websocket` or `nhooyr.io/websocket`
- `gopkg.in/yaml.v3` for config parsing
- standard library `os/exec`, `context`, `sync`, `net/http`, `bufio`

## Suggested libraries
- Router: `github.com/go-chi/chi/v5`
- Middleware: `github.com/go-chi/httplog` or standard logging middleware
- UUID if needed: `github.com/google/uuid`

# 5.2 Frontend
- React
- TypeScript
- Vite
- Tailwind
- shadcn/ui
- TanStack Query
- Zustand (optional)
- React Router (if needed; single dashboard can start without it)

---

# 6) Monorepo structure

```txt
devctl/
  backend/
    cmd/devctl/
      main.go

    internal/
      app/
        app.go

      api/
        router.go
        middleware.go
        handlers_workspace.go
        handlers_services.go
        handlers_logs.go
        handlers_git.go
        handlers_actions.go
        handlers_presets.go
        ws_hub.go
        ws_events.go

      config/
        loader.go
        schema.go
        validate.go
        defaults.go

      workspace/
        service.go

      runtime/
        manager.go
        service_instance.go
        worker_instance.go
        process_runner.go
        process_state.go
        events.go

      logs/
        manager.go
        ring_buffer.go
        parser.go

      git/
        service.go
        models.go

      health/
        service.go
        checker.go
        models.go

      actions/
        runner.go
        models.go

      openers/
        browser.go
        editor.go
        terminal.go
        finder.go

      common/
        errs/
        shell/
        clock/
        ids/

  frontend/
    src/
      app/
      pages/
      components/
      hooks/
      lib/
      types/
      features/
```

---

# 7) Backend domain model

# 7.1 Config schema types

```go
package config

type WorkspaceConfig struct {
	Name     string          `yaml:"name"`
	Services []ServiceConfig `yaml:"services"`
	Presets  []PresetConfig  `yaml:"presets"`
}

type ServiceConfig struct {
	ID           string         `yaml:"id"`
	Name         string         `yaml:"name"`
	Type         string         `yaml:"type"`
	Path         string         `yaml:"path"`
	StartCommand []string       `yaml:"startCommand"`
	Env          map[string]string `yaml:"env"`
	Port         int            `yaml:"port"`
	OpenURLs     []string       `yaml:"openUrls"`
	DependsOn    []string       `yaml:"dependsOn"`
	HealthChecks []HealthCheck  `yaml:"healthChecks"`
	Workers      []WorkerConfig `yaml:"workers"`
	Actions      []ActionConfig `yaml:"actions"`
	Tags         []string       `yaml:"tags"`
}

type WorkerConfig struct {
	ID           string            `yaml:"id"`
	Name         string            `yaml:"name"`
	StartCommand []string          `yaml:"startCommand"`
	Env          map[string]string `yaml:"env"`
	AutoStart    bool              `yaml:"autoStart"` // start/stop with the parent service (e.g. Sidekiq with Rails)
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

type HealthCheck struct {
	Type     string `yaml:"type"` // tcp | http
	Port     int    `yaml:"port,omitempty"`
	URL      string `yaml:"url,omitempty"`
	Interval int    `yaml:"interval,omitempty"`
	Timeout  int    `yaml:"timeout,omitempty"`
}
```

---

# 7.2 Runtime state types

```go
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
	WorkerStopping WorkerStatus = "stopping"
)

type WorkerState struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Status       WorkerStatus `json:"status"`
	PID          int          `json:"pid,omitempty"`
	LastError    string       `json:"lastError,omitempty"`
	LastExitCode *int         `json:"lastExitCode,omitempty"`
}

type HealthState struct {
	Status string `json:"status"` // unknown | healthy | unhealthy
}

type GitState struct {
	Branch string `json:"branch"`
	Dirty  bool   `json:"dirty"`
	Ahead  int    `json:"ahead"`
	Behind int    `json:"behind"`
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
```

---

# 7.3 API DTOs

Use DTOs instead of exposing internal runtime structs directly if the app grows.

```go
package api

type ServiceDTO struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Path      string                 `json:"path"`
	Port      int                    `json:"port"`
	OpenURLs  []string               `json:"openUrls"`
	DependsOn []string               `json:"dependsOn"`
	State     ServiceStateDTO        `json:"state"`
	Actions   []ActionSummaryDTO     `json:"actions"`
	Workers   []WorkerSummaryDTO     `json:"workers"`
}

type ServiceStateDTO struct {
	Status       string          `json:"status"`
	PID          int             `json:"pid,omitempty"`
	StartedAt    *string         `json:"startedAt,omitempty"`
	LastError    string          `json:"lastError,omitempty"`
	LastExitCode *int            `json:"lastExitCode,omitempty"`
	Git          GitStateDTO     `json:"git"`
	Health       HealthStateDTO  `json:"health"`
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

type WorkerSummaryDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	PID          int    `json:"pid,omitempty"`
	LastError    string `json:"lastError,omitempty"`
	LastExitCode *int   `json:"lastExitCode,omitempty"`
	AutoStart    bool   `json:"autoStart"`
}
```

---

# 8) Config loading and validation

# 8.1 Config file location
Initial strategy:
- CLI arg: `--config /path/to/devctl.yaml`
- fallback default: `~/.config/devctl/devctl.yaml`

## 8.2 Validation rules
Validation should fail fast on startup if:

- workspace name is empty
- duplicate service IDs exist
- duplicate preset IDs exist
- duplicate worker IDs inside the same service exist
- a preset references a missing service ID
- a service depends on a missing service ID
- `startCommand` is empty for any service
- action command is empty
- health check type is unsupported
- path does not exist (optional strict mode; maybe warning instead of hard fail)

## 8.3 Path handling
Expand `~` in config paths before use.

---

# 9) Core backend services

# 9.1 App container

```go
type App struct {
	ConfigLoader  *config.Loader
	WorkspaceSvc  *workspace.Service
	RuntimeMgr    *runtime.Manager
	LogMgr        *logs.Manager
	GitSvc        *git.Service
	HealthSvc     *health.Service
	ActionRunner  *actions.Runner
	WSHub         *api.WSHub
}
```

The app container wires dependencies together and provides them to HTTP handlers.

---

# 9.2 Workspace service

## Responsibility
Provide read access to config-defined workspace metadata.

### Methods
```go
type Service interface {
	GetWorkspace() config.WorkspaceConfig
	GetService(serviceID string) (config.ServiceConfig, bool)
	ListServices() []config.ServiceConfig
	GetPreset(presetID string) (config.PresetConfig, bool)
	ListPresets() []config.PresetConfig
}
```

---

# 9.3 Runtime manager

This is the core orchestrator.

## Responsibility
- manage service processes
- manage worker processes
- own service state in memory
- emit state events
- connect stdout/stderr to log manager
- react to process exit

## Suggested public interface

```go
type Manager interface {
	ListStates() []ServiceState
	GetState(serviceID string) (ServiceState, bool)

	StartService(ctx context.Context, serviceID string) error
	StopService(ctx context.Context, serviceID string) error
	RestartService(ctx context.Context, serviceID string) error

	StartWorker(ctx context.Context, serviceID, workerID string) error
	StopWorker(ctx context.Context, serviceID, workerID string) error
}
```

## Internal structures

```go
type serviceRuntime struct {
	config    config.ServiceConfig
	state     ServiceState
	cmd       *exec.Cmd
	cancel    context.CancelFunc
	stdoutDone chan struct{}
	stderrDone chan struct{}
	mu        sync.RWMutex

	workers map[string]*workerRuntime
}

type workerRuntime struct {
	config config.WorkerConfig
	state  WorkerState
	cmd    *exec.Cmd
	cancel context.CancelFunc
	mu     sync.RWMutex
}
```

---

# 10) Process runner design

Create a reusable process runner abstraction so service and worker logic share the same behavior.

## Interface
```go
type ProcessRunner interface {
	Start(ctx context.Context, opts ProcessOptions) (*RunningProcess, error)
	Stop(proc *RunningProcess, timeout time.Duration) error
}

type ProcessOptions struct {
	ID      string
	Command []string
	Dir     string
	Env     map[string]string
}

type RunningProcess struct {
	Cmd        *exec.Cmd
	PID        int
	Stdout     io.ReadCloser
	Stderr     io.ReadCloser
	CancelFunc context.CancelFunc
}
```

## Start behavior
1. Validate command length > 0
2. Build command using `exec.CommandContext`
3. Set working directory
4. Merge env vars with current environment
5. Capture stdout and stderr pipes
6. Start process
7. Return process handle

## Stop behavior
Recommended order:
1. send SIGTERM
2. wait up to N seconds
3. if still alive, send SIGKILL

For macOS, graceful stop via `Process.Signal(syscall.SIGTERM)` is appropriate.

---

# 11) Service lifecycle spec

# 11.1 Start service flow

```txt
User clicks Start
 -> handler validates service exists
 -> runtime manager checks current state
 -> if already running: no-op or return conflict
 -> mark state = starting
 -> emit service.updated
 -> start process
 -> attach stdout/stderr readers
 -> begin log streaming
 -> register process exit watcher
 -> schedule health check loop if configured
 -> if process starts successfully:
      update pid
      set status running
      emit service.updated
    else:
      set status failed
      set last error
      emit service.updated
```

## 11.2 Stop service flow
```txt
User clicks Stop
 -> runtime manager checks current state
 -> if stopped: no-op
 -> set status stopping
 -> emit service.updated
 -> stop all running workers first (optional but recommended)
 -> terminate process
 -> set status stopped
 -> clear pid
 -> emit service.updated
```

## 11.3 Restart service flow
Implementation can simply do:
1. Stop
2. Start

Potential future improvement:
- if stop fails, surface partial failure
- preserve logs in buffer across restart

---

# 12) Worker lifecycle spec

Workers are child runtime units attached to a service.

## Rules
- workers are configured inside a service
- workers can be started/stopped independently by default
- `autoStart: true` on a worker ties it to its parent service's lifecycle:
  it starts when the service starts and stops when the service stops or
  crashes (e.g. Sidekiq with Rails). Default `false` keeps it decoupled.
- worker logs should be stored separately but still associated with the parent service in the UI

## Log stream key
Use a stream identifier:
- `service:<serviceId>`
- `worker:<serviceId>:<workerId>`

---

# 13) Log manager design

# 13.1 Goals
- keep recent logs in memory
- stream logs live to UI
- allow “fetch recent logs”
- separate stdout and stderr if useful

# 13.2 Ring buffer model

```go
type LogEntry struct {
	ID        string    `json:"id"`
	StreamKey string    `json:"streamKey"`
	Source    string    `json:"source"` // stdout | stderr
	Line      string    `json:"line"`
	Time      time.Time `json:"time"`
	Level     string    `json:"level,omitempty"` // info | warn | error if parsed
}

type RingBuffer struct {
	entries []LogEntry
	start   int
	count   int
	cap     int
	mu      sync.RWMutex
}
```

## Recommended default
- keep last **2,000–5,000 lines** per stream

## 13.3 Log append flow
When stdout/stderr reader receives a line:
1. create `LogEntry`
2. append to ring buffer
3. publish websocket event `log.appended`

## 13.4 Log parsing
For MVP, log parsing can be heuristic:
- if line contains `ERROR`, `Error`, `FATAL` → mark `error`
- if line contains `WARN`, `Warning` → mark `warn`

---

# 14) Git service design

## 14.1 Responsibilities
- read current branch
- read dirty state
- run fetch/pull/push/checkout
- optionally compute ahead/behind

## 14.2 Public interface

```go
type Service interface {
	GetState(ctx context.Context, repoPath string) (runtime.GitState, error)
	Fetch(ctx context.Context, repoPath string) error
	Pull(ctx context.Context, repoPath string) error
	Push(ctx context.Context, repoPath string) error
	Checkout(ctx context.Context, repoPath, branch string) error
}
```

## 14.3 Implementation strategy
Use `git` CLI, not a Go git library, at least for v1.  
That keeps behavior aligned with local dev expectations.

Examples:
- branch: `git rev-parse --abbrev-ref HEAD`
- dirty: `git status --porcelain`
- fetch: `git fetch`
- pull: `git pull`
- push: `git push`
- checkout: `git checkout <branch>`

## 14.4 Refresh strategy
Do not poll every second.

Recommended:
- refresh on app load
- refresh after git actions
- refresh every 30–60s in background for visible services only, or all services if count is small

---

# 15) Health service design

## 15.1 Health check types
Support only these initially:
- `tcp`
- `http`

## 15.2 Interfaces

```go
type Service interface {
	StartMonitoring(serviceID string, checks []config.HealthCheck)
	StopMonitoring(serviceID string)
	GetHealth(serviceID string) runtime.HealthState
}
```

## 15.3 TCP check
Try opening a TCP connection to `localhost:<port>` within timeout.

## 15.4 HTTP check
Perform HTTP GET to configured URL and expect 2xx/3xx.

## 15.5 Health loop
Each running service with health checks gets a goroutine ticker based on check interval.

When service stops:
- cancel health monitoring

---

# 16) Action runner design

## 16.1 Responsibilities
Run user-defined commands for a service:
- install deps
- db migrate
- tests
- custom scripts

## 16.2 Behavior
- actions are one-off commands
- action output should be streamed to logs or a separate action output channel
- actions should not mutate service status unless explicitly designed to

## 16.3 Interface

```go
type Runner interface {
	RunAction(ctx context.Context, service config.ServiceConfig, actionID string) error
}
```

## 16.4 Action output model
For MVP, stream action output into a dedicated stream key:
- `action:<serviceId>:<actionId>:<runId>`

---

# 17) Open helpers (macOS-specific)

## 17.1 Browser
Open URL with:
```bash
open http://localhost:3000
```

## 17.2 Finder
Open repo folder:
```bash
open /path/to/repo
```

## 17.3 Terminal
Potential choices:
- open Terminal.app in repo path
- open iTerm if user config says so
- just reveal folder initially and leave terminal later if needed

## 17.4 IDE
Can be config-driven:
- VS Code: `code /path`
- Cursor: `cursor /path`
- JetBrains Toolbox-specific launch later if needed

---

# 18) WebSocket event contract

Use a single event envelope.

```go
type EventEnvelope struct {
	Type      string      `json:"type"`
	ServiceID string      `json:"serviceId,omitempty"`
	Payload   interface{} `json:"payload"`
	Time      string      `json:"time"`
}
```

Worker events carry the worker's own ID inside their payload (see
`worker.updated` below) rather than a top-level `workerId` — every event is
still scoped to a service via `serviceId`.

## 18.1 Event types

### `service.updated`
Payload is the full runtime `ServiceState` (§7.2) for the service, not just
the changed fields:
```json
{
  "id": "core-be",
  "name": "Core BE",
  "status": "running",
  "pid": 12345,
  "port": 3000,
  "git": { "branch": "main", "dirty": false, "ahead": 0, "behind": 0 },
  "health": { "status": "healthy" },
  "workers": [
    { "id": "sidekiq", "name": "Sidekiq", "status": "running", "autoStart": true }
  ]
}
```

### `worker.updated`
Payload:
```json
{
  "worker": {
    "id": "sidekiq",
    "name": "Sidekiq",
    "status": "running",
    "pid": 12346,
    "autoStart": true
  }
}
```

### `log.appended`
Payload:
```json
{
  "entry": {
    "id": "log_123",
    "streamKey": "service:core-be",
    "source": "stdout",
    "line": "Rails server started",
    "time": "2026-07-10T10:00:00Z",
    "level": "info"
  }
}
```

### `health.updated`
Payload:
```json
{
  "health": { "status": "healthy" }
}
```

### `git.updated`
Payload:
```json
{
  "git": {
    "branch": "feature/devctl-v2",
    "dirty": true,
    "ahead": 0,
    "behind": 2
  }
}
```

### `action.output`
Payload:
```json
{
  "runId": "action_001",
  "entry": {
    "line": "Bundle install complete"
  }
}
```

---

# 19) HTTP API spec

All routes prefixed with `/api`.

# 19.1 Workspace endpoints

## `GET /api/workspace`
Returns workspace metadata and presets.

### Response example
```json
{
  "name": "mealsuite",
  "presets": [
    { "id": "core", "name": "Core", "services": ["core-app", "core-be"] }
  ]
}
```

---

# 19.2 Services endpoints

## `GET /api/services`
Return all services with current state.

### Response example
```json
{
  "services": [
    {
      "id": "core-be",
      "name": "Core BE",
      "type": "rails",
      "path": "/Users/harryta/Desktop/code/core-be",
      "port": 3000,
      "openUrls": [],
      "dependsOn": [],
      "state": {
        "status": "running",
        "pid": 12345,
        "git": { "branch": "main", "dirty": false, "ahead": 0, "behind": 0 },
        "health": { "status": "healthy" }
      },
      "actions": [
        { "id": "migrate", "name": "DB Migrate" }
      ],
      "workers": [
        { "id": "sidekiq", "name": "Sidekiq", "status": "running", "autoStart": true }
      ]
    }
  ]
}
```

## `GET /api/services/:id`
Return one service with state and config summary.

## `POST /api/services/:id/start`
Starts service.

### Response
- `200 OK` with updated service state
- `409 Conflict` if already running (or return 200 no-op if preferred)

## `POST /api/services/:id/stop`
Stops service.

## `POST /api/services/:id/restart`
Restarts service.

---

# 19.3 Worker endpoints

## `POST /api/services/:id/workers/:workerId/start`
Start worker.

## `POST /api/services/:id/workers/:workerId/stop`
Stop worker.

### Response
Both return `200 OK` with the full updated `ServiceDTO` (not just the
worker) — the frontend patches its whole services cache entry from one
response, same as start/stop/restart.

---

# 19.4 Logs endpoints

## `GET /api/services/:id/logs?limit=500`
Returns recent logs for the service main stream.

## Optional future
- `GET /api/streams/:streamKey/logs?limit=500`

### Response example
```json
{
  "entries": [
    {
      "id": "log1",
      "streamKey": "service:core-be",
      "source": "stdout",
      "line": "Started GET /api/v1/users",
      "time": "2026-07-10T10:00:00Z",
      "level": "info"
    }
  ]
}
```

---

# 19.5 Preset endpoints

## `POST /api/presets/:id/start`
Start all services in preset.

### Suggested behavior
- resolve service order using dependencies if possible
- start services sequentially or with limited concurrency
- if one fails, continue or stop based on future policy; for MVP, continue but surface failures

## `POST /api/presets/:id/stop`
Stop all services in preset, in reverse dependency order.

### Response
```json
{ "errors": [] }
```
`errors` lists one message per service that failed to start/stop (partial
failure summary); an empty array means every service in the preset succeeded.
Per-service progress is still observed via `service.updated` events, not
this response.

---

# 19.3.1 Force-kill endpoint

## `POST /api/services/:id/force-kill`
Kills whatever process is listening on the service's configured port,
regardless of whether this devctl backend believes it owns that process.
Manual escape hatch for orphan reconciliation (§26.1) guessing wrong.

### Response
`200 OK` with the updated `ServiceDTO` (status reset to `stopped`).

---

# 19.6 Git endpoints

## `GET /api/services/:id/git/branches`
Return every local branch plus every remote-tracking branch's short name
(deduplicated, sorted) — the list a branch-search checkout UI filters
client-side.

### Response
```json
{ "branches": ["feature-a", "feature-b", "main"] }
```

## `POST /api/services/:id/git/fetch`
## `POST /api/services/:id/git/pull`
## `POST /api/services/:id/git/push`

## `POST /api/services/:id/git/checkout`
### Request body
```json
{
  "branch": "feature/my-branch"
}
```

### Response
Return updated git state.

---

# 19.7 Actions endpoints

## `POST /api/services/:id/actions/:actionId`
Run custom action.

### Response example
```json
{
  "runId": "action_001",
  "status": "started"
}
```

---

# 19.8 Open helper endpoints

## `POST /api/services/:id/open-browser`
Open first matching URL or preferred URL.

## `POST /api/services/:id/open-repo`
Open repo in Finder or IDE depending on endpoint split.

## `POST /api/services/:id/open-terminal`
Open terminal at repo path.

---

# 20) Error model

Use a consistent error response shape.

```json
{
  "error": {
    "code": "service_not_found",
    "message": "Service core-be was not found"
  }
}
```

## Suggested error codes
- `service_not_found`
- `preset_not_found`
- `worker_not_found`
- `action_not_found`
- `service_already_running`
- `service_not_running`
- `invalid_request`
- `process_start_failed`
- `git_command_failed`
- `action_failed`

---

# 21) Frontend architecture

# 21.1 Frontend folder structure

```txt
frontend/src/
  app/
    App.tsx
    providers.tsx

  pages/
    DashboardPage.tsx

  components/
    layout/
      AppShell.tsx
      Sidebar.tsx
      TopBar.tsx

    workspace/
      PresetBar.tsx
      ServiceGrid.tsx
      ServiceCard.tsx
      ServiceDetailsPanel.tsx

    logs/
      LogViewer.tsx
      LogToolbar.tsx

    git/
      GitPanel.tsx
      BranchCheckoutForm.tsx

    health/
      HealthBadge.tsx

    actions/
      ActionList.tsx

    common/
      StatusBadge.tsx
      EmptyState.tsx
      LoadingState.tsx

  hooks/
    useWorkspaceQuery.ts
    useServicesQuery.ts
    useServiceLogs.ts
    useRealtimeEvents.ts

  lib/
    api.ts
    ws.ts
    utils.ts

  store/
    uiStore.ts

  types/
    api.ts
```

---

# 21.2 Page layout

## `DashboardPage`
Main page layout:

- left sidebar: workspace + presets
- top bar: global actions
- center: service cards grid
- right panel: selected service details

---

# 21.3 Frontend state model

## Server state — TanStack Query
Use for:
- workspace metadata
- service list
- initial logs fetch
- action responses

## Realtime state — WebSocket + local store
Use for:
- service.updated
- log.appended
- worker.updated
- git.updated
- health.updated

## UI state — Zustand or local component state
Use for:
- selected service
- selected log filter
- follow tail toggle
- open/closed panels

---

# 22) Frontend component spec

# 22.1 `ServiceCard`
## Props
```ts
type ServiceCardProps = {
  service: ServiceDto;
  isSelected: boolean;
  onSelect: () => void;
  onStart: () => void;
  onStop: () => void;
  onRestart: () => void;
  onOpenBrowser: () => void;
};
```

## Displays
- name
- status badge
- branch
- dirty badge
- port
- worker indicators
- buttons

---

# 22.2 `ServiceDetailsPanel`
Tabs:
- Logs
- Git
- Info
- Actions

## Behavior
When selected service changes:
- fetch recent logs
- render current state
- subscribe to live log events for selected stream if needed

---

# 22.3 `LogViewer`
## Features
- virtualized rendering if log volume becomes high
- follow tail toggle
- filter by severity
- search text
- copy visible logs

## Data source
- initial fetch from `GET /api/services/:id/logs`
- incremental updates via `log.appended`

---

# 22.4 `GitPanel`
Displays:
- current branch
- dirty state
- ahead/behind if available
- fetch/pull/push buttons
- checkout branch form

---

# 22.5 `PresetBar`
Displays preset buttons:
- Start Core
- Stop Core
- Start Touch
- Stop Touch

Potential future:
- startup progress display for preset launch

---

# 23) Frontend API client spec

Create a typed API client wrapper.

## Example functions
```ts
export async function getWorkspace(): Promise<WorkspaceDto> {}
export async function getServices(): Promise<ServiceDto[]> {}
export async function startService(serviceId: string): Promise<void> {}
export async function stopService(serviceId: string): Promise<void> {}
export async function restartService(serviceId: string): Promise<void> {}
export async function getServiceLogs(serviceId: string, limit = 500): Promise<LogEntryDto[]> {}
export async function checkoutBranch(serviceId: string, branch: string): Promise<void> {}
```

---

# 24) Realtime client spec

Create a single `useRealtimeEvents()` hook that:
1. opens websocket connection
2. parses event envelopes
3. dispatches updates to query cache and/or Zustand store

## Event handling examples
- `service.updated` → update service state in query cache
- `log.appended` → append log entry to selected service log store
- `git.updated` → update git state in service cache
- `health.updated` → update health badge state

---

# 25) Dependency startup behavior

## MVP behavior
If service `A` depends on `B`, starting `A` should optionally:
- either start only `A`
- or offer “start dependencies too”

Recommended initial rule:
- preset start handles dependency order
- individual service start does **not** automatically start dependencies unless explicitly enabled later

This avoids surprising side effects.

---

# 26) Preset startup algorithm

## Inputs
- preset service list
- dependency graph from config

## Behavior
1. build a deduplicated list of services in the preset
2. topologically sort by dependencies if possible
3. start services in sorted order
4. wait a short delay between starts if needed
5. emit service updates as they start/fail

If cycle detected:
- log warning
- fall back to config order

---

# 26.1) Orphan reconciliation and force-kill

Services run `Setsid` (§6, `process_runner.go`), each in its own session —
so they survive a devctl backend restart as orphans still holding their
port, even though the backend's fresh in-memory state has no record of them.

## Reconciliation on startup
On `app.New()`, `Manager.ReconcileOrphans` probes each stopped service's
configured port; if something is already listening, it's adopted as
`running` (best-effort, no real PID/process handle — never kills anything,
just makes displayed state match reality).

## Stop on a reconciled/handle-less service
`StopService` on a service with no in-memory process handle (either
adopted by reconciliation, or never started by this backend instance) falls
back to `ForceKillPort`: kill whatever holds the configured port, by PID,
via `lsof -ti :<port>` + `SIGKILL`.

## Manual escape hatch
`POST /api/services/:id/force-kill` (§19.3.1) exposes the same
`ForceKillPort` mechanism directly, for when reconciliation's guess is
wrong or a service is stuck in a state the UI can't otherwise recover from.

## Not implemented
A `pidfile:` config field would let reconciliation identify a service more
precisely than a bare port probe (a port can be reused by something else
entirely). Deferred — port probing covers the common case and needs no new
config field.

---

# 27) Logging UX decisions

## Keep it practical
For MVP:
- do not build a full log database
- do not build log persistence across app restarts
- do not over-parse logs

Do:
- keep recent in-memory logs
- make copy/search/filter good
- highlight likely errors

---

# 28) Performance guidance

## Backend
- only health-check running services
- git refresh should be low-frequency
- avoid one goroutine per tiny thing if it can be grouped sensibly
- cap log buffers

## Frontend
- avoid rerendering the entire service grid for every log event
- keep logs in a focused store for selected service
- virtualize log viewer if line count is high

---

# 29) Security / safety assumptions

This is a **local-only trusted developer tool**.  
Still, a few guardrails are worth having:

- do not expose server outside localhost by default
- bind to `127.0.0.1`
- do not execute arbitrary UI-provided shell snippets unless intentionally supported
- actions should come from config, not raw user text input in MVP

---

# 30) macOS-specific assumptions

v1 is optimized for macOS:
- `open` command for browser/folder
- POSIX signals for process stop
- local repo paths on macOS filesystem
- shell commands assume zsh/bash compatible environment

Cross-platform support can come later if needed.

---

# 31) Testing strategy

# 31.1 Backend tests
## Unit tests
- config validation
- dependency resolution / preset ordering
- ring buffer behavior
- git command parsing
- health checker logic

## Integration-ish tests
- process runner starts a small dummy command
- runtime manager state transitions
- logs captured from dummy process
- stop/restart behavior

# 31.2 Frontend tests
Focus on:
- service card actions
- log viewer rendering/filtering
- websocket event reducer behavior

---

# 32) First implementation milestones

# Milestone 1 — backend alpha
Deliver:
- config loader
- runtime manager
- start/stop/restart service
- log capture
- `/api/services`
- `/api/services/:id/start`
- `/api/services/:id/stop`
- `/api/services/:id/logs`
- websocket service/log events

# Milestone 2 — frontend alpha
Deliver:
- dashboard shell
- service grid
- service start/stop
- log panel
- live updates

# Milestone 3 — daily-driver beta
Deliver:
- presets
- health checks
- git actions
- workers
- custom actions
- open browser/repo helpers

---

# 33) Mapping from `appctl2.py` concepts to v2

## Current likely concept → v2 destination

### Project/service definitions
- **old:** Python dicts / constants / inline config
- **new:** YAML `services[]`

### Preset groups
- **old:** hardcoded start groups
- **new:** YAML `presets[]`

### Start/stop shell logic
- **old:** Python functions + tmux shelling
- **new:** `runtime.Manager` + `ProcessRunner`

### Log viewing
- **old:** tmux pane capture / polling
- **new:** stdout/stderr stream -> `logs.Manager` -> WebSocket

### Sidekiq start
- **old:** custom branch of logic
- **new:** `workers[]` per service + worker runtime

### Git actions
- **old:** Python helper functions
- **new:** `git.Service`

### Open browser / open folder
- **old:** Python shell helpers
- **new:** open helper endpoints

---

# 34) Suggested initial backlog

## Backend
1. config schema + loader
2. workspace service
3. process runner
4. runtime manager for services
5. log manager
6. websocket hub
7. services API
8. health service
9. git service
10. worker runtime
11. action runner

## Frontend
1. API types/client
2. dashboard shell
3. service grid/cards
4. service actions
5. logs panel
6. websocket integration
7. detail panel tabs
8. git panel
9. preset bar
10. actions panel

---

# 35) Final implementation stance

If you follow this spec, the app should end up as:

- **lighter than Electron-first**
- **much nicer UI than Textual**
- **cleaner than a tmux-driven control script**
- **good enough to become your actual daily local dev control center**

The key architectural rule is still the same:

## **Go owns runtime + logs + orchestration. React owns the control UI. tmux is optional, not foundational.**
