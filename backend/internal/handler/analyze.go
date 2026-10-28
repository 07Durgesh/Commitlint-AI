package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/07Durgesh/commitlint-AI/internal/model"
	"github.com/07Durgesh/commitlint-AI/internal/service"
)

const (
	// MaxDiffLength prevents malicious or accidental token explosions
	// by capping diff input to 50,000 characters.
	MaxDiffLength = 50000

	// MaxRequestBodySize limits the total HTTP request body to 1MB.
	MaxRequestBodySize = 1024 * 1024
)

// AnalyzeHandler handles requests to analyze git diffs and generate commit/PR metadata.
// It uses dependency injection (via closure) to receive the GeminiService instance.
func AnalyzeHandler(svc *service.GeminiService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Enforce POST HTTP method
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed. Use POST.")
			return
		}

		// 2. Protect server from huge payloads (DoS mitigation)
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

		// 3. Decode incoming JSON payload
		var req model.AnalyzeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
			return
		}

		// 4. Validate input
		trimmedDiff := strings.TrimSpace(req.Diff)
		if trimmedDiff == "" {
			writeError(w, http.StatusBadRequest, "Diff cannot be empty")
			return
		}

		if len(trimmedDiff) > MaxDiffLength {
			writeError(w, http.StatusBadRequest, "Diff exceeds maximum allowed length of 50,000 characters")
			return
		}

		// Normalize commit style (default to 'conventional')
		style := strings.ToLower(strings.TrimSpace(req.Style))
		if style == "" {
			style = "conventional"
		}

		switch style {
		case "conventional", "angular", "simple":
			// valid style
		default:
			writeError(w, http.StatusBadRequest, "Invalid style. Supported styles: 'conventional', 'angular', 'simple'")
			return
		}

		// 5. Call the Gemini service with request context to allow early cancellation
		result, err := svc.AnalyseWithContext(r.Context(), trimmedDiff, req.Context, style)
		if err != nil {
			log.Printf("[ERROR] Gemini analysis failed: %v", err)
			writeError(w, http.StatusInternalServerError, "Failed to analyze diff: "+err.Error())
			return
		}

		// 6. Return successful structured response
		writeJSON(w, http.StatusOK, result)
	}
}
