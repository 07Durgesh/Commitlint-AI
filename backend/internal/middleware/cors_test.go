package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_PreflightOptions(t *testing.T) {
	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	corsHandler := CORS(nextHandler)

	req := httptest.NewRequest(http.MethodOptions, "/api/analyze", nil)
	rr := httptest.NewRecorder()

	corsHandler.ServeHTTP(rr, req)

	// Preflight should NOT invoke the next handler
	if called {
		t.Errorf("preflight OPTIONS request should not call next handler")
	}

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rr.Code)
	}

	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", origin)
	}
}

func TestCORS_PassThroughHeaders(t *testing.T) {
	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := CORS(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/analyze", nil)
	rr := httptest.NewRecorder()

	corsHandler.ServeHTTP(rr, req)

	if !called {
		t.Errorf("POST request should call next handler")
	}

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", origin)
	}
}

