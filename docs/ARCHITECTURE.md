# Dev Control App V2 — ARCHITECTURE.md

## 1) Purpose

This document explains the **high-level architecture** for the Dev Control App v2 rebuild.

The goal is to replace the current `appctl2.py` style tool with a cleaner system that is:

- **macOS-first**
- **lightweight**
- **good-looking and practical for daily dev work**
- **better structured than a Python/TUI/tmux-heavy setup**
- **easy to evolve without turning into a giant Electron app**

This file sits above `SPEC.md`.  
Think of it as the **why + system shape + architectural decisions** document.

---

# 2) Product goal

Build a local developer control app that can manage multiple dev projects from one place:

- start / stop / restart FE and BE services
- start Sidekiq or other workers
- stream and inspect logs
- run git actions
- open frontend URLs in browser
- open repo folders / terminal / editor
- run project-specific custom actions
- launch service groups / presets for common workflows

The app should feel like a **local control center for a multi-repo dev setup**.

---

# 3) Core product constraints

These constraints drive the architecture.

## 3.1 Must feel lighter than Electron-first desktop apps
The current problem is not just “missing features”; it’s also that many solutions feel too heavy for a local dev utility.

So the app should:
- avoid a giant desktop shell unless there’s a clear benefit
- keep memory and CPU reasonable
- use simple runtime primitives where possible

## 3.2 Must have a much better UI than the current Python/TUI tool
The current `appctl2.py` / TUI-style approach is practical but limited:
- layout flexibility is constrained
- logs and service detail panels can feel cramped
- richer interactions are harder to do well
- polishing UX takes more effort than in a browser UI stack

So the new architecture should prioritize a **web-style UI**.

## 3.3 Must be good at local process orchestration
This app is not just a launcher.  
It needs to own the lifecycle of local development processes and reflect their state correctly.

That means:
- starting processes directly
- tracking process state
- capturing logs
- handling stop/restart properly
- supporting workers as first-class runtime units

## 3.4 Must be config-driven
The app should not require code changes every time a service changes.

A service, preset, Sidekiq worker, FE URL, or git action should be configurable in YAML.

---

# 4) Chosen architecture in one sentence

## **Go local backend for runtime/process orchestration + React frontend for the control UI, communicating over local HTTP/WebSocket.**

That is the center of the design.

---

# 5) Why this architecture

# 5.1 Why not keep it as Python + TUI
Python + Textual/TUI is still a valid route, but it has tradeoffs for this app:

## Pros of Python/TUI
- fast to hack on
- single-language simplicity
- lower initial setup cost
- easy to shell out to commands

## Why it’s not the best fit here
For *this* tool, the main pain is no longer “can I automate shell commands?”  
It’s more about **UX, architecture, and maintainability**.

You want:
- a nicer dashboard
- better detail panels
- cleaner log UX
- easier growth into presets, git controls, actions, health checks, etc.

A browser UI with React is simply better suited for that layer.

---

# 5.2 Why not Electron as the main shell
Electron would absolutely work, but it’s not the best default for a “local dev control utility” if the priority is staying light.

## Electron advantages
- polished desktop packaging
- one app bundle
- access to native shell APIs
- familiar for web devs

## Why it’s not the default choice here
- extra memory footprint
- more desktop-shell complexity than needed for v1
- the app doesn’t need heavy native UI features at the start
- a local web app already gives most of the UX benefit

A local React UI served by a Go backend gets you **most of the UI quality** without committing to Electron up front.

If later you want packaging, you can still wrap it later with:
- Wails
- Tauri
- Electron
- native packaging scripts

without changing the core architecture much.

---

# 5.3 Why Go for backend/runtime instead of Python
The backend here is not just “API glue”. It’s the runtime engine.

It needs to:
- own child processes
- manage state transitions
- stream logs
- run health checks
- coordinate workers
- handle multiple concurrent services cleanly

Go is a strong fit for that because it gives:
- lightweight concurrency
- easy process management via stdlib
- low memory overhead compared to desktop-shell-heavy options
- easy distribution as a single binary
- good long-term maintainability for a local systems-ish tool

Python can do this too, but Go feels better if the tool is going to become a durable internal app rather than a flexible script.

---

# 5.4 Why React for frontend
The frontend problem is mostly about **control panel UX**:
- cards
- split panels
- logs
- filters
- badges
- actions
- stateful detail views

React + TypeScript gives:
- very fast UI iteration
- rich component ecosystem
- much better layout/control than TUI
- easier polished UX for logs, tabs, filters, status badges, etc.

This is the part of the stack that should optimize for developer experience and interface quality.

---

# 6) Architectural principles

These principles matter more than any specific package choice.

# 6.1 Backend owns runtime truth
The Go backend is the source of truth for:
- whether a service is running
- the PID of the running process
- worker state
- latest health status
- latest git state
- log buffers

The frontend should not infer service state from UI assumptions.

---

# 6.2 Direct process ownership beats “tmux as the runtime”
The app should start services directly using OS processes, not delegate runtime ownership to tmux.

tmux can still exist as an optional helper if you want it later, but it should not be the foundation.

Why:
- clearer state model
- cleaner stop/restart logic
- easier log capture
- fewer edge cases around detached sessions
- backend can truly know what it owns

---

# 6.3 UI should be event-driven, not polling-heavy
The frontend should not poll every second for logs and status if avoidable.

Use:
- HTTP for initial reads
- WebSocket for live updates

That keeps the UI responsive without building a wasteful polling loop.

---

# 6.4 Config defines workspace shape
The workspace should be described in YAML:
- services
- workers
- actions
- presets
- health checks
- open URLs
- dependencies

This makes the app adaptable to new projects without code edits.

---

# 6.5 Start with local-only trust model
This is a local dev utility, not a multi-tenant SaaS.

So the app can assume:
- trusted user
- localhost-only server
- config-defined actions are acceptable

That lets the first version stay simple without auth, RBAC, or a heavy persistence layer.

---

# 7) System context

```txt
+--------------------------------------------------------------+
|                        Developer                             |
|                                                              |
|   interacts with local dashboard in browser                  |
+------------------------------+-------------------------------+
                               |
                               v
+--------------------------------------------------------------+
|                    React Frontend UI                         |
|                                                              |
|  - service grid                                              |
|  - detail panel                                              |
|  - logs viewer                                               |
|  - git controls                                              |
|  - worker controls                                           |
|  - preset launcher                                           |
+------------------------------+-------------------------------+
                               |
                    HTTP / WebSocket on localhost
                               |
                               v
+--------------------------------------------------------------+
|                     Go Backend Runtime                       |
|                                                              |
|  - config loader                                             |
|  - workspace service                                         |
|  - runtime manager                                           |
|  - log manager                                               |
|  - git service                                               |
|  - health service                                            |
|  - action runner                                             |
|  - open helpers                                              |
+------------------------------+-------------------------------+
                               |
                               v
+--------------------------------------------------------------+
|                  Local OS / Dev Environment                  |
|                                                              |
|  - FE dev servers                                            |
|  - Rails / BE servers                                        |
|  - Sidekiq workers                                           |
|  - git CLI                                                   |
|  - browser / terminal / editor open commands                 |
+--------------------------------------------------------------+
```

---

# 8) Top-level subsystem architecture

The system is split into two main runtime pieces.

## 8.1 Frontend app
Responsibilities:
- render the dashboard
- trigger user actions
- show live service state
- show logs and detail views
- display health / git / worker info

The frontend should be **thin in terms of business logic**.  
It is mostly a control surface for the backend.

## 8.2 Backend app
Responsibilities:
- load workspace config
- own service lifecycle
- capture logs
- expose APIs
- push realtime updates
- run helper actions

The backend is the operational core.

---

# 9) Backend module architecture

The backend should be split into clear modules rather than one giant “manager” package.

## 9.1 App / composition layer
Responsible for:
- bootstrapping dependencies
- wiring services together
- starting HTTP server
- graceful shutdown

This is the composition root.

## 9.2 Config module
Responsible for:
- parsing YAML
- applying defaults
- validating schema
- expanding paths

This module defines the static workspace shape.

## 9.3 Workspace module
Responsible for:
- providing read access to config-defined services and presets
- mapping service IDs to definitions
- acting as the “catalog” of the workspace

This module does not own runtime state.

## 9.4 Runtime module
Responsible for:
- starting/stopping services
- tracking service state
- tracking worker state
- reacting to process exits
- coordinating log capture for managed processes

This is the most important module in the system.

## 9.5 Logs module
Responsible for:
- holding recent logs in memory
- appending stdout/stderr lines
- returning recent logs to API callers
- feeding log events to the WebSocket layer

## 9.6 Git module
Responsible for:
- reading branch / dirty state
- executing fetch/pull/push/checkout
- refreshing git state when needed

## 9.7 Health module
Responsible for:
- TCP/HTTP checks
- per-service monitoring while running
- exposing latest health state
- pushing health updates

## 9.8 Actions module
Responsible for:
- running config-defined one-off commands
- streaming action output
- keeping action runs separate from service runtime state

## 9.9 API / transport module
Responsible for:
- HTTP routing
- request/response DTOs
- WebSocket hub
- translating backend services into API endpoints

---

# 10) Frontend module architecture

The frontend should also avoid becoming a blob.

## 10.1 App shell / layout
Owns:
- page shell
- layout regions
- selected service panel container

## 10.2 Workspace / service views
Owns:
- service grid
- service cards
- preset bar
- detail panel

## 10.3 Logs UI
Owns:
- log viewer
- search/filter controls
- copy/follow-tail behavior

## 10.4 Git UI
Owns:
- git actions
- branch display
- checkout UI

## 10.5 Shared data layer
Owns:
- API client
- React Query hooks
- WebSocket event handling
- local selected-service state

---

# 11) Data flow architecture

# 11.1 Initial page load flow

```txt
Frontend loads
 -> GET /api/workspace
 -> GET /api/services
 -> render service cards + presets
 -> open WebSocket connection
```

## Why this split
- HTTP is good for initial snapshots
- WebSocket is good for incremental live updates

---

# 11.2 Start service flow

```txt
User clicks Start on a service
 -> Frontend POST /api/services/:id/start
 -> Backend runtime manager validates state
 -> Backend starts process
 -> Backend updates service state
 -> Backend emits service.updated
 -> Backend captures stdout/stderr
 -> Backend emits log.appended events as logs arrive
 -> Frontend updates service card + log panel
```

---

# 11.3 Stop service flow

```txt
User clicks Stop
 -> Frontend POST /api/services/:id/stop
 -> Backend runtime manager sends termination signal
 -> Backend updates service state to stopping/stopped
 -> Backend emits service.updated
 -> workers are stopped if applicable
```

---

# 11.4 Run preset flow

```txt
User clicks Start Core preset
 -> Frontend POST /api/presets/core/start
 -> Backend resolves service list
 -> Backend sorts by dependency order if available
 -> Backend starts each service
 -> service.updated events stream as services change state
 -> logs begin flowing for each started service
```

---

# 11.5 Git action flow

```txt
User clicks Pull
 -> Frontend POST /api/services/:id/git/pull
 -> Backend executes git pull in repo path
 -> Backend refreshes git state
 -> Backend emits git.updated
 -> Frontend updates branch/dirty/ahead/behind UI
```

---

# 11.6 Worker flow

```txt
User clicks Start Sidekiq
 -> Frontend POST /api/services/:id/workers/sidekiq/start
 -> Backend starts worker process
 -> Backend tracks worker state
 -> worker.updated event emitted
 -> worker logs stream to worker log stream
```

---

# 12) Runtime ownership model

This is one of the most important architecture decisions.

# 12.1 Service runtime units
Each service is a runtime unit with:
- config definition
- current status
- PID if running
- stdout/stderr readers
- health state
- git state snapshot
- child worker states

## Conceptually:
```txt
ServiceDefinition + RuntimeState + ProcessHandle + LogStream + WorkerMap
```

## The runtime manager owns these instances in memory.

---

# 12.2 Worker runtime units
Workers are attached to a parent service but are independently controllable.

Example:
- `core-be`
  - service process = Rails server
  - worker process = Sidekiq

That means worker state should not be hacked into “extra buttons”.  
It should be a first-class runtime concept.

---

# 12.3 Action runs are separate from service runtime
A custom action like “db migrate” or “npm install” should not be treated as the service process itself.

It should be its own one-off execution flow with:
- a run ID
- output stream
- success/failure result

This prevents action execution from polluting service lifecycle logic.

---

# 13) State architecture

## 13.1 Static state
Comes from config:
- service definitions
- presets
- workers
- actions
- URLs
- health checks

This changes only when config is reloaded.

## 13.2 Dynamic runtime state
Changes while the app runs:
- service status
- PID
- last error
- health
- git snapshot
- logs
- worker status

This is owned in memory by the backend.

## 13.3 UI state
Frontend-only concerns:
- selected service
- active detail tab
- log filter text
- follow tail enabled
- modal visibility

This belongs in frontend state, not backend.

---

# 14) Communication architecture

# 14.1 HTTP for commands + snapshots
Use HTTP for:
- initial data fetches
- command-style actions
- one-off operations

Examples:
- list services
- start/stop service
- fetch recent logs
- run git pull
- run preset start

## Why
HTTP is easy for command-response flows and plays well with React Query.

---

# 14.2 WebSocket for realtime state changes
Use WebSocket for:
- service state updates
- log appends
- worker updates
- health updates
- git updates
- action output

## Why
This avoids wasteful polling and makes the dashboard feel live.

---

# 15) Persistence architecture

# 15.1 What should persist in v1
Minimal persistence only:
- YAML config file
- maybe a tiny frontend preference store later (selected theme, preferred view)

## 15.2 What should not persist in v1
Do not rush into a database for:
- service history
- full log history
- long-term analytics

For the first version, logs can stay in memory.

That keeps the architecture much simpler.

---

# 16) Logging architecture

## 16.1 Log source
Logs come from stdout/stderr of child processes started by the backend.

## 16.2 Log storage strategy
Use in-memory ring buffers per stream:
- one stream per service
- one stream per worker
- one stream per action run if needed

## 16.3 Why ring buffers
You want:
- recent logs quickly available
- bounded memory
- no need for database persistence

This is a good fit for a local dev utility.

---

# 17) Health architecture

Health checks are not the same thing as process running state.

A service can:
- have a running process but still be unhealthy
- be “starting” before port is ready
- be alive but failing HTTP health checks

So health must be modeled separately from runtime state, even though it’s attached to a service.

---

# 18) Git architecture

Git operations should be treated as **service-scoped repo actions**.

The Git module should know:
- which repo path belongs to a service
- how to read branch / dirty state
- how to execute fetch/pull/push/checkout

Git should not live inside the runtime manager because it’s a different concern.

---

# 19) Open helper architecture

Opening a browser, folder, or terminal is not “service runtime”, but it is still service-adjacent behavior.

So these actions belong in a small helper layer:
- open FE URL
- open repo folder
- open terminal at repo path
- maybe open editor later

These are thin OS integration helpers, not a core domain.

---

# 20) Dependency architecture

A service may depend on another service, but dependencies should be handled carefully.

## Initial stance
- **Preset startup** should respect dependency order
- **Individual service start** should not silently auto-start everything unless explicitly enabled later

Why:
- more predictable
- avoids surprising side effects
- keeps the first version easier to reason about

---

# 21) Failure handling architecture

# 21.1 Service start failure
If process start fails:
- mark service failed
- store error message
- emit service.updated

## 21.2 Service crash after successful start
If process exits unexpectedly:
- capture exit code
- mark service failed or stopped depending on context
- emit service.updated

## 21.3 Worker failure
Worker failures should update only that worker’s state, unless later you decide certain workers are critical to service health.

## 21.4 Git/action failure
Git and action failures should not crash the backend or corrupt service runtime state.
They should surface as command failures + UI messages.

---

# 22) Performance architecture

The goal is not “micro-optimized”, but it should stay light.

## 22.1 Backend performance rules
- only monitor health for running services
- keep log buffers bounded
- avoid constant git polling
- avoid unnecessary goroutine explosions
- avoid shelling out on every UI paint

## 22.2 Frontend performance rules
- do not rerender the whole dashboard for every log line
- keep log updates localized to log viewer state
- use virtualization if logs get large
- use React Query for server snapshots, not custom chaos state everywhere

---

# 23) Security model

This is intentionally simple for v1.

## Assumptions
- single local user
- localhost-only
- trusted config
- trusted repos and scripts

## Guardrails still worth having
- bind backend to `127.0.0.1`
- do not expose arbitrary shell command execution from raw text inputs
- keep actions config-defined for MVP
- validate service paths and commands at startup

---

# 24) Packaging / deployment architecture

# 24.1 First delivery model
The simplest development model is:

- run Go backend locally
- run React frontend locally
- open browser to the dashboard

## Optional production-ish local packaging later
Later you can:
- build frontend static assets
- embed them in Go binary
- ship a single local binary

This is a very good second step because it keeps the architecture the same while improving convenience.

---

# 24.2 Possible future desktop wrappers
If you later want a “real desktop app” feel:
- Wails
- Tauri
- Electron

The key is that the **Go backend + React UI split still survives**.  
Only the shell changes.

---

# 25) Why this architecture is a good fit for your use case specifically

Based on the kind of tool you described, the important parts are:

- controlling multiple FE/BE projects
- seeing logs fast
- starting Sidekiq/workers
- opening FE/browser
- running git actions
- keeping the UI nicer than the current TUI app
- not turning it into a bloated app

This architecture fits because it separates the two jobs cleanly:

## Go handles the “ops/control plane” part
- processes
- workers
- logs
- health
- git
- orchestration

## React handles the “daily driver UI” part
- dashboard layout
- service cards
- log panel
- tabs and actions
- UX polish

That split is exactly what your current Python/TUI version struggles to do elegantly in one place.

---

# 26) Architecture decision summary

## ADR-01 — Use Go backend + React frontend
**Decision:** Yes  
**Reason:** best balance of lightweight runtime + good UI

## ADR-02 — Backend owns processes directly
**Decision:** Yes  
**Reason:** clearer runtime state and easier lifecycle control than tmux-owned processes

## ADR-03 — Use YAML as workspace config
**Decision:** Yes  
**Reason:** avoids code edits for service changes

## ADR-04 — Use WebSocket for live updates
**Decision:** Yes  
**Reason:** log streaming and status changes need realtime UX without polling spam

## ADR-05 — Keep persistence minimal in v1
**Decision:** Yes  
**Reason:** the tool is local and operational, not a historical data system

## ADR-06 — macOS-first design
**Decision:** Yes  
**Reason:** matches your immediate environment and reduces v1 complexity

---

# 27) Recommended implementation philosophy

Don’t try to build the perfect platform first.

Build in this order:

## Phase 1 — replace the pain
- config loader
- service runtime manager
- logs
- service dashboard
- start/stop/restart
- live updates

## Phase 2 — become the daily driver
- presets
- health
- git actions
- Sidekiq/workers

## Phase 3 — make it pleasant
- actions panel
- open helpers
- log search/filter/copy
- better error surfaces
- polish

That order matters because the app becomes useful much earlier.

---

# 28) Final architecture statement

The architecture should be treated as:

## **A local developer control plane with a Go runtime core and a React operations dashboard.**

Not:
- a fancy tmux wrapper
- a terminal app with extra buttons
- a bloated desktop shell first
- a generic platform trying to solve every devops problem

If you keep the scope centered on **“daily local multi-project dev control with a clean UI”**, this architecture is the right level of power without becoming overbuilt.
