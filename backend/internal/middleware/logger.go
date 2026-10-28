package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"
)

// responseWriterInterceptor wraps http.ResponseWriter to capture the HTTP status code.
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriterInterceptor(w http.ResponseWriter) *responseWriterInterceptor {
	return &responseWriterInterceptor{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default to 200 if WriteHeader is not explicitly called
	}
}

func (rw *responseWriterInterceptor) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// generateRequestID generates a random 8-byte hex string for request tracing.
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "req-unknown"
	}
	return hex.EncodeToString(bytes)
}

// Logger logs each incoming request with method, path, client IP, status code,
// latency, and an assigned X-Request-ID for distributed tracing.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Read or generate X-Request-ID
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
		}

		// Inject X-Request-ID into response headers so client can correlate errors
		w.Header().Set("X-Request-ID", reqID)

		interceptor := newResponseWriterInterceptor(w)

		// Execute next handler
		next.ServeHTTP(interceptor, r)

		duration := time.Since(start)
		clientIP := extractIP(r)

		log.Printf(
			"[%s] %s %s %s -> %d (%v)",
			reqID,
			clientIP,
			r.Method,
			r.URL.Path,
			interceptor.statusCode,
			duration,
		)
	})
}

