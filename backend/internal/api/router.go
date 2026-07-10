package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"devctl/internal/actions"
	"devctl/internal/git"
	"devctl/internal/logs"
	"devctl/internal/runtime"
	"devctl/internal/workspace"
)

// Handlers holds the backend services the HTTP/WS layer translates into API
// endpoints (SPEC.md §9.1 app container, scoped to what api needs directly).
type Handlers struct {
	Workspace *workspace.Service
	Runtime   *runtime.Manager
	Logs      *logs.Manager
	Git       *git.Service
	Actions   *actions.Service
	Hub       *Hub
}

func NewRouter(h *Handlers) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/ws", h.Hub.ServeWS)

	r.Route("/api", func(r chi.Router) {
		r.Get("/workspace", h.GetWorkspace)

		r.Route("/presets/{presetID}", func(r chi.Router) {
			r.Post("/start", h.StartPreset)
			r.Post("/stop", h.StopPreset)
		})

		r.Post("/stop-all", h.StopAll)

		r.Route("/services", func(r chi.Router) {
			r.Get("/", h.ListServices)

			r.Route("/{serviceID}", func(r chi.Router) {
				r.Get("/", h.GetService)
				r.Get("/logs", h.GetServiceLogs)
				r.Post("/start", h.StartService)
				r.Post("/stop", h.StopService)
				r.Post("/restart", h.RestartService)
				r.Post("/force-kill", h.ForceKillService)
				r.Post("/open-browser", h.OpenBrowser)
				r.Post("/open-repo", h.OpenRepo)
				r.Post("/open-terminal", h.OpenTerminal)

				r.Route("/git", func(r chi.Router) {
					r.Get("/branches", h.GitBranches)
					r.Post("/fetch", h.GitFetch)
					r.Post("/pull", h.GitPull)
					r.Post("/push", h.GitPush)
					r.Post("/checkout", h.GitCheckout)
				})

				r.Route("/workers/{workerID}", func(r chi.Router) {
					r.Post("/start", h.StartWorker)
					r.Post("/stop", h.StopWorker)
				})

				r.Post("/actions/{actionID}", h.RunAction)
			})
		})
	})

	return r
}
