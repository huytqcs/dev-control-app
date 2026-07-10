# Dev Control App V2 — Beta Implementation Plan

**Status: shipped.** All of T-050 → T-064, T-092, T-093 are implemented and
passed the beta smoke test against real repos. See "Decisions made" below for
how the one open design question got resolved, and "Beyond original scope"
for one addition made after this plan was written.

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

## New problem surfaced by alpha usage — decision made, implemented

Services now run in their own detached session (`Setsid`), which means they
**survive a devctl backend restart** as orphans — the backend's in-memory
runtime state resets on restart, so it no longer knows a process it
previously started is still alive, holding a port/pidfile. Hit in practice:
restarting devctl during dev iteration left a live Rails server behind;
starting it again through the UI failed with Rails' own
"A server is already running" error, with no way to resolve it from the UI.

Three ways were on the table; **went with (1) + (3), the recommended combo**:

1. **Reconcile on startup (recommended, shipped)** — `Manager.ReconcileOrphans`
   probes each stopped service's configured port at backend startup; if
   something's already listening, adopts it as `running` best-effort instead
   of showing a stale `stopped`/`failed`. Doesn't kill anything, just makes
   displayed state match reality. `internal/runtime/reconcile.go`.
2. **Kill-on-exit** — not implemented, as expected: would stop dev servers on
   every devctl restart, fighting the "daily driver, iterate often" use case.
3. **Manual force-kill action (shipped)** — `POST /api/services/:id/force-kill`
   + a button in the service detail panel's Info tab, kills whatever's
   listening on the configured port via `lsof -ti :<port>` + `SIGKILL`,
   regardless of whether devctl thinks it owns that process. Also reused as
   `StopService`'s fallback for any service with no in-memory process handle
   (an adopted orphan, or one never started by this backend instance) — a
   deliberate Stop needs *some* way to actually kill it. `ForceKillPort` in
   the same file backs both paths.

`pidfile:` as a more precise alternative to bare port-probing was considered
and deferred — see SPEC.md §26.1 "Not implemented".

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

- ~~Orphan-reconciliation strategy~~ — resolved, see above.
- Should `pidfile:` become a first-class config field (some services, like
  Rails, already write one) to make reconciliation more precise than a bare
  port probe? Still deferred — port probing is simpler and works for
  anything with a configured port; pidfile-based is more precise but
  Rails-specific. Revisit if port reuse by an unrelated process causes a bad
  adoption in practice.
- Dependency cycle handling (T-051 MVP fallback: log + config order) —
  shipped as-is, no cycles hit in the beta smoke test's real preset. Revisit
  if it causes confusing preset-start behavior in practice.

---

## Beyond original scope: worker `autoStart`

Not in the original T-060 → T-063 breakdown, added after a direct ask
("start Sidekiq with its service, not as a separate manual step"):

- `WorkerConfig.autoStart` (YAML `autoStart: true`) ties a worker to its
  parent service's lifecycle — starts when the service starts, stops when
  the service stops or crashes. Default `false` keeps the original
  independently-controllable behavior (ARCHITECTURE.md §12.2).
- Set on both real Sidekiq workers in `backend/devctl.yaml`; documented in
  `devctl.example.yaml`.
- Building this exposed a real bug in the original worker stop path: unlike
  `StopService`, `StopWorker` had no `WorkerStopping` status, so a worker
  killed by `SIGTERM` (exit via signal, not a clean exit 0 — the common case
  for most real long-running commands) was misreported as `failed` instead
  of `stopped`, for *any* worker stop, not just autoStart ones. Fixed by
  giving workers the same stopping-state guard services already had.

---

## Beyond original scope: searchable branch checkout

T-059's `BranchCheckoutForm.tsx` originally took free-text branch input.
Following a direct ask ("show list branch then can search"), it's now a
live-filtered picker instead:

- `git.Service.ListBranches` (`internal/git/service.go`) — local branches
  plus remote-tracking branches' short names, deduplicated and sorted, via
  `git for-each-ref`. A remote-only name still works with `Checkout` since
  `git checkout <name>` auto-creates a local tracking branch for it (git's
  own DWIM behavior).
- `GET /api/services/:id/git/branches` (SPEC.md §19.6).
- `BranchCheckoutForm` filters client-side as you type, caps to 8 visible
  matches, marks the current branch disabled. Verified against a real repo
  with ~8,000 branches — filtering stayed instant.
- `GitPanel` invalidates the branch list after Fetch/Pull, since either can
  surface remote branches that didn't exist locally before.
