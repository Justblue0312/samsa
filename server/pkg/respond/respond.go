package respond

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/justblue/samsa/pkg/apierror"
)

// JSON writes any value as a JSON response with the given status code.
// This is the foundation — all other helpers call this.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Encoding failure after WriteHeader — we can't change the status anymore.
		// Log it and move on.
		slog.Error("respond: failed to encode response", "error", err)
	}
}

// OK writes a 200 JSON response.
func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, data)
}

// Created writes a 201 JSON response.
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, data)
}

// NoContent writes a 204 with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Message writes a JSON response with a single "message" field.
func Message(w http.ResponseWriter, message string) {
	OK(w, map[string]string{"message": message})
}

// Error writes an apierror.APIError as a JSON response.
// The HTTP status code is derived from the error's own HTTPStatus() method —
// no status code duplication in callers.
func Error(w http.ResponseWriter, err *apierror.APIError) {
	JSON(w, err.HTTPStatus(), err)
}
