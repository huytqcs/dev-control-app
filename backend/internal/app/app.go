// Package app is the composition root: it wires config, workspace, runtime,
// logs, health, and git together and exposes the top-level router and
// shutdown hook to cmd/devctl.
package app

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"devctl/internal/actions"
	"devctl/internal/api"
	"devctl/internal/applog"
	"devctl/internal/config"
	"devctl/internal/git"
	"devctl/internal/health"
	"devctl/internal/logs"
	"devctl/internal/runtime"
	"devctl/internal/webui"
	"devctl/internal/workspace"
)

type App struct {
	router  http.Handler
	Runtime *runtime.Manager
}

func New(configPath string) (*App, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	ws := workspace.New(cfg)
	logMgr := logs.NewManager()
	hub := api.NewHub()
	runtimeMgr := runtime.NewManager(ws, runtime.NewOSProcessRunner(), logMgr, hub)

	healthMonitor := health.NewMonitor(func(serviceID string, status health.Status) {
		runtimeMgr.SetServiceHealth(serviceID, string(status))
	})
	runtimeMgr.SetHealthMonitor(healthMonitor)

	gitSvc := git.NewService()
	runtimeMgr.SetGitProbe(runtime.NewGitAdapter(gitSvc))

	actionsSvc := actions.NewService(logMgr, hub)

	// Best-effort startup work: adopt any service still holding its port from
	// a previous backend instance (BETA_PLAN orphan-reconciliation decision,
	// option 1) and take an initial git-status snapshot for every service
	// (T-058 "on service load", not polled thereafter).
	runtimeMgr.ReconcileOrphans(context.Background())
	go runtimeMgr.RefreshAllGitStates(context.Background())

	// Watches each repo's .git/HEAD and refs/heads for branch switches made
	// outside the app (e.g. `git checkout` in an external terminal), which
	// RefreshAllGitStates alone would never see again after startup.
	if err := runtimeMgr.StartGitWatcher(context.Background()); err != nil {
		applog.Error("app", "git watcher: %v", err)
	}

	// Always embed: during `go run` (daily devctl development) dist/ only
	// has its .gitkeep placeholder, so this static fallback just 404s and
	// the Vite dev server on :5173 is what's actually browsed. A `make
	// build` release binary is the only case where dist/ has real content
	// (DELTA_PLAN.md "embed unconditionally, don't build-tag it").
	distFS, err := fs.Sub(webui.DistFS, "dist")
	if err != nil {
		return nil, fmt.Errorf("load embedded frontend: %w", err)
	}

	router := api.NewRouter(&api.Handlers{
		Workspace: ws,
		Runtime:   runtimeMgr,
		Logs:      logMgr,
		Git:       gitSvc,
		Actions:   actionsSvc,
		Hub:       hub,
	}, distFS)

	return &App{router: router, Runtime: runtimeMgr}, nil
}

func (a *App) Router() http.Handler {
	return a.router
}

func (a *App) ShutdownTimeout() time.Duration {
	return 5 * time.Second
}

// Shutdown is a no-op in alpha (SPEC.md defers "stop managed services on
// exit" — TASKS.md T-080 — to a later phase).
func (a *App) Shutdown(ctx context.Context) {}
