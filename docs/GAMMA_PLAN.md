# Dev Control App V2 — Gamma Implementation Plan

Scope: TASKS.md Sprint 4 ("productivity features + polish"), T-070 → T-082,
plus cross-cutting T-094 (frontend component tests) and T-095 (end-to-end
manual checklist). Builds on the alpha runtime and beta's presets/health/
git/workers — nothing in this phase replaces prior work, it adds the action
runner, open helpers, log toolbar UX, and shutdown/polish behavior on top.

---

## Carried forward from beta (don't regress)

- **Stopping-state guard pattern**: `StopService`/`StopWorker` both need an
  explicit `*Stopping` status before signaling, so a deliberate stop that
  exits via SIGTERM (not a clean exit 0) is reported as stopped, not failed
  (`internal/runtime/manager.go`, fixed for workers after beta shipped with
  the bug). If action runs end up as long-lived processes for any command
  (e.g. a `--watch` test runner), apply the same pattern rather than
  re-learning this the hard way a third time.
- **Orphan reconciliation is a deliberate feature, not a gap**: beta made
  services survive a devctl backend restart on purpose (`ReconcileOrphans`,
  `docs/SPEC.md` §26.1). This directly affects how T-080 should be read —
  see below.
- **Workers have two lifecycles now**: independently controlled (default) or
  `autoStart`-tied to their parent service. Actions are one-off runs, not
  long-running units (ARCHITECTURE.md §12.3) — no `autoStart` equivalent
  makes sense for them; don't add one.

---

## Decision needed: T-080 ("graceful stop on shutdown") vs. orphan reconciliation

TASKS.md's original T-080 says "when devctl exits, stop managed services
optionally or preserve based on future config." Beta's core design decision
was the opposite of "stop on exit": services are *meant* to survive a devctl
restart, so a restart can reconcile and re-adopt them. If T-080 shipped with
"stop everything on exit" as the default, it would silently undo the whole
point of `ReconcileOrphans`.

**Recommendation**: don't implement literal stop-on-exit. Instead ship a
**"Stop All" UI action** — a single button that calls `StopService`/
`StopWorker` for everything currently running, reusing existing lifecycle
code. This covers the real use case ("I'm done for the day, kill everything")
as a deliberate, visible action instead of an implicit side effect of
quitting the backend, and doesn't fight reconciliation. `Shutdown()` in
`internal/app/app.go` stays a no-op.

Confirm this reading before starting — it changes what T-080 actually builds.

---

## Task breakdown

### Action runner (T-070, T-071, T-072, T-073)

- `backend/internal/actions/service.go` — run config-defined one-off
  commands (`ServiceConfig.Actions`, already in schema and surfaced as
  `ActionSummaryDTO` on every service — just never executable). Each run
  gets its own run ID, output stream, and success/failure result, kept
  separate from service runtime state (ARCHITECTURE.md §12.3).
- One in-flight run per `(service, action)` pair — mirror the
  already-running/already-started guard `StartService`/`StartWorker` use,
  don't allow a second concurrent run of the same action.
- `POST /api/services/:id/actions/:actionId` — starts a run, returns a
  `runId` immediately (matches the `{"runId": "...", "status": "started"}`
  shape already sketched in SPEC.md §19.7).
- `action.output` WS event streaming stdout/stderr per line (stream key
  `action:<serviceId>:<actionId>:<runId>`, per SPEC.md §16.4), plus a
  terminal event (or a status field on the last output) carrying exit code.
- `frontend/src/components/actions/ActionsPanel.tsx` — new "Actions" tab in
  `ServiceDetailsPanel`, one row per configured action with a Run button and
  a small streaming output panel (can reuse `LogViewer`'s scroll/style
  patterns rather than building a second log renderer from scratch).

### Open helpers (T-074, T-075, T-076)

- `backend/internal/openhelpers/service.go` — thin OS integration, not a
  core domain (ARCHITECTURE.md §19): open browser (`open <url>` on macOS),
  open repo folder (`open <path>`, Finder by default), open terminal
  (`open -a Terminal <path>`).
- `POST /api/services/:id/open-browser` (uses the service's first configured
  `openUrls` entry, or a specific one if the request names it),
  `POST /api/services/:id/open-repo`, `POST /api/services/:id/open-terminal`.
  These endpoints are all-new — `openUrls` has been plumbed through
  config → DTO → frontend types since alpha but nothing has ever opened one.
- Wire small icon buttons into `ServiceCard`/`ServiceDetailsPanel` (next to
  Start/Stop, low visual weight — these are conveniences, not primary
  actions).

### Log toolbar (T-077, T-078)

`LogViewer.tsx` today is just an auto-scrolling list with stderr in red —
none of this exists yet:

- follow-tail toggle: pause auto-scroll when the user scrolls up to read,
  show a "N new lines — resume" affordance instead of yanking them back down
- clear visible logs (client-side only — doesn't touch the backend ring
  buffer, so it comes back on next fetch/reselect)
- copy visible logs to clipboard
- search box + severity filter, filtering the already-buffered client-side
  lines (no new backend query — logs are already fully in memory per
  `useServiceLogs`)
- error highlighting: extend beyond "stderr is red" to pattern-matching
  likely errors in stdout too (case-insensitive `error`/`exception`/`fatal`
  etc.), since plenty of real error output goes to stdout depending on the
  framework's logger config

### Preset startup UX (T-079)

`StartPreset`/`StopPreset` already return a partial-failure `errors` array
and per-service progress is already observable live via `service.updated`
events — this task is UI-only:

- show "starting N/M" progress in `PresetBar` while the mutation is pending
- surface which specific service failed inline instead of just the raw
  joined error string currently shown

### Shutdown / polish (T-080 revised, T-081, T-082)

- T-080: ship "Stop All" per the decision above, not stop-on-exit.
- T-081 structured backend logging — replace the scattered `log.Printf`
  calls (`reconcile.go`, `manager.go`, etc.) with a small structured logger
  (level + component tag). Mechanical, low risk, do it in one pass.
- T-082 service card / detail panel polish — now that git/health/workers
  are all real data (not alpha-era stubs), revisit spacing, dependency list
  display, and failure-state summaries with real content to design against
  instead of placeholders.

### Cross-cutting (T-094, T-095)

- T-094: there is currently **no frontend test framework at all** in this
  repo. Add Vitest + React Testing Library, start with component tests for
  `ServiceCard` and `LogViewer` (the two most-reused, most load-bearing
  components) rather than trying to cover everything at once.
- T-095: distill the alpha and beta smoke tests (both done manually, driven
  in a real browser via chrome-devtools MCP) into a repeatable written
  checklist, so gamma's own smoke test — and any future one — doesn't
  depend on remembering what alpha/beta happened to check.

---

## Blocking chain

```
Action runner (T-070/071/072/073) ─┐
Open helpers  (T-074/075/076)      ─┤
Log toolbar   (T-077/078)          ─┼─→ Gamma smoke test (T-095)
Preset UX     (T-079)              ─┤      (needs all five areas done)
Shutdown/polish (T-080/081/082)    ─┘
```

All five areas touch different files and can be built in any order or in
parallel — same shape as beta. T-094 (frontend test infra) has no hard
dependency on the others; it's reasonable to land it first so new
components from this phase can ship with tests instead of retrofitting them.

---

## Open questions to resolve before starting

- **T-080 reading** — confirmed above as "Stop All" action, not stop-on-exit
  default. Flag if that's not the intended interpretation.
- **Open-terminal target** — recommend `open -a Terminal <path>` (zero
  config, ships on every Mac) as the default, with an optional
  `terminalApp:` config field later if iTerm/Warp support is wanted. Don't
  build the config field speculatively before someone actually needs it.
- **Open-repo target** — recommend Finder (`open <path>`) as the default
  rather than guessing an editor; let a future `editorCommand:` field
  (`code`, `cursor`, etc.) opt a service into launching an editor instead,
  per the TASKS.md T-075 note. Same "don't build it until needed" stance.
- **Action run concurrency** — recommend one in-flight run per
  `(service, action)` pair, rejecting a second start the same way
  `StartService`/`StartWorker` already reject a double-start. Confirm this
  before T-070, since it affects the action runtime's state shape.
