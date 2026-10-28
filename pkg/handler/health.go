package handler

import (
	"encoding/json"
	"net/http"

	"github.com/07Durgesh/commitlint-AI/pkg/model"
)

// writeJSON is a reusable helper function that serializes data into JSON,
// sets the appropriate Content-Type header, and writes the HTTP status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
	}
}

// writeError standardizes API error responses across all handlers.
// Using model.ErrorResponse guarantees clients always receive a consistent schema.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, model.ErrorResponse{
		Error: message,
		Code:  status,
	})
}

// HealthHandler returns dynamic application health, version, commit, and build timestamp metadata.
// This allows monitoring tools (DataDog, Prometheus, Render health checks) and DevOps pipelines
// to inspect running production versions in real time.
func HealthHandler(version, commit, buildDate string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed. Use GET.")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status":     "ok",
			"version":    version,
			"commit":     commit,
			"build_date": buildDate,
		})
	}
}
