package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"devctl/internal/openhelpers"
)

func (h *Handlers) OpenBrowser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "serviceID")
	cfg, ok := h.Workspace.GetService(id)
	if !ok {
		writeError(w, http.StatusNotFound, "service_not_found", fmt.Sprintf("service %q not found", id))
		return
	}
	if len(cfg.OpenURLs) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "no_url_configured", fmt.Sprintf("service %q has no openUrls configured", id))
		return
	}
	if err := openhelpers.OpenBrowser(cfg.OpenURLs[0]); err != nil {
		writeError(w, http.StatusInternalServerError, "open_browser_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handlers) OpenRepo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "serviceID")
	cfg, ok := h.Workspace.GetService(id)
	if !ok {
		writeError(w, http.StatusNotFound, "service_not_found", fmt.Sprintf("service %q not found", id))
		return
	}
	if err := openhelpers.OpenRepo(cfg.Path); err != nil {
		writeError(w, http.StatusInternalServerError, "open_repo_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handlers) OpenTerminal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "serviceID")
	cfg, ok := h.Workspace.GetService(id)
	if !ok {
		writeError(w, http.StatusNotFound, "service_not_found", fmt.Sprintf("service %q not found", id))
		return
	}
	if err := openhelpers.OpenTerminal(cfg.Path); err != nil {
		writeError(w, http.StatusInternalServerError, "open_terminal_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
