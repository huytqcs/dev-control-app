package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"devctl/internal/logs"
)

const defaultLogLimit = 500

func (h *Handlers) GetServiceLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "serviceID")
	if _, ok := h.Workspace.GetService(id); !ok {
		writeError(w, http.StatusNotFound, "service_not_found", fmt.Sprintf("service %q not found", id))
		return
	}

	limit := defaultLogLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	entries := h.Logs.Recent(logs.ServiceStreamKey(id), limit)
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}
