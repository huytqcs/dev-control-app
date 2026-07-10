# Dev Control App V2 Plan

## Goal
Build a lightweight local developer control center for macOS with:

- **Backend:** Go
- **Frontend:** React + TypeScript
- Modern UI
- Better process/runtime management than `appctl2.py`
- Live logs, git actions, workspace presets, Sidekiq/worker support
- Lighter than Electron-first desktop architecture

---

# 1) Product goal

## What this app is
A **local developer workspace manager** that helps you:

- start / stop / restart local services
- manage multi-service presets (`core`, `touch`, `pos`, etc.)
- view live logs
- run Sidekiq / workers
- switch branches / run git actions
- open FE in browser
- open repo in IDE
- run common setup / migration / test commands

## What this app is not
- not a CI/CD platform
- not Kubernetes for local dev
- not a generic terminal replacement
- not “tmux with buttons”

The app should own the **developer workflow layer**, not just shell out blindly.

---

# 2) Architecture decision

## Final stack

### Backend
- **Go**
- HTTP API + WebSocket
- local config-driven runtime manager
- direct process management (not tmux-first)

### Frontend
- **React**
- **TypeScript**
- **Vite**
- **Tailwind**
- **shadcn/ui**
- **TanStack Query** for server state
- **Zustand** for local UI state if needed

## Packaging approach

### Phase 1
Run as a **local web app**
- Go server on localhost
- React app served locally
- open in browser

### Phase 2 optional
Wrap in **Tauri** if you want a true desktop app later.

That keeps v1 lighter and simpler.

---

# 3) Core product principles

## Principle 1 — Go owns the runtime
Go should directly manage:
- service processes
- workers
- log streams
- health checks
- service state
- restart behavior

Not tmux.

tmux can exist as an **optional escape hatch**:
- “Open service in tmux”
- “Attach shell”
- “Run command in tmux mode”

But it should not be the main source of truth.

## Principle 2 — config-driven, not hardcoded
All projects/services/presets should live in config files, not in code.

## Principle 3 — event-driven where possible
Avoid the `appctl2.py` pattern of “poll everything every second”.

Use:
- process events
- log stream events
- explicit refresh actions
- slower background refresh for git metadata

## Principle 4 — logs are a first-class feature
The app should not just “show terminal output”. It should make logs usable:
- live stream
- search
- copy
- filter
- preserve recent logs
- error highlighting

## Principle 5 — optimize for daily dev workflows
The app should make these tasks faster:
- “Start my morning workspace”
- “Switch to a branch and boot related services”
- “See why the backend failed”
- “Restart FE + open browser”
- “Run sidekiq and inspect logs”
- “Fetch/pull and switch repo state”

---

# 4) Functional scope for v1

## A. Workspace & service control
### Required features
- list all configured services
- show service status:
  - stopped
  - starting
  - running
  - failed
- start / stop / restart service
- start / stop preset
- start worker for service if configured
- show service port
- open FE/app URL in browser
- open repo in IDE / Finder / terminal
- show dependency relationships at least at a basic level

## B. Logs
### Required features
- live log stream per service
- logs panel in UI
- auto-follow tail toggle
- search in logs
- filter by:
  - all
  - errors
  - warnings
- copy logs
- clear visible logs
- preserve recent logs while app is running

### Nice v1 additions if easy
- service log status badge when recent errors detected
- keep last N lines in memory per service

## C. Git actions
### Required features
- show current branch per repo
- fetch
- pull
- push
- checkout branch
- show dirty/clean state

### Maybe v1.1
- ahead/behind status
- recent branches
- create branch from main

## D. Health & diagnostics
### Required features
- port health check for running services
- startup failure detection
- show last error / last exit reason
- mark service unhealthy if process dies or health check fails

## E. Common actions / tasks
### Required features
- run custom commands defined in config:
  - install deps
  - db migrate
  - run tests
  - open terminal
  - open IDE
- show output or at least success/failure

## F. Config/workspace management
### Required features
- load workspace/service definitions from config file
- support presets
- support worker definitions
- support open URLs
- support dependencies
- support health checks

---

# 5) MVP boundaries

## MVP includes
- service dashboard
- start/stop/restart
- preset start/stop
- logs streaming
- git branch display + basic git actions
- open browser / open repo / open terminal
- worker support
- config-driven services
- basic health checks

## MVP does not include
- tmux integration as a first-class runtime
- multi-user/team sync
- cloud sync
- database persistence
- plugin system
- Docker orchestration UI
- full terminal emulator in app
- complex auth

---

# 6) High-level architecture

```txt
React UI
   |
   | HTTP / WebSocket
   v
Go Local Server
   |
   +-- Workspace Config Loader
   +-- Runtime Manager
   +-- Log Manager
   +-- Git Manager
   +-- Health Checker
   +-- Action Runner
```

---

# 7) Backend design in Go

## Main backend modules

```txt
devctl/
  cmd/devctl/
    main.go

  internal/
    app/
      app.go

    api/
      router.go
      handlers_services.go
      handlers_workspace.go
      handlers_git.go
      handlers_logs.go
      handlers_actions.go
      ws.go

    config/
      loader.go
      schema.go
      validate.go

    runtime/
      manager.go
      service_instance.go
      process.go
      events.go

    logs/
      buffer.go
      stream.go
      parser.go

    git/
      service.go

    health/
      service.go
      checks.go

    actions/
      runner.go

    workspace/
      service.go
```

---

# 8) Go domain model

## Workspace config

```go
type WorkspaceConfig struct {
	Name     string          `yaml:"name"`
	Services []ServiceConfig `yaml:"services"`
	Presets  []PresetConfig  `yaml:"presets"`
}

type ServiceConfig struct {
	ID           string         `yaml:"id"`
	Name         string         `yaml:"name"`
	Path         string         `yaml:"path"`
	Type         string         `yaml:"type"` // frontend, rails, api, worker, etc.
	StartCommand []string       `yaml:"startCommand"`
	Port         int            `yaml:"port"`
	OpenURLs     []string       `yaml:"openUrls"`
	DependsOn    []string       `yaml:"dependsOn"`
	HealthChecks []HealthCheck  `yaml:"healthChecks"`
	Workers      []WorkerConfig `yaml:"workers"`
	Actions      []ActionConfig `yaml:"actions"`
}

type WorkerConfig struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	StartCommand []string `yaml:"startCommand"`
}

type PresetConfig struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Services []string `yaml:"services"`
}
```

## Runtime state

```go
type ServiceStatus string

const (
	StatusStopped  ServiceStatus = "stopped"
	StatusStarting ServiceStatus = "starting"
	StatusRunning  ServiceStatus = "running"
	StatusFailed   ServiceStatus = "failed"
)

type ServiceState struct {
	ID           string        `json:"id"`
	Status       ServiceStatus `json:"status"`
	PID          int           `json:"pid"`
	Port         int           `json:"port"`
	Branch       string        `json:"branch"`
	Dirty        bool          `json:"dirty"`
	StartedAt    *time.Time    `json:"startedAt,omitempty"`
	LastExitCode *int          `json:"lastExitCode,omitempty"`
	LastError    string        `json:"lastError,omitempty"`
	WorkerStates []WorkerState `json:"workerStates"`
}
```

## Events

```go
type EventType string

const (
	EventServiceUpdated EventType = "service.updated"
	EventLogAppended    EventType = "log.appended"
	EventGitUpdated     EventType = "git.updated"
	EventHealthUpdated  EventType = "health.updated"
)

type AppEvent struct {
	Type      EventType   `json:"type"`
	ServiceID string      `json:"serviceId,omitempty"`
	Payload   interface{} `json:"payload"`
	Time      time.Time   `json:"time"`
}
```

---

# 9) Backend responsibilities by module

## A. `config`
Load and validate workspace config from YAML/JSON.

Should handle:
- reading config file
- path normalization
- default values
- schema validation
- duplicate service ID detection

## B. `runtime`
This is the heart of the backend.

It should:
- start service process
- stop service process
- restart service process
- track PID and status
- track workers
- stream stdout/stderr
- emit service state events
- maintain in-memory service runtime registry

### Start flow
1. mark service as `starting`
2. spawn process with configured cwd/env
3. attach stdout/stderr readers
4. stream logs to buffer + websocket
5. run health checks
6. transition to `running` or `failed`

## C. `logs`
Own per-service log buffers and streaming.

Recommended:
- in-memory ring buffer per service
- optional stdout/stderr tagging
- optional parser for severity

## D. `git`
Run git operations for configured repo paths.

Functions:
- get current branch
- fetch
- pull
- push
- checkout branch
- dirty status

Important: do **not** refresh git status every second. Use on-demand refresh and slow background refresh.

## E. `health`
Determine if a running service is healthy.

Health check types:
- TCP port open
- HTTP endpoint returns success
- process still alive

## F. `actions`
Run configured custom commands like:
- install deps
- migrate DB
- run tests
- open IDE
- open terminal
- open Finder

## G. `api`
Expose the app to the React frontend using REST + WebSocket.

---

# 10) API design

## HTTP endpoints

### Workspace
- `GET /api/workspace`
- `GET /api/services`
- `GET /api/services/:id`

### Service actions
- `POST /api/services/:id/start`
- `POST /api/services/:id/stop`
- `POST /api/services/:id/restart`

### Worker actions
- `POST /api/services/:id/workers/:workerId/start`
- `POST /api/services/:id/workers/:workerId/stop`

### Logs
- `GET /api/services/:id/logs?limit=500`

### Git
- `POST /api/services/:id/git/fetch`
- `POST /api/services/:id/git/pull`
- `POST /api/services/:id/git/push`
- `POST /api/services/:id/git/checkout`

### Presets
- `POST /api/presets/:id/start`
- `POST /api/presets/:id/stop`

### Actions
- `POST /api/services/:id/actions/:actionId`

### Open helpers
- `POST /api/services/:id/open-browser`
- `POST /api/services/:id/open-repo`
- `POST /api/services/:id/open-terminal`

## WebSocket
### `GET /ws`

Use it for:
- service state changes
- log lines
- git state refreshes
- health updates
- action output if needed

Example events:

```json
{
  "type": "service.updated",
  "serviceId": "core-be",
  "payload": {
    "status": "running",
    "pid": 12345
  }
}
```

```json
{
  "type": "log.appended",
  "serviceId": "core-be",
  "payload": {
    "line": "Started GET /api/v1/users 200"
  }
}
```

---

# 11) Frontend plan

## Frontend stack
- React
- TypeScript
- Vite
- Tailwind
- shadcn/ui
- TanStack Query
- Zustand optional for UI state
- Skip xterm.js for MVP unless terminal emulation becomes necessary

## Frontend app structure

```txt
frontend/
  src/
    app/
      App.tsx
      router.tsx

    pages/
      DashboardPage.tsx

    components/
      layout/
        AppShell.tsx
        TopBar.tsx
        Sidebar.tsx

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
        BranchSwitcher.tsx

      health/
        HealthBadge.tsx

      actions/
        ActionList.tsx

    hooks/
      useServices.ts
      useWorkspace.ts
      useLogs.ts
      useRealtime.ts
```

---

# 12) Main UI layout

## Left sidebar
- workspace name
- presets list
- maybe filter by group/tag later

## Top bar
- Start All / Stop All
- preset actions
- search / command palette later
- refresh button

## Main center
- service cards grid

## Right detail panel
When a service is selected, show tabs:
- Logs
- Git
- Info
- Actions

---

# 13) Service card design

Each service card should show:
- service name
- status badge
- branch name
- dirty badge if repo changed
- port
- worker status if exists
- quick buttons:
  - Start / Stop / Restart
  - Logs
  - Open browser
  - Open repo
  - More actions

Status colors:
- green = running
- yellow = starting
- red = failed
- gray = stopped

---

# 14) Detail panel design

## Logs tab
- live log stream
- filter chips: all / error / warn
- search input
- copy logs
- follow tail toggle

## Git tab
- branch
- dirty state
- fetch / pull / push
- checkout branch input/select

## Info tab
- repo path
- command
- port
- dependencies
- health state

## Actions tab
- custom action buttons:
  - run tests
  - db migrate
  - install deps
  - open terminal

---

# 15) Config format proposal

Use **YAML** for local config.

```yaml
name: mealsuite

presets:
  - id: core
    name: Core
    services: [core-app, core-be]

  - id: touch
    name: Touch
    services: [touch-app, touch-be]

services:
  - id: core-app
    name: Core App
    type: frontend
    path: /Users/harryta/Desktop/code/core-app
    startCommand: ["npm", "run", "local"]
    port: 8080
    openUrls:
      - "http://localhost:8080"
    dependsOn:
      - core-be
    healthChecks:
      - type: tcp
        port: 8080
        interval: 3
    actions:
      - id: install
        name: Install deps
        command: ["npm", "install"]

  - id: core-be
    name: Core BE
    type: rails
    path: /Users/harryta/Desktop/code/core-be
    startCommand: ["bundle", "exec", "rails", "s"]
    port: 3000
    healthChecks:
      - type: tcp
        port: 3000
        interval: 3
    workers:
      - id: sidekiq
        name: Sidekiq
        startCommand: ["bundle", "exec", "sidekiq"]
    actions:
      - id: migrate
        name: DB Migrate
        command: ["bundle", "exec", "rails", "db:migrate"]
```

---

# 16) Migration plan from `appctl2.py`

Do **not** port file-by-file. Migrate **by capability**.

---

# 17) Phase-by-phase build plan

## Phase 0 — define scope and extract current behavior
### Goal
Understand exactly what `appctl2.py` already does and what should survive.

### Tasks
- list all current commands/features in `appctl2.py`
- identify current app/service config shape
- identify tmux-dependent behaviors
- decide which features are v1 vs later
- define the new config schema

### Deliverables
- v2 feature checklist
- YAML config schema
- service/preset list from current projects

---

## Phase 1 — backend foundation in Go
### Goal
Build the runtime engine with no fancy UI yet.

### Tasks
1. **Project scaffolding**
   - create Go module
   - choose router (`chi` recommended)
   - add config loader
   - add logger
   - add basic app bootstrap

2. **Config module**
   - load YAML config
   - validate service IDs / preset references
   - normalize paths

3. **Runtime manager**
   - service registry in memory
   - start/stop/restart process
   - track state
   - attach stdout/stderr readers

4. **Log manager**
   - ring buffer per service
   - append log lines
   - fetch recent logs

5. **Basic API**
   - `GET /services`
   - `POST /services/:id/start`
   - `POST /services/:id/stop`
   - `GET /services/:id/logs`

### Deliverable
You can start/stop a service and fetch its logs through API.

---

## Phase 2 — real-time events + health
### Goal
Make the backend usable as a live control plane.

### Tasks
- WebSocket endpoint
- emit service state updates
- emit log updates
- implement TCP/HTTP health checks
- track failed starts / process exits
- keep last error / exit code

### Deliverable
Frontend can receive live service status + logs without polling every second.

---

## Phase 3 — React dashboard MVP
### Goal
Replace Textual with a useful modern UI.

### Tasks
#### Setup
- Vite + React + TS
- Tailwind + shadcn/ui
- API client
- TanStack Query
- WebSocket hook

#### Build UI
- app shell
- service cards grid
- service detail panel
- logs viewer
- preset action bar
- basic top bar

### Deliverable
A usable dashboard where you can:
- start/stop services
- inspect logs
- view status

---

## Phase 4 — git actions + workspace actions
### Goal
Bring back the high-value dev actions from `appctl2.py`.

### Tasks
#### Backend
- git manager
- branch lookup
- fetch/pull/push
- checkout branch
- dirty status refresh

#### Frontend
- Git tab in service panel
- branch display on service cards
- checkout UI
- action buttons

### Deliverable
You can do common git flows from the UI.

---

## Phase 5 — workers + custom actions
### Goal
Support Sidekiq and common project commands.

### Tasks
#### Backend
- worker lifecycle per service
- action runner for custom commands
- action result reporting

#### Frontend
- worker status controls
- actions tab
- run custom tasks

### Deliverable
You can start Sidekiq, run migration, install deps, etc.

---

## Phase 6 — polish + migration away from tmux dependency
### Goal
Make it feel like a proper daily-use tool.

### Tasks
- open browser / open repo / open terminal actions
- better error display
- log filters and copy UX
- preserve recent logs after service stop
- dependency-aware preset startup
- optional tmux “open shell” action only

### Deliverable
This becomes the actual replacement for `appctl2.py`.

---

# 18) Suggested implementation order inside each phase

## Backend order
1. config loader
2. service runtime start/stop
3. log capture
4. service state API
5. WebSocket events
6. health checks
7. git actions
8. workers
9. custom actions

## Frontend order
1. dashboard layout
2. service cards
3. logs panel
4. preset actions
5. detail panel
6. git panel
7. actions panel
8. polish / filters / copy UX

---

# 19) Things intentionally out of scope at first

- full tmux integration as primary runtime
- terminal emulator inside the app
- SQLite persistence for everything
- plugin system
- drag-and-drop config builder
- Docker/Kubernetes orchestration UI
- team sync / cloud sync
- advanced analytics around logs

---

# 20) Risks / decisions to settle early

## A. Will Go own processes directly?
**Yes.**

## B. Will logs come from stdout/stderr or tmux pane capture?
**stdout/stderr directly.**

## C. Will git refresh be event-driven or polled?
Mostly **on-demand + slow background refresh**.

## D. Will the first version be browser-based or desktop-wrapped?
**Browser-based local app first.**

## E. Will config be user-editable in UI initially?
**No. Start with YAML config file.**

---

# 21) Recommended 2-week build slice

## Week 1
### Backend
- config loader
- service runtime manager
- start/stop/restart
- logs buffer
- `/services`, `/start`, `/stop`, `/logs`
- websocket events

### Frontend
- app shell
- service grid
- logs panel
- start/stop buttons
- live status updates

## Week 2
### Backend
- health checks
- git current branch
- fetch/pull/checkout
- preset start/stop
- worker support

### Frontend
- service detail panel
- preset bar
- git panel
- worker actions
- open browser / open repo actions

This should be enough for a usable internal alpha.

---

# 22) Build strategy recommendation

1. Use `appctl2.py` as **feature reference only**, not architecture reference.
2. Define the **YAML config schema** first.
3. Build the Go backend around **service runtime + logs + presets** before git niceties.
4. Build the React dashboard once the backend can:
   - list services
   - start/stop them
   - stream logs
   - report status
5. Then add git, workers, and convenience actions.

---

# 23) Concise recommendation

Build a **local browser-based developer control app** with:
- **Go backend** as the runtime/orchestration engine
- **React + Vite + Tailwind + shadcn** frontend
- **YAML workspace config**
- **WebSocket live events**
- **direct process/log management**
- **tmux only as an optional escape hatch, not the core runtime**

Ship in this order:
1. **service runtime + logs**
2. **dashboard UI**
3. **presets + health**
4. **git actions**
5. **workers + custom actions**
6. **polish + optional Tauri wrapper later**
