package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"devctl/internal/actions"
	"devctl/internal/config"
)

func (h *Handlers) RunAction(w http.ResponseWriter, r *http.Request) {
	serviceID := chi.URLParam(r, "serviceID")
	actionID := chi.URLParam(r, "actionID")

	cfg, ok := h.Workspace.GetService(serviceID)
	if !ok {
		writeError(w, http.StatusNotFound, "service_not_found", fmt.Sprintf("service %q not found", serviceID))
		return
	}

	var actionCfg *config.ActionConfig
	for i := range cfg.Actions {
		if cfg.Actions[i].ID == actionID {
			actionCfg = &cfg.Actions[i]
			break
		}
	}
	if actionCfg == nil {
		writeError(w, http.StatusNotFound, "action_not_found", fmt.Sprintf("action %q not found on service %q", actionID, serviceID))
		return
	}

	runID, err := h.Actions.Run(r.Context(), serviceID, *actionCfg, cfg.Path)
	if err != nil {
		if errors.Is(err, actions.ErrActionAlreadyRunning) {
			writeError(w, http.StatusConflict, "action_already_running", err.Error())
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "action_start_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"runId": runID, "status": "started"})
}
