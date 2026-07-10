package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type checkoutRequest struct {
	Branch string `json:"branch"`
}

// runGitAction runs a git action against the service's repo path, then
// explicitly refreshes and returns its git state (T-058 — refreshed after
// every git action completes, never polled).
func (h *Handlers) runGitAction(w http.ResponseWriter, r *http.Request, action func(ctx context.Context, path string) error) {
	id := chi.URLParam(r, "serviceID")
	cfg, ok := h.Workspace.GetService(id)
	if !ok {
		writeError(w, http.StatusNotFound, "service_not_found", fmt.Sprintf("service %q not found", id))
		return
	}

	if err := action(r.Context(), cfg.Path); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "git_action_failed", err.Error())
		return
	}

	state, err := h.Runtime.RefreshGitState(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "git_refresh_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, GitStateDTO{
		Branch: state.Branch,
		Dirty:  state.Dirty,
		Ahead:  state.Ahead,
		Behind: state.Behind,
	})
}

func (h *Handlers) GitBranches(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "serviceID")
	cfg, ok := h.Workspace.GetService(id)
	if !ok {
		writeError(w, http.StatusNotFound, "service_not_found", fmt.Sprintf("service %q not found", id))
		return
	}

	branches, err := h.Git.ListBranches(r.Context(), cfg.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "git_branches_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"branches": branches})
}

func (h *Handlers) GitFetch(w http.ResponseWriter, r *http.Request) {
	h.runGitAction(w, r, h.Git.Fetch)
}

func (h *Handlers) GitPull(w http.ResponseWriter, r *http.Request) {
	h.runGitAction(w, r, h.Git.Pull)
}

func (h *Handlers) GitPush(w http.ResponseWriter, r *http.Request) {
	h.runGitAction(w, r, h.Git.Push)
}

func (h *Handlers) GitCheckout(w http.ResponseWriter, r *http.Request) {
	var body checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	h.runGitAction(w, r, func(ctx context.Context, path string) error {
		return h.Git.Checkout(ctx, path, body.Branch)
	})
}
