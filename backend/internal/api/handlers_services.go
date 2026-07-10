package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) ListServices(w http.ResponseWriter, r *http.Request) {
	cfgs := h.Workspace.ListServices()
	dtos := make([]ServiceDTO, 0, len(cfgs))
	for _, cfg := range cfgs {
		state, _ := h.Runtime.GetState(cfg.ID)
		dtos = append(dtos, toServiceDTO(cfg, state))
	}
	writeJSON(w, http.StatusOK, map[string]any{"services": dtos})
}

func (h *Handlers) GetService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "serviceID")
	cfg, ok := h.Workspace.GetService(id)
	if !ok {
		writeError(w, http.StatusNotFound, "service_not_found", fmt.Sprintf("service %q not found", id))
		return
	}
	state, _ := h.Runtime.GetState(id)
	writeJSON(w, http.StatusOK, toServiceDTO(cfg, state))
}

func (h *Handlers) StartService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "serviceID")
	if err := h.Runtime.StartService(r.Context(), id); err != nil {
		h.writeRuntimeError(w, err)
		return
	}
	h.writeServiceState(w, id)
}

func (h *Handlers) StopService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "serviceID")
	if err := h.Runtime.StopService(r.Context(), id); err != nil {
		h.writeRuntimeError(w, err)
		return
	}
	h.writeServiceState(w, id)
}

func (h *Handlers) RestartService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "serviceID")
	if err := h.Runtime.RestartService(r.Context(), id); err != nil {
		h.writeRuntimeError(w, err)
		return
	}
	h.writeServiceState(w, id)
}

func (h *Handlers) writeServiceState(w http.ResponseWriter, id string) {
	cfg, ok := h.Workspace.GetService(id)
	if !ok {
		writeError(w, http.StatusNotFound, "service_not_found", fmt.Sprintf("service %q not found", id))
		return
	}
	state, _ := h.Runtime.GetState(id)
	writeJSON(w, http.StatusOK, toServiceDTO(cfg, state))
}

func (h *Handlers) writeRuntimeError(w http.ResponseWriter, err error) {
	status, code := mapRuntimeError(err)
	writeError(w, status, code, err.Error())
}
