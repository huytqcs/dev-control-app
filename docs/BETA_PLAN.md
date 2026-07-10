# Dev Control App V2 — Beta Implementation Plan

Scope: TASKS.md Sprint 3 / Milestone C ("daily-driver beta core"), T-050 → T-064,
plus cross-cutting T-092 (dependency-ordering tests) and T-093 (git service
tests). Builds on the alpha runtime (config, process runner, log buffer,
runtime manager, WS hub, REST API, dashboard) — nothing in this phase
replaces alpha, it adds presets, health checks, git actions, and workers on
top of it.

---

## Carried forward from alpha (don't regress these)

Real bugs surfaced by actually running the alpha against real repos, fixed
in `internal/runtime/process_runner.go` and `internal/runtime/manager.go`:

- **Login shell wrapping**: services are started via `$SHELL -lic "<command>"`,
  not exec'd directly — otherwise nvm/rbenv/Homebrew-shellenv PATH setup
  from `.zprofile`/`.zshrc` never gets sourced, and things like `npm`/`bundle`
  fail to resolve or resolve to the wrong binary.
- **`Setsid`, not `Setpgid`**: the interactive login shell (`-i`) tries to
  grab controlling-terminal job control on startup. `Setpgid` alone leaves
  the child in the parent's session/tty as a background process group, and
  the kernel `SIGTTOU`s it into a frozen `T` state before the real command
  ever runs. `Setsid` gives it a fresh session with no controlling terminal
  to fight over.
- **Path existence checked before start**: a missing `cwd` used to surface
  as a nonsensical `fork/exec /bin/zsh: no such file or directory` (a Go
  stdlib quirk — a failed `chdir` combined with a process-group `SysProcAttr`
  gets mislabeled as an exec failure on the shell binary). Now checked
  explicitly with a clear error before ever touching `exec`.
- **Config defaulting**: `dependsOn`/`openUrls`/preset `services` left unset
  in YAML must default to `[]`, not Go's nil-slice-marshals-to-`null` — the
  frontend calls `.length` on them unconditionally.
- **WS client is a true idempotent singleton**: no refcounted connect/
  disconnect — that broke under React StrictMode's double-invoke (two
  independent hooks decrementing a shared refcount could each think they
  were "the last consumer" and close a socket the other still needed).

## New problem surfaced by alpha usage (needs a decision before beta lands)

Services now run in their own detached session (`Setsid`), which means they
**survive a devctl backend restart** as orphans — the backend's in-memory
runtime state resets on restart, so it no longer knows a process it
previously started is still alive, holding a port/pidfile. Hit in practice:
restarting devctl during dev iteration left a live Rails server behind;
starting it again through the UI failed with Rails' own
"A server is already running" error, with no way to resolve it from the UI.

Three ways to handle this — **not yet decided, pick one before/while building
health checks (T-053), since reconciliation and health probing share the same
"is something already listening here" logic**:

1. **Reconcile on startup (recommended)** — when the backend starts, probe
   each service's configured port (or a `pidfile:` if we add that to the
   schema); if something's already listening, adopt it as `running`
   best-effort instead of showing a stale `stopped`/`failed`. Doesn't kill
   anything, just makes displayed state match reality. Natural extension of
   the health checker being built in this phase anyway.
2. **Kill-on-exit** — backend shutdown sends `SIGTERM` to everything it
   started. Simple, but means restarting devctl itself stops your dev
   servers, which fights the "daily driver, iterate often" use case.
3. **Manual force-kill action** — a per-service UI button that kills
   whatever's listening on its configured port regardless of whether devctl
   thinks it owns that process. Doesn't solve reconciliation, just gives an
   escape hatch.

(1) and (3) aren't mutually exclusive — (1) fixes the common case
automatically, (3) is a reasonable safety valve for when reconciliation
guesses wrong. (2) is probably not wanted. Confirm before starting T-053.

---

## Task breakdown

### Presets (T-050, T-051, T-052)
*(independent of health/git/workers below — can go first or in parallel)*

- `backend/internal/api/handlers_presets.go` — `POST /api/presets/:id/start`,
  `POST /api/presets/:id/stop` (T-050)
- Dependency-ordered startup in `internal/runtime/` — topological sort over
  `dependsOn`; cycle → fall back to config order + log warning (T-051, T-092
  tests)
- `frontend/src/components/workspace/PresetBar.tsx`, wired into
  `DashboardPage` top bar (T-052)

### Health checks (T-053, T-054, T-055)
*(resolve the orphan-reconciliation decision above first — same "probe a
port" primitive)*

- `backend/internal/health/service.go`, `checker.go` — TCP + HTTP checks,
  start/stop monitoring per running service, in-memory `HealthState` (T-053)
- Wire into `runtime.Manager`: replace the current hardcoded
  `HealthState{Status: "unknown"}` stub with real checker results
- `health.updated` WS event (T-054)
- `frontend/src/components/health/HealthBadge.tsx`, shown on
  `ServiceCard` and `ServiceDetailsPanel` (T-055)

### Git actions (T-056, T-057, T-058, T-059)

- `backend/internal/git/service.go` — branch, dirty status, fetch, pull,
  push, checkout, all via the `git` CLI (not a Go git library) (T-056, T-093
  tests)
- Replace the current `GitState{}` stub in `runtime.ServiceState` with real
  data
- `POST /api/services/:id/git/{fetch,pull,push,checkout}` (T-057)
- Refresh git state on service load and after any git action completes —
  explicitly, not polled (T-058)
- `frontend/src/components/git/GitPanel.tsx`, `BranchCheckoutForm.tsx`,
  new "Git" tab in `ServiceDetailsPanel` (T-059)

### Workers (T-060, T-061, T-062, T-063)

- `internal/runtime/worker_instance.go` — worker runtime struct, start/stop,
  state tracking, log capture via `logs.WorkerStreamKey(serviceID, workerID)`
  (already defined, unused until now) (T-060)
- `POST /api/services/:id/workers/:workerId/{start,stop}` (T-061)
- `worker.updated` WS event (T-062)
- Worker controls in `ServiceDetailsPanel` (e.g. "Start Sidekiq" / "Stop
  Sidekiq" + running badge) — replaces the current always-`"stopped"`
  `WorkerSummaryDTO` stub (T-063)

### Beta smoke test (T-064)

Manual: start a preset → dependent services come up in order → health
badges go healthy → git tab shows branch/dirty state → start a worker →
logs still behave correctly throughout. Same "drive it in a real browser
against a real backend" approach used for the alpha smoke test — it's what
caught all the real bugs listed above.

---

## Blocking chain

```
Presets (T-050/051/052) ─┐
                          ├─→ Beta smoke test (T-064)
Health  (T-053/054/055) ─┤      (needs all four areas done)
                          │
Git     (T-056/057/058/059) ┤
                          │
Workers (T-060/061/062/063) ┘
```

Presets, Health, Git, and Workers don't block each other — they touch
different files and can be built in any order or in parallel. Health should
land first if the orphan-reconciliation decision affects its design (see
above).

---

## Open questions deferred

- Orphan-reconciliation strategy (see above) — needs a decision, not just a
  recommendation, before T-053.
- Should `pidfile:` become a first-class config field (some services, like
  Rails, already write one) to make reconciliation more precise than a bare
  port probe? Port probing is simpler and works for anything with a
  configured port; pidfile-based is more precise but Rails-specific.
- Dependency cycle handling (T-051 MVP fallback: log + config order) — fine
  for beta, revisit if it causes confusing preset-start behavior in practice.
