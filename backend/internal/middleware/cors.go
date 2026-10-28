package middleware

import "net/http"

// CORS wraps an http.Handler and sets Cross-Origin Resource Sharing headers.
// It also intercepts and handles HTTP OPTIONS preflight requests.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers allowing frontend clients to interact with this API.
		// In development or public APIs, "*" allows any origin.
		// For restricted production APIs, this can be matched against an allowlist.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400") // Cache preflight response for 24 hours

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent) // 204 No Content
			return
		}

		// Proceed to the next handler in the pipeline
		next.ServeHTTP(w, r)
	})
}

