# Dev Control App V2 — Delta Implementation Plan

Not a TASKS.md-numbered sprint — Sprint 4 (gamma) was the last one defined,
and Plan.md's own MVP boundary list (§5) is now fully covered. This was the
wrap-up before calling v1 done: close the two things gamma left dangling
(T-082, T-095), ship the packaging step ARCHITECTURE.md §24.1 calls "a very
good second step" (a single binary, no more two-terminal dev workflow for
daily use), and pick up the one git stretch feature Plan.md §C flagged for
"maybe v1.1" (create branch from main). **All four are now shipped** — see
each section below for what landed.

---

## Carried forward / stale things noticed

- ~~**README.md is badly out of date.**~~ Done — rewritten: status/feature
  list synced to what's actually shipped, and the running instructions now
  lead with `make build` + the binary as the primary path, two-terminal
  `go run`/`npm run dev` labeled as the devctl-development path.
- ~~**No build tooling exists.**~~ Done — root `Makefile`'s `build` target
  does frontend build → embed copy → `go build`, per the packaging section
  below.

---

## Task breakdown

### Single-binary packaging (shipped — ARCHITECTURE.md §24.1)

Done. `backend/internal/webui` embeds `dist/` via `go:embed all:dist` (the
`all:` prefix is needed because the checked-in `dist/.gitkeep` placeholder
is a dotfile, which plain `embed dist` excludes and fails to compile on
until a real build populates the directory). `internal/api.NewRouter` takes
an `fs.FS` and, when non-nil, registers a `chi` `NotFound` handler that
serves embedded assets or falls back to `index.html` for any unmatched
path — except `/api/*` and `/ws`, which 404 normally instead of getting
swallowed by the SPA fallback. Root `Makefile`'s `build` target runs `npm
run build`, copies `frontend/dist` → `backend/internal/webui/dist`, then
`go build`s `backend/cmd/devctl` into `bin/devctl`. `go run ./cmd/devctl`
(daily devctl development) is unaffected — `dist/` still only has its
placeholder in source control, so the fallback handler just 404s and Vite
on :5173 is what's actually browsed, exactly as before.

### Close T-082 — service card / detail panel polish (shipped)

- Dependency list: Info tab's `dl` comma-string replaced with `Badge` pills
  (`ServiceDetailsPanel.tsx`), matching the Workers/Actions tabs' visual
  language instead of a plain string.
- Spacing/density: `ServiceCard.tsx` content gap tightened (`gap-3` →
  `gap-2.5`), branch rendered as a small mono chip instead of plain text
  alongside the port, dot-separated.
- Failure-state summaries: both the card and the Info tab's `lastError` are
  now click-to-expand ("Show more"/"Show less") instead of relying solely
  on a `title` tooltip for the full text — card still truncates by default,
  Info tab line-clamps to 3 lines. Expand state resets on service switch.
- Info tab: replaced the single flat `dl` grid with a bordered
  type/path/port/PID/exit-code section, a separate dependency-badges
  section, and (when present) a dedicated "Last error" section — closer to
  how Git/Workers/Actions already read.

### Close T-095 — written smoke-test checklist (shipped)

`docs/SMOKE_TEST.md` — the repeatable checklist: start/stop/restart a
service, preset start respects dependency order, health badges go healthy,
git tab shows real branch/dirty/ahead-behind, branch search + create-from
work, worker `autoStart` starts/stops with its service, an action runs and
streams output to a real completion, log toolbar controls, Stop All, and
the mid-session backend-restart orphan-reconciliation check.

### Git stretch: create branch from main (shipped — Plan.md §C "maybe v1.1")

- `internal/git/service.go`: `CreateBranch(ctx, repoPath, name, from
  string) error` — `git checkout -b <name> <from>`, reusing `Checkout`'s
  ref-name validation (factored out as `checkRefName`) for both `name` and
  `from`. Covered by `TestCreateBranch_CreatesAndChecksOut` and
  `TestCreateBranch_RejectsInvalidRef`.
- `POST /api/services/:id/git/checkout`'s request body takes optional
  `create: true` + `from` fields rather than a new endpoint — `GitCheckout`
  routes to `CreateBranch` when `create` is set, `Checkout` otherwise.
- Frontend: `BranchCheckoutForm` shows an inline "Create `<query>` from
  `<current branch>`" row in the dropdown when the typed name matches no
  existing branch; `GitPanel`/`api.ts` thread `createFrom` through and
  refresh the branch list on success so the new branch is immediately
  selectable.
- Follow-up UX fix: dropped the standalone "Checkout" button — clicking a
  dropdown row already checked out immediately, so the button was a second,
  confusing path to the same action. Enter now acts on the top dropdown row
  (a match, or the create row if there's no match) instead.

### Bug fix: health badge stuck after stop (found during delta smoke-testing)

`health.Monitor.Stop` cancelled the probe loop's context and immediately
reported `unknown` itself, without waiting for a probe already in flight to
actually finish — that straggler could still land its own (now-stale)
healthy/unhealthy result *after* the reset, leaving a stopped service's
health badge stuck instead of hiding. Fixed by giving the probe loop a
`done` channel it closes on exit; `Stop` now blocks on it before reporting
`unknown`, so no late write can land afterward. Reproduced against the old
code first (`monitor_test.go`'s `TestStop_WaitsForInFlightProbeBeforeReportingUnknown`
fails reliably pre-fix, passes post-fix under `-race`), then verified live
through the built binary with 10 start/stop cycles against a real health
check.

### README rewrite

- Status section: reflect what's actually shipped — presets, health checks,
  git actions + searchable branch checkout, workers + `autoStart`, custom
  actions, open helpers, Stop All
- Running section: once packaging lands, lead with "download/build the
  binary and run it" as the primary path; keep today's two-terminal
  instructions as the "if you're developing devctl itself" path, clearly
  labeled as such

---

## Open questions

- **Embed unconditionally, or gate it?** Resolved: always embed, no
  build-tag scheme — shipped as described above. Hasn't been slow enough
  in practice to revisit.
- **Create-branch endpoint shape** — confirmed above as extending checkout
  with `create`/`from` fields rather than a new endpoint. Flag if a
  separate `POST .../git/create-branch` is preferred instead.
- **Is v1 "done" after this?** Recommend yes. Plan.md's own MVP boundary
  list (§5) is fully covered once this lands; anything past this point
  (Docker orchestration UI, plugin system, multi-user/team sync, etc.) is
  explicitly out of scope per Plan.md §5 "MVP does not include," not a
  natural continuation of this backlog.
