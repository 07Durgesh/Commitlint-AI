package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/07Durgesh/commitlint-AI/internal/model"
)

// Recoverer catches any unexpected panics in downstream handlers,
// logs the stack trace for debugging, and returns a safe 500 Internal Server Error
// instead of letting the entire Go process crash.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				// Capture stack trace
				stack := string(debug.Stack())
				log.Printf("[PANIC RECOVERED] %v\nStack trace:\n%s", rvr, stack)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(model.ErrorResponse{
					Error: "An unexpected internal server error occurred.",
					Code:  http.StatusInternalServerError,
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

