# devctl — Dev Control App

A local developer control center for managing multiple dev projects from one place: start/stop/restart services, stream live logs, run git actions, presets, and workers — all from a single dashboard instead of juggling tmux panes.

Rebuild of an earlier `appctl2.py` (Textual/tmux) tool, with a cleaner architecture:

- **Backend:** Go — owns process lifecycle, log capture, and state directly (no tmux)
- **Frontend:** React + TypeScript + Vite + Tailwind + shadcn/ui
- **Transport:** HTTP for commands/snapshots, WebSocket for live updates
- **Config:** a single YAML file describes your workspace — services, ports, commands, presets

See `docs/ARCHITECTURE.md` and `docs/SPEC.md` for the full design rationale and spec.

## Status

**v1 complete** (see `docs/DELTA_PLAN.md`). Working:

- Config-driven service list (YAML)
- Start / stop / restart / force-kill services, Stop All
- Live log streaming per service (WebSocket), with search/filter/copy/clear/follow-tail toolbar
- Dashboard: service grid, status badges, per-service detail panel (Logs / Git / Workers / Actions / Info tabs)
- Presets — start/stop a group of services at once, respecting dependency order
- Health checks — port health checks, startup failure detection, last-error surfacing
- Git actions — current branch, dirty/clean state, fetch, pull, push, searchable branch checkout, create branch from any existing branch
- Workers (e.g. Sidekiq) as first-class runtime units, with `autoStart` tied to their parent service
- Custom actions (db migrate, install deps, run tests, etc.) with streamed output
- Open helpers — open browser, open repo, open terminal for a service
- Single-binary packaging — `make build` embeds the frontend into the Go binary, no dev servers needed to just run it

Anything past this — Docker orchestration UI, plugin system, multi-user/team
sync, etc. — is explicitly out of scope per `docs/Plan.md` §5, not a gap.
See `docs/SMOKE_TEST.md` for the manual checklist this has been verified
against.

## Requirements

- Go 1.23+
- Node.js + npm

## Setup

1. **Configure your workspace.** Copy the example config and edit it with your real services:

   ```bash
   cp devctl.example.yaml backend/devctl.yaml
   ```

   Edit `backend/devctl.yaml`: set each service's `id`, `path` (absolute path to the repo), `startCommand`, and `port`. This file is gitignored — it's your local machine's real paths, not something to commit.

2. **Install frontend dependencies:**

   ```bash
   cd frontend
   npm install
   ```

## Running

### Just want to use it

Build a single binary (frontend gets embedded into it) and run that — no separate dev servers, no two terminals:

```bash
make build
./bin/devctl --config backend/devctl.yaml
```

Listens on `http://127.0.0.1:4312` and serves the dashboard itself — open that URL in your browser.

### Developing devctl itself

The build above bakes in whatever the frontend looked like at build time, so it's not what you want while iterating on devctl's own code. Run the backend and frontend as two separate dev servers instead:

**Backend** (from `backend/`):

```bash
go run ./cmd/devctl --config devctl.yaml
```

Listens on `http://127.0.0.1:4312`.

**Frontend** (from `frontend/`):

```bash
npm run dev
```

Serves the dashboard at `http://localhost:5173` (Vite dev server proxies `/api` and `/ws` to the backend). Open that URL in your browser — not port 4312, which in this mode has no embedded frontend to serve.

## Using the dashboard

- Each configured service shows as a card: name, status (stopped / starting / running / failed / stopping), health badge, port, and current git branch.
- **Start / Stop / Restart / Force-kill** buttons on each card control that service's process directly. **Stop All** in the top bar tears down every running service.
- Presets in the top bar start/stop a whole group of services at once, in dependency order.
- Click a card to select it and open the detail panel on the right:
  - **Logs** tab — live-streamed stdout/stderr, with search/filter/copy/clear and follow-tail controls.
  - **Git** tab — branch, dirty/clean state, ahead/behind, fetch/pull/push, searchable branch checkout.
  - **Workers** tab — start/stop worker processes (e.g. Sidekiq) tied to this service; `autoStart` workers come up/down with their parent.
  - **Actions** tab — run config-defined custom commands (db migrate, install deps, tests, ...) and stream their output; open-browser/open-repo/open-terminal helpers.
  - **Info** tab — type, path, port, dependencies, PID, last exit code / last error.
- The top bar shows a running/failed/total count across all services.

Everything updates live over the WebSocket connection — no manual refresh needed.

## Project layout

```
backend/            Go backend (runtime engine)
  cmd/devctl/        entrypoint
  internal/
    config/          YAML schema, loading, validation
    workspace/        read-only access to config-defined services/presets
    runtime/          process lifecycle, state machine, events
    logs/             in-memory ring-buffer log storage
    git/              branch/status/fetch/pull/push/checkout
    health/           port health checks, startup failure detection
    actions/          config-defined custom command execution
    openhelpers/       open browser/repo/terminal
    api/              HTTP routes + WebSocket hub
    app/              composition root
    applog/           internal backend logging

frontend/            React dashboard
  src/
    components/       layout, workspace (grid/cards/detail panel), logs, common UI
    hooks/            data-fetching + realtime hooks
    lib/              API client, WebSocket client
    pages/            DashboardPage

docs/                Architecture, spec, task backlog, build plan
devctl.example.yaml   Config template (copy to backend/devctl.yaml)
```

## Tests

```bash
cd backend
go test ./... -race
```
