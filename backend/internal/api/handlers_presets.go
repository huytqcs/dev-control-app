package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) StartPreset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "presetID")
	preset, ok := h.Workspace.GetPreset(id)
	if !ok {
		writeError(w, http.StatusNotFound, "preset_not_found", fmt.Sprintf("preset %q not found", id))
		return
	}
	errs := h.Runtime.StartPreset(r.Context(), preset.Services)
	writeJSON(w, http.StatusOK, presetResultDTO(errs))
}

func (h *Handlers) StopPreset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "presetID")
	preset, ok := h.Workspace.GetPreset(id)
	if !ok {
		writeError(w, http.StatusNotFound, "preset_not_found", fmt.Sprintf("preset %q not found", id))
		return
	}
	errs := h.Runtime.StopPreset(r.Context(), preset.Services)
	writeJSON(w, http.StatusOK, presetResultDTO(errs))
}

// presetResultDTO reports partial failures instead of a single opaque
// success/fail — one bad service in a preset shouldn't hide errors for the
// rest.
func presetResultDTO(errs []error) map[string]any {
	messages := make([]string, 0, len(errs))
	for _, e := range errs {
		messages = append(messages, e.Error())
	}
	return map[string]any{"errors": messages}
}
