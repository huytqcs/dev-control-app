# Smoke test checklist

Every phase (alpha, beta, gamma, delta) has been smoke-tested live in a real
browser against a real backend before being called done — and every phase
caught real bugs doing it that unit tests missed. This is that checklist,
written down so each round doesn't re-derive it from scratch (T-095).

Run this against `devctl.example.yaml` (copy to `backend/devctl.yaml` and
point the paths at real repos) or your own workspace config. Check items off
as you go; if something doesn't match the expected result, that's a bug —
file it before shipping.

## Setup

- [ ] `make build` succeeds, `./bin/devctl --config backend/devctl.yaml`
      starts, and the dashboard loads at `http://127.0.0.1:4312` (single
      binary, no separate frontend dev server)
- [ ] `go run ./cmd/devctl` (from `backend/`) + `npm run dev` (from
      `frontend/`) still works as the devctl-development path — dashboard
      loads at `http://localhost:5173`

## Service lifecycle

- [ ] Start a stopped service — status goes `stopped` → `starting` →
      `running`, live log output appears immediately
- [ ] Stop a running service — status goes `running` → `stopping` →
      `stopped`
- [ ] Restart a running service — process actually cycles (new PID in the
      Info tab), not just a status flicker
- [ ] Stop a service whose start command is broken/exits immediately — status
      lands on `failed`, "Last error" surfaces the real failure reason (not
      a generic message), on both the service card and the Info tab
- [ ] Force-kill (Info tab) actually kills whatever holds the configured
      port, even if devctl didn't start it

## Presets

- [ ] Starting a preset with a dependency chain (e.g. `core` — `core-be` →
      `core-app`) starts services in dependency order, not all at once
- [ ] Stopping a preset stops in reverse dependency order
- [ ] Stop All (top bar) tears down every running service regardless of
      preset membership

## Health checks

- [ ] A service with a configured TCP health check goes from `unknown` to
      `healthy` shortly after it actually starts listening on its port
- [ ] Killing a healthy service's process out from under devctl (e.g. `kill
      -9` the PID from Info tab, outside the UI) flips it to failed/unhealthy
      rather than staying stuck on stale "running/healthy" state
- [ ] Stopping a healthy service normally (Stop button, not a crash) hides
      the health badge right away — it shouldn't stay stuck showing
      "Healthy"/"Unhealthy" after the service is already stopped

## Git tab

- [ ] Selecting a service with a real git repo shows its actual current
      branch and dirty/clean state, not placeholders
- [ ] A repo with a real ahead/behind vs. upstream shows the right ↑/↓ counts
- [ ] Branch search: typing a partial name filters the dropdown; selecting a
      match checks it out and the branch/dirty state refreshes immediately
- [ ] Branch search on a name with no match shows the inline "Create `<name>`
      from `<current branch>`" option; using it creates and checks out a new
      branch off the current one, and the branch list picks up the new
      branch without a page reload
- [ ] Fetch and Pull both refresh git state *and* the branch dropdown
      (a branch that only existed on the remote before Fetch should be
      selectable after)

## Workers

- [ ] A worker with `autoStart: true` starts automatically when its parent
      service starts, and stops when the parent stops or crashes
- [ ] A worker without `autoStart` can be started/stopped independently from
      the Workers tab without affecting the parent service

## Actions

- [ ] Running a custom action (e.g. `DB Migrate`) streams real stdout/stderr
      output live, not just a spinner, and reaches a real completion state
      (success or failure) — not stuck on "running" after the underlying
      command has actually exited
- [ ] A failing action surfaces its actual error, not a silent failure

## Open helpers

- [ ] "Open" (browser) opens the service's configured URL
- [ ] "Finder" opens the repo's path in Finder
- [ ] "Terminal" opens a terminal at the repo's path

## Log toolbar

- [ ] Search filters visible log lines live as you type
- [ ] Filter (by level/source, whatever's wired) actually narrows the log
      view
- [ ] Copy puts the visible (filtered) log content on the clipboard, not the
      entire unfiltered buffer
- [ ] Clear empties the visible log view
- [ ] Follow-tail: auto-scrolls to new lines when enabled; scrolling up
      manually disables it so you're not fighting the log viewer to read
      history

## Restart-resilience (has caught real bugs twice)

- [ ] With one or more services running, restart the devctl backend process
      itself (kill and re-`go run`/re-launch the binary) — on reconnect, the
      dashboard should show those services as still `running` with their
      real PIDs (orphan reconciliation adopts them), not reset to `stopped`
      or duplicate the process
