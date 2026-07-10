# Dev Control App V2 — TASKS.md

## Purpose

This file breaks the v2 rebuild into implementation-ready tasks with:

- priority
- effort estimate
- dependencies
- recommended implementation order
- suggested sprint grouping

The goal is to take the app from the current `appctl2.py` prototype to a usable **Go + React internal alpha**, then to a **daily-driver beta**.

---

# 1) Delivery strategy

## Milestone A — Backend foundation
You can:
- load config
- list services
- start/stop/restart services
- capture logs
- expose basic API

## Milestone B — Frontend alpha
You can:
- open dashboard
- see service cards
- start/stop services
- inspect live logs

## Milestone C — Daily-driver beta
You can:
- start presets
- run workers / Sidekiq
- run git actions
- use health checks
- run custom actions
- open FE/browser/repo

---

# 2) Estimation scale

## Effort estimate legend
- **XS** = < 0.5 day
- **S** = 0.5–1 day
- **M** = 1–2 days
- **L** = 2–4 days
- **XL** = 4+ days

## Priority legend
- **P0** = required for MVP / blocks other work
- **P1** = high value, should land in first usable version
- **P2** = important polish / beta feature
- **P3** = optional or later

---

# 3) Recommended implementation order

## Order of attack
1. Backend scaffolding + config
2. Runtime manager + process runner
3. Log capture + basic service APIs
4. WebSocket realtime events
5. Frontend shell + service grid + start/stop
6. Logs panel + live updates
7. Presets + health checks
8. Git actions
9. Workers / Sidekiq
10. Custom actions + open helpers
11. UX polish / dependency-aware startup

---

# 4) Sprint plan overview

## Sprint 0 — Scope + architecture setup
Goal: lock down schema, repo structure, backend skeleton

## Sprint 1 — Backend runtime alpha
Goal: start/stop services, logs, basic APIs

## Sprint 2 — Frontend alpha
Goal: usable dashboard with service control + logs

## Sprint 3 — Daily-driver beta core
Goal: presets, health, git, workers

## Sprint 4 — Productivity features + polish
Goal: actions, open helpers, UX improvements, cleanup

---

# 5) Task board

---

# Sprint 0 — Scope + architecture setup

## T-001 — Audit current `appctl2.py` features
- **Priority:** P0
- **Estimate:** S
- **Depends on:** none

### Goal
Create a clean inventory of what the current prototype already supports.

### Deliverables
- feature list
- current project/service list
- current start commands
- current Sidekiq/worker behavior
- current tmux-dependent behaviors
- current git/browser/open actions

### Notes
Do not port code line-for-line. This is just feature extraction.

---

## T-002 — Finalize v2 config schema
- **Priority:** P0
- **Estimate:** M
- **Depends on:** T-001

### Goal
Lock the YAML structure for:
- workspace
- services
- presets
- workers
- actions
- health checks
- open URLs

### Deliverables
- `SPEC.md` schema aligned
- `devctl.example.yaml`
- validation rules list

---

## T-003 — Create monorepo/project structure
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-002

### Goal
Scaffold the repo for backend + frontend.

### Deliverables
```txt
backend/
frontend/
docs/
```

### Suggested folders
```txt
backend/cmd/devctl
backend/internal/api
backend/internal/config
backend/internal/runtime
backend/internal/logs
backend/internal/git
backend/internal/health
backend/internal/actions
backend/internal/workspace

frontend/src/app
frontend/src/pages
frontend/src/components
frontend/src/hooks
frontend/src/lib
frontend/src/types
```

---

## T-004 — Decide dev startup workflow
- **Priority:** P1
- **Estimate:** XS
- **Depends on:** T-003

### Goal
Define how the app itself is run during development.

### Suggested approach
- backend on `localhost:4312`
- frontend on `localhost:5173`
- Vite dev proxy to backend
- later: embed built frontend into Go binary if desired

---

# Sprint 1 — Backend runtime alpha

## T-010 — Initialize Go backend app bootstrap
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-003

### Goal
Create the main Go app entry and dependency container.

### Deliverables
- `main.go`
- app config loading
- app container struct
- HTTP server bootstrap
- graceful shutdown skeleton

---

## T-011 — Implement config loader
- **Priority:** P0
- **Estimate:** M
- **Depends on:** T-002, T-010

### Goal
Load YAML config into typed structs.

### Deliverables
- `config/schema.go`
- `config/loader.go`
- support `--config`
- support default config path
- support `~` expansion

---

## T-012 — Implement config validation
- **Priority:** P0
- **Estimate:** M
- **Depends on:** T-011

### Goal
Validate service/preset/action/worker definitions on startup.

### Validation rules
- duplicate service IDs
- duplicate preset IDs
- missing preset service refs
- missing dependency refs
- empty commands
- unsupported health check types

---

## T-013 — Implement workspace service
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-011, T-012

### Goal
Provide read access to config-defined workspace data.

### Deliverables
- `GetWorkspace`
- `ListServices`
- `GetService`
- `ListPresets`
- `GetPreset`

---

## T-014 — Implement process runner abstraction
- **Priority:** P0
- **Estimate:** M
- **Depends on:** T-010

### Goal
Create reusable start/stop process logic for services and workers.

### Deliverables
- `ProcessOptions`
- `RunningProcess`
- `Start`
- `Stop`
- env merge
- cwd support
- stdout/stderr pipes

---

## T-015 — Implement log ring buffer
- **Priority:** P0
- **Estimate:** M
- **Depends on:** T-010

### Goal
Store recent logs in memory per stream.

### Deliverables
- `LogEntry`
- `RingBuffer`
- append
- fetch recent entries
- max line cap

### Suggested default
- 3000 lines per stream

---

## T-016 — Implement logs manager
- **Priority:** P0
- **Estimate:** M
- **Depends on:** T-015

### Goal
Manage stream buffers and append/fetch operations.

### Deliverables
- get/create stream buffer
- append log entry
- list recent logs by stream
- stream naming convention

---

## T-017 — Implement runtime manager for services
- **Priority:** P0
- **Estimate:** XL
- **Depends on:** T-013, T-014, T-016

### Goal
Manage lifecycle of configured services.

### Deliverables
- in-memory service state registry
- `StartService`
- `StopService`
- `RestartService`
- service state transitions
- PID tracking
- process exit watcher
- update `LastError` / `LastExitCode`

---

## T-018 — Wire stdout/stderr log capture into runtime manager
- **Priority:** P0
- **Estimate:** M
- **Depends on:** T-016, T-017

### Goal
When a service starts, its output should flow into the log manager.

### Deliverables
- stdout reader goroutine
- stderr reader goroutine
- log entry append
- stream key `service:<serviceId>`

---

## T-019 — Implement basic service API handlers
- **Priority:** P0
- **Estimate:** M
- **Depends on:** T-017, T-018

### Endpoints
- `GET /api/workspace`
- `GET /api/services`
- `GET /api/services/:id`
- `POST /api/services/:id/start`
- `POST /api/services/:id/stop`
- `POST /api/services/:id/restart`

---

## T-020 — Implement logs API handler
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-016, T-019

### Endpoint
- `GET /api/services/:id/logs?limit=500`

---

## T-021 — Add consistent error response middleware/helpers
- **Priority:** P1
- **Estimate:** S
- **Depends on:** T-019

### Goal
Standardize API error responses.

### Deliverables
- error JSON helper
- common error codes
- status code mapping

---

## T-022 — Add backend integration test for runtime start/stop
- **Priority:** P1
- **Estimate:** M
- **Depends on:** T-017, T-018

### Goal
Verify service lifecycle works against a dummy command.

### Suggested dummy command
- shell script that prints a line every second and exits on signal

---

# Sprint 2 — Realtime + frontend alpha

## T-030 — Implement WebSocket hub
- **Priority:** P0
- **Estimate:** M
- **Depends on:** T-010

### Goal
Create a hub for broadcasting runtime/log/health/git events.

### Deliverables
- ws connection registry
- broadcast method
- graceful disconnect handling

---

## T-031 — Emit `service.updated` events
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-017, T-030

### Goal
Broadcast service state changes when services start/stop/fail.

---

## T-032 — Emit `log.appended` events
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-018, T-030

### Goal
Broadcast live log lines to the frontend.

---

## T-033 — Initialize React frontend app
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-003

### Deliverables
- Vite app
- TypeScript
- Tailwind
- shadcn/ui base
- app shell skeleton

---

## T-034 — Create frontend API client + DTO types
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-033, T-019

### Deliverables
- `getWorkspace`
- `getServices`
- `startService`
- `stopService`
- `restartService`
- `getServiceLogs`

---

## T-035 — Set up TanStack Query + app providers
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-033, T-034

---

## T-036 — Build dashboard page shell
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-033

### Layout
- sidebar
- top bar
- main service grid
- right detail panel placeholder

---

## T-037 — Build service card component
- **Priority:** P0
- **Estimate:** M
- **Depends on:** T-034, T-036

### Must show
- service name
- status
- branch placeholder
- port
- quick action buttons

---

## T-038 — Build service grid with `GET /api/services`
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-037

---

## T-039 — Hook service start/stop/restart actions into UI
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-038

---

## T-040 — Build service details panel shell
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-036

### Tabs
- Logs
- Git
- Info
- Actions

---

## T-041 — Build initial log viewer
- **Priority:** P0
- **Estimate:** M
- **Depends on:** T-020, T-040

### Features
- fetch recent logs for selected service
- scrollable panel
- monospace display
- timestamp + line

---

## T-042 — Implement frontend WebSocket client
- **Priority:** P0
- **Estimate:** M
- **Depends on:** T-030, T-031, T-032, T-033

### Deliverables
- connect to `/ws`
- parse event envelopes
- reconnect logic
- event dispatch

---

## T-043 — Apply `service.updated` events to service grid state
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-042

---

## T-044 — Apply `log.appended` events to log viewer
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-041, T-042

---

## T-045 — Add selected service state store
- **Priority:** P1
- **Estimate:** XS
- **Depends on:** T-040

### Goal
Keep track of the currently selected service for the detail panel.

---

## T-046 — Alpha smoke test
- **Priority:** P0
- **Estimate:** S
- **Depends on:** T-039, T-044

### Goal
Verify the first usable flow:
- load services
- start service
- see state change
- see logs stream
- stop service

---

# Sprint 3 — Daily-driver beta core

**Status: done.** T-050 → T-064, T-092, T-093 shipped — see docs/BETA_PLAN.md
for the implementation notes, the orphan-reconciliation decision, and one
addition beyond this original breakdown (worker `autoStart`).

## T-050 — Implement preset start/stop backend endpoints
- **Priority:** P1
- **Estimate:** M
- **Depends on:** T-013, T-017

### Endpoints
- `POST /api/presets/:id/start`
- `POST /api/presets/:id/stop`

---

## T-051 — Implement preset dependency resolution / ordering
- **Priority:** P1
- **Estimate:** M
- **Depends on:** T-050

### Goal
Sort services by dependencies before preset startup.

### MVP fallback
If cycle exists, fall back to config order and log warning.

---

## T-052 — Build preset bar UI
- **Priority:** P1
- **Estimate:** S
- **Depends on:** T-034, T-036, T-050

---

## T-053 — Implement health service
- **Priority:** P1
- **Estimate:** L
- **Depends on:** T-017

### Deliverables
- TCP health check
- HTTP health check
- start/stop monitoring per running service
- in-memory health state

---

## T-054 — Emit `health.updated` events
- **Priority:** P1
- **Estimate:** S
- **Depends on:** T-053, T-030

---

## T-055 — Show health badge in service cards/detail panel
- **Priority:** P1
- **Estimate:** S
- **Depends on:** T-054, T-037

---

## T-056 — Implement git service
- **Priority:** P1
- **Estimate:** L
- **Depends on:** T-013

### Functions
- get branch
- dirty status
- fetch
- pull
- push
- checkout

---

## T-057 — Add git endpoints
- **Priority:** P1
- **Estimate:** M
- **Depends on:** T-056

### Endpoints
- `POST /api/services/:id/git/fetch`
- `POST /api/services/:id/git/pull`
- `POST /api/services/:id/git/push`
- `POST /api/services/:id/git/checkout`

---

## T-058 — Refresh git state on service load / after actions
- **Priority:** P1
- **Estimate:** M
- **Depends on:** T-056, T-057

---

## T-059 — Build Git tab UI
- **Priority:** P1
- **Estimate:** M
- **Depends on:** T-057, T-040

### Features
- branch display
- dirty state
- fetch/pull/push buttons
- checkout input

---

## T-060 — Implement worker runtime manager support
- **Priority:** P1
- **Estimate:** L
- **Depends on:** T-014, T-016, T-017

### Deliverables
- worker runtime struct
- start worker
- stop worker
- worker state tracking
- worker log capture

---

## T-061 — Add worker endpoints
- **Priority:** P1
- **Estimate:** S
- **Depends on:** T-060

### Endpoints
- `POST /api/services/:id/workers/:workerId/start`
- `POST /api/services/:id/workers/:workerId/stop`

---

## T-062 — Emit `worker.updated` events
- **Priority:** P1
- **Estimate:** XS
- **Depends on:** T-060, T-030

---

## T-063 — Show worker controls in service detail panel
- **Priority:** P1
- **Estimate:** M
- **Depends on:** T-061, T-040

### Example
- Start Sidekiq
- Stop Sidekiq
- worker running badge

---

## T-064 — Beta smoke test
- **Priority:** P1
- **Estimate:** S
- **Depends on:** T-052, T-055, T-059, T-063

### Goal
Verify real daily-use flow:
- start preset
- FE + BE come up
- health changes visible
- git actions work
- Sidekiq starts/stops
- logs still behave correctly

---

# Sprint 4 — Productivity features + polish

**Status: mostly done.** T-070 → T-081 and T-094 shipped — see
docs/GAMMA_PLAN.md for implementation notes and two real bugs the live
smoke test caught. T-082 (dedicated visual polish) and T-095 (a written
smoke-test checklist) are not done.

## T-070 — Implement action runner
- **Priority:** P2
- **Estimate:** L
- **Depends on:** T-013, T-014, T-016

### Goal
Run config-defined custom actions.

### Examples
- install deps
- db migrate
- run tests

---

## T-071 — Add action endpoint
- **Priority:** P2
- **Estimate:** S
- **Depends on:** T-070

### Endpoint
- `POST /api/services/:id/actions/:actionId`

---

## T-072 — Stream action output to frontend
- **Priority:** P2
- **Estimate:** M
- **Depends on:** T-070, T-030

### Event
- `action.output`

---

## T-073 — Build Actions tab UI
- **Priority:** P2
- **Estimate:** M
- **Depends on:** T-071, T-072, T-040

---

## T-074 — Implement open browser helper
- **Priority:** P2
- **Estimate:** XS
- **Depends on:** T-013

### Endpoint
- `POST /api/services/:id/open-browser`

---

## T-075 — Implement open repo / open terminal helpers
- **Priority:** P2
- **Estimate:** S
- **Depends on:** T-013

### Endpoints
- `POST /api/services/:id/open-repo`
- `POST /api/services/:id/open-terminal`

---

## T-076 — Hook open actions into service card/detail panel
- **Priority:** P2
- **Estimate:** S
- **Depends on:** T-074, T-075, T-037, T-040

---

## T-077 — Add log toolbar features
- **Priority:** P2
- **Estimate:** M
- **Depends on:** T-041, T-044

### Features
- follow tail toggle
- clear visible logs
- copy visible logs
- severity filter
- search box

---

## T-078 — Add error highlighting in logs
- **Priority:** P2
- **Estimate:** S
- **Depends on:** T-015, T-041

---

## T-079 — Improve preset startup UX
- **Priority:** P2
- **Estimate:** M
- **Depends on:** T-051, T-052

### Ideas
- progress indicator
- show which service is currently starting
- partial failure summary

---

## T-080 — Add graceful stop on app shutdown
- **Priority:** P2
- **Estimate:** S
- **Depends on:** T-017, T-060

### Goal
When devctl exits, stop managed services optionally or preserve based on future config.

---

## T-081 — Add structured backend logging / diagnostics
- **Priority:** P2
- **Estimate:** S
- **Depends on:** T-010

---

## T-082 — Polish service card/info panel UX
- **Priority:** P2
- **Estimate:** M
- **Depends on:** T-055, T-059, T-063, T-076

### Ideas
- cleaner badges
- dependency list
- better spacing
- failure state summaries

---

# 6) Cross-cutting cleanup / quality tasks

## T-090 — Add unit tests for config validation
- **Priority:** P1
- **Estimate:** S
- **Depends on:** T-012

## T-091 — Add unit tests for ring buffer
- **Priority:** P1
- **Estimate:** S
- **Depends on:** T-015

## T-092 — Add unit tests for dependency ordering
- **Priority:** P1
- **Estimate:** S
- **Depends on:** T-051

## T-093 — Add unit tests for git service helpers
- **Priority:** P2
- **Estimate:** S
- **Depends on:** T-056

## T-094 — Add frontend component tests for service card + log viewer
- **Priority:** P2
- **Estimate:** M
- **Depends on:** T-037, T-041

## T-095 — Add end-to-end manual test checklist
- **Priority:** P1
- **Estimate:** XS
- **Depends on:** T-046, T-064

---

# 7) Suggested backlog by “must have first”

# Must-have for first usable alpha
These are the tasks I’d personally do first if the goal is “replace Textual ASAP”.

- T-001 Audit current features
- T-002 Finalize config schema
- T-003 Create monorepo/project structure
- T-010 Initialize Go backend
- T-011 Implement config loader
- T-012 Implement config validation
- T-013 Implement workspace service
- T-014 Implement process runner
- T-015 Implement log ring buffer
- T-016 Implement logs manager
- T-017 Implement runtime manager for services
- T-018 Wire stdout/stderr log capture
- T-019 Implement basic service API handlers
- T-020 Implement logs API handler
- T-030 Implement WebSocket hub
- T-031 Emit service.updated
- T-032 Emit log.appended
- T-033 Initialize React frontend
- T-034 Create API client + DTO types
- T-035 Set up TanStack Query
- T-036 Build dashboard shell
- T-037 Build service card
- T-038 Build service grid
- T-039 Hook start/stop/restart actions
- T-040 Build service detail panel shell
- T-041 Build initial log viewer
- T-042 Implement frontend WebSocket client
- T-043 Apply service.updated
- T-044 Apply log.appended
- T-046 Alpha smoke test

That’s the **smallest version I’d call worth using**.

---

# 8) Suggested backlog for first real beta
After alpha is stable, do these next:

- T-050 preset start/stop
- T-051 dependency ordering
- T-052 preset bar UI
- T-053 health service
- T-054 health.updated
- T-055 health badge UI
- T-056 git service
- T-057 git endpoints
- T-058 git refresh behavior
- T-059 Git tab UI
- T-060 worker runtime
- T-061 worker endpoints
- T-062 worker.updated
- T-063 worker controls
- T-064 Beta smoke test

This is the set that turns it into a **daily-use developer control tool** instead of just a pretty launcher.

---

# 9) Suggested backlog for polish / v1.1
- T-070 action runner
- T-071 action endpoint
- T-072 action output streaming
- T-073 Actions tab UI
- T-074 open browser helper
- T-075 open repo/open terminal
- T-076 wire open actions into UI
- T-077 log toolbar features
- T-078 error highlighting
- T-079 preset startup UX improvements
- T-080 graceful stop on app shutdown
- T-081 structured backend diagnostics
- T-082 service card/detail panel polish

---

# 10) My recommended execution plan if you want fastest value

If I were building this myself and wanted a good result without overengineering the first week, I’d do it in this exact order:

## Week 1
- T-001
- T-002
- T-003
- T-010
- T-011
- T-012
- T-013
- T-014
- T-015
- T-016
- T-017
- T-018
- T-019
- T-020

## Week 2
- T-030
- T-031
- T-032
- T-033
- T-034
- T-035
- T-036
- T-037
- T-038
- T-039
- T-040
- T-041
- T-042
- T-043
- T-044
- T-046

## Week 3
- T-050
- T-051
- T-052
- T-053
- T-054
- T-055
- T-056
- T-057
- T-058
- T-059

## Week 4
- T-060
- T-061
- T-062
- T-063
- T-064
- T-070
- T-071
- T-074
- T-075
- T-076
- T-077

---

# 11) Final stance

The app will feel dramatically better once these three things exist together:

1. **Go runtime manager** that owns services directly
2. **Live logs via WebSocket**
3. **React dashboard with service cards + detail panel**

Everything else is valuable, but those three are the actual turning point where this stops being “a Python script with UI” and becomes a proper dev control app.
