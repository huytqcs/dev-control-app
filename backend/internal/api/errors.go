package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"devctl/internal/runtime"
)

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
}

// mapRuntimeError translates a runtime package error into an HTTP status +
// error code (SPEC.md §20).
func mapRuntimeError(err error) (int, string) {
	switch {
	case errors.Is(err, runtime.ErrServiceNotFound):
		return http.StatusNotFound, "service_not_found"
	case errors.Is(err, runtime.ErrServiceAlreadyRunning):
		return http.StatusConflict, "service_already_running"
	case errors.Is(err, runtime.ErrServiceNotRunning):
		return http.StatusConflict, "service_not_running"
	default:
		return http.StatusInternalServerError, "process_start_failed"
	}
}
