package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"devctl/internal/runtime"
)

func (h *Handlers) StartWorker(w http.ResponseWriter, r *http.Request) {
	serviceID := chi.URLParam(r, "serviceID")
	workerID := chi.URLParam(r, "workerID")
	if err := h.Runtime.StartWorker(r.Context(), serviceID, workerID); err != nil {
		h.writeWorkerError(w, err)
		return
	}
	h.writeServiceState(w, serviceID)
}

func (h *Handlers) StopWorker(w http.ResponseWriter, r *http.Request) {
	serviceID := chi.URLParam(r, "serviceID")
	workerID := chi.URLParam(r, "workerID")
	if err := h.Runtime.StopWorker(r.Context(), serviceID, workerID); err != nil {
		h.writeWorkerError(w, err)
		return
	}
	h.writeServiceState(w, serviceID)
}

func (h *Handlers) writeWorkerError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, runtime.ErrServiceNotFound), errors.Is(err, runtime.ErrWorkerNotFound):
		writeError(w, http.StatusNotFound, "worker_not_found", err.Error())
	case errors.Is(err, runtime.ErrWorkerAlreadyRunning):
		writeError(w, http.StatusConflict, "worker_already_running", err.Error())
	case errors.Is(err, runtime.ErrWorkerNotRunning):
		writeError(w, http.StatusConflict, "worker_not_running", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "worker_action_failed", err.Error())
	}
}
