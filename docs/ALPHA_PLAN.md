# Dev Control App V2 — Alpha Implementation Plan

Scope: TASKS.md Milestone A + B (T-001 → T-046). Config-driven service runtime,
logs, live WebSocket updates, dashboard with start/stop/restart/logs. No
presets/health/git/workers/actions yet — that's the next round (beta).

Scaffolded directly in this folder (`/Users/harryta/Desktop/Dev/Tools/dev-controller`).

---

## Decisions locked

- **Git**: init now. `.gitignore` covers `node_modules/`, `frontend/dist/`,
  backend binary, `backend/devctl.yaml` (real local config, gitignored —
  `devctl.example.yaml` stays checked in as the template).
- **Go module**: `devctl` (no domain path). Router `chi`, WebSocket
  `gorilla/websocket`, YAML `gopkg.in/yaml.v3`.
- **Ports**: backend `:4312`, frontend Vite `:5173` with dev proxy for
  `/api` and `/ws`.
- **Frontend**: npm. No Zustand for alpha — selected-service state lifted in
  `DashboardPage`/`App` (single page, doesn't need a store yet; revisit if
  beta adds more cross-cutting UI state).
- `ServiceStateDTO.Git` / `.Health` present but stubbed
  (`branch: ""`, `health.status: "unknown"`) so the response shape doesn't
  break when beta's git/health modules land.
- **Backend tests**: stdlib `testing` only, no testify.
- **Log parsing / level-tagging** (`parser.go`, error highlighting) deferred
  to beta (T-078). Alpha `LogEntry.Level` stays empty.
- Added two tasks that TASKS.md backlogs but omits from its alpha summary
  bullet, because they directly de-risk what alpha builds:
  - **T-021** — consistent API error envelope (SPEC already specifies the shape)
  - **T-022** — integration test for runtime start/stop (runtime manager is
    the single highest-risk module, XL effort, worth a smoke test before
    frontend builds on top of it)

  Also folding in **T-090** / **T-091** (config validation + ring buffer unit
  tests) since they test alpha modules directly.

---

## Task breakdown

### Phase 0 — scaffolding
*(blocks everything)*

- `git init`, `.gitignore`
- `backend/go.mod`, `backend/cmd/devctl/main.go` (T-010)
- copy `devctl.example.yaml` → `backend/devctl.yaml` (gitignored)
- `frontend/` via `npm create vite@latest` (TS template) + Tailwind + shadcn/ui init (T-033)

### Phase 1 — backend config + workspace
*(blocks Phase 3+)*

- `backend/internal/config/schema.go`, `loader.go`, `defaults.go` (T-011)
- `backend/internal/config/validate.go` + `validate_test.go` (T-012, T-090)
- `backend/internal/workspace/service.go` (T-013)

### Phase 2 — process + logs primitives
*(blocks Phase 3)*

- `backend/internal/runtime/process_runner.go` (T-014)
- `backend/internal/logs/ring_buffer.go` + `ring_buffer_test.go` (T-015, T-091)
- `backend/internal/logs/manager.go` (T-016)

### Phase 3 — runtime manager
*(blocks Phase 4; depends on Phase 1 + 2)*

- `backend/internal/runtime/service_instance.go`, `manager.go`, `events.go` (T-017)
- wire stdout/stderr → logs manager (T-018)
- `backend/internal/runtime/manager_test.go` — dummy script that prints + exits on signal (T-022)

### Phase 4 — HTTP API
*(blocks Phase 5 partially; depends on Phase 3)*

- `backend/internal/api/dto.go`, `errors.go` (T-021)
- `backend/internal/api/handlers_workspace.go`, `handlers_services.go` (T-019)
- `backend/internal/api/handlers_logs.go` (T-020)
- `backend/internal/api/router.go`, `middleware.go`
- `backend/internal/app/app.go` — wiring container + graceful shutdown

### Phase 5 — WebSocket
*(parallel-ish with Phase 4, needs runtime events from Phase 3)*

- `backend/internal/api/ws_hub.go` (T-030)
- `backend/internal/api/ws_events.go` — `service.updated` (T-031), `log.appended` (T-032)

### Phase 6 — frontend data layer
*(depends on Phase 4/5 API being up so types match)*

- `frontend/src/types/api.ts`
- `frontend/src/lib/api.ts` (T-034), `ws.ts`
- `frontend/src/app/providers.tsx` — TanStack Query setup (T-035)
- `frontend/src/hooks/useWorkspaceQuery.ts`, `useServicesQuery.ts`, `useServiceLogs.ts`, `useRealtimeEvents.ts`

### Phase 7 — frontend UI
*(depends on Phase 6)*

- `frontend/src/components/layout/AppShell.tsx`, `Sidebar.tsx`, `TopBar.tsx`,
  `frontend/src/pages/DashboardPage.tsx` (T-036)
- `frontend/src/components/workspace/ServiceCard.tsx` (T-037), `ServiceGrid.tsx` (T-038)
- wire start/stop/restart buttons → mutations (T-039)
- `frontend/src/components/workspace/ServiceDetailsPanel.tsx` — Logs + Info tabs only for alpha (T-040)
- `frontend/src/components/logs/LogViewer.tsx` — fetch + scroll, no toolbar yet (T-041)
- `frontend/src/components/common/StatusBadge.tsx`, `EmptyState.tsx`, `LoadingState.tsx`

### Phase 8 — realtime wiring + smoke test
*(depends on Phase 5 + Phase 7)*

- apply `service.updated` to query cache (T-043), `log.appended` to log viewer (T-044)
- manual alpha smoke test against real `devctl.yaml` (T-046): load → start →
  see state change → see logs stream → stop

---

## Blocking chain (critical path)

```
Phase 0
  → Phase 1 & 2 (parallel)
    → Phase 3
      → Phase 4 & 5 (parallel)
        → Phase 6
          → Phase 7
            → Phase 8
```

---

## Deferred / open questions

- **Path reconciliation**: `devctl.yaml` needs real ids/paths matching
  `appctl2.py`'s actual layout (`~/Desktop/Code/mealsuite-core`,
  `mealsuite-core-app`, `mealsuite-touch-*`, plus `pos`) before T-046 is a
  genuine smoke test against real repos — that's a config edit, not a code
  task, but flagging so it doesn't get lost.
- **Port-override-per-service** (a real feature in `appctl2.py`) isn't in
  alpha scope — config `port` is fixed via YAML for now; revisit in beta if
  wanted.
- Whether `pos` gets added to the workspace config alpha targets, or alpha
  stays scoped to core+touch only — decide when editing `devctl.yaml`.

---

## Legacy audit reference (`~/appctl2.py`)

1725-line Textual/tmux app, managing services `core-app`(8080), `core-be`(3000),
`touch-app`(4202), `touch-be`(3001), `pos`(4205) at `~/Desktop/Code/mealsuite-*`.
Feature reference only per ARCHITECTURE.md/Plan.md ("migrate by capability,
not port line-for-line") — no tmux/Textual code carries over.

Real features found there, beyond current alpha scope (belongs in beta/polish
backlog, not blocking alpha):

- per-service port override (persisted, rewrites start cmd)
- unhealthy-vs-starting grace period on health probe
- rich log toolbar (search/filter/wrap/line-numbers/select-mode/copy variants)
- git dirty-gate before checkout + auto stash-switch + post-switch
  setup-command detection (bundle/npm install, db:migrate)
- separate "Activity" log per service (distinct from raw stdout)
- command palette (ctrl+k)
