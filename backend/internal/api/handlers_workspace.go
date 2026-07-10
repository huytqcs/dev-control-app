package api

import "net/http"

func (h *Handlers) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	cfg := h.Workspace.GetWorkspace()

	presets := make([]PresetDTO, 0, len(cfg.Presets))
	for _, p := range cfg.Presets {
		presets = append(presets, PresetDTO{ID: p.ID, Name: p.Name, Services: p.Services})
	}

	writeJSON(w, http.StatusOK, WorkspaceDTO{Name: cfg.Name, Presets: presets})
}
