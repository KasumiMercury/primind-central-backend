package health

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// LiveHandler returns a simple health check response for liveness probes.
// This endpoint only checks if the process is running.
func (c *Checker) LiveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		slog.Warn("failed to write health response", slog.String("error", err.Error()))
	}
}

// ReadyHandler returns a detailed health check response for readiness probes.
// This endpoint checks all dependencies and returns their status.
func (c *Checker) ReadyHandler(w http.ResponseWriter, r *http.Request) {
	status := c.Check(r.Context())

	w.Header().Set("Content-Type", "application/json")

	if status.Status == StatusHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		slog.Warn("failed to write health response", slog.String("error", err.Error()))
	}
}
