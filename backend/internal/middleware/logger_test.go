package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogger_InjectsRequestID(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	loggedHandler := Logger(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()

	loggedHandler.ServeHTTP(rr, req)

	reqID := rr.Header().Get("X-Request-ID")
	if reqID == "" {
		t.Errorf("expected X-Request-ID header to be populated")
	}
}

func TestRecoverer_RecoversPanic(t *testing.T) {
	// A buggy handler that deliberately panics
	panickyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("simulated fatal nil-pointer dereference")
	})

	safeHandler := Recoverer(panickyHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/panic", nil)
	rr := httptest.NewRecorder()

	// Should NOT crash or propagate panic
	safeHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

