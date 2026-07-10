# devctl — Dev Control App

A local developer control center for managing multiple dev projects from one place: start/stop/restart services, stream live logs, and (soon) run git actions, presets, and workers — all from a single dashboard instead of juggling tmux panes.

Rebuild of an earlier `appctl2.py` (Textual/tmux) tool, with a cleaner architecture:

- **Backend:** Go — owns process lifecycle, log capture, and state directly (no tmux)
- **Frontend:** React + TypeScript + Vite + Tailwind + shadcn/ui
- **Transport:** HTTP for commands/snapshots, WebSocket for live updates
- **Config:** a single YAML file describes your workspace — services, ports, commands, presets

See `docs/ARCHITECTURE.md` and `docs/SPEC.md` for the full design rationale and spec.

## Status

**Alpha** (current). Working:

- Config-driven service list (YAML)
- Start / stop / restart services
- Live log streaming per service (WebSocket)
- Dashboard: service grid, status badges, per-service log viewer

Not yet built (planned next — see `docs/TASKS.md`):

- Presets (start/stop a group of services at once)
- Health checks
- Git actions (branch, pull, push, checkout)
- Workers (e.g. Sidekiq) as first-class runtime units
- Custom actions (db migrate, install deps, etc.) and open-browser/repo/terminal helpers

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

Run the backend and frontend in two terminals.

**Backend** (from `backend/`):

```bash
go run ./cmd/devctl --config devctl.yaml
```

Listens on `http://127.0.0.1:4312`.

**Frontend** (from `frontend/`):

```bash
npm run dev
```

Serves the dashboard at `http://localhost:5173` (Vite dev server proxies `/api` and `/ws` to the backend). Open that URL in your browser.

## Using the dashboard

- Each configured service shows as a card: name, status (stopped / starting / running / failed / stopping), port, and branch placeholder.
- **Start / Stop / Restart** buttons on each card control that service's process directly.
- Click a card to select it and open the detail panel on the right:
  - **Logs** tab — live-streamed stdout/stderr for that service, auto-scrolling.
  - **Info** tab — type, path, port, dependencies, PID, last exit code.
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
    api/              HTTP routes + WebSocket hub
    app/              composition root

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
