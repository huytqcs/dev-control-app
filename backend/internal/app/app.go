// Package app is the composition root: it wires config, workspace, runtime,
// logs, health, and git together and exposes the top-level router and
// shutdown hook to cmd/devctl.
package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"devctl/internal/api"
	"devctl/internal/config"
	"devctl/internal/git"
	"devctl/internal/health"
	"devctl/internal/logs"
	"devctl/internal/runtime"
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

	// Best-effort startup work: adopt any service still holding its port from
	// a previous backend instance (BETA_PLAN orphan-reconciliation decision,
	// option 1) and take an initial git-status snapshot for every service
	// (T-058 "on service load", not polled thereafter).
	runtimeMgr.ReconcileOrphans(context.Background())
	go runtimeMgr.RefreshAllGitStates(context.Background())

	router := api.NewRouter(&api.Handlers{
		Workspace: ws,
		Runtime:   runtimeMgr,
		Logs:      logMgr,
		Git:       gitSvc,
		Hub:       hub,
	})

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
