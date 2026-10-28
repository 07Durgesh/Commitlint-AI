package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()

	handler := HealthHandler("v1.0.0", "abcdef1", "2026-10-28")
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp["status"])
	}

	if resp["version"] != "v1.0.0" {
		t.Errorf("expected version 'v1.0.0', got '%s'", resp["version"])
	}

	if resp["commit"] != "abcdef1" {
		t.Errorf("expected commit 'abcdef1', got '%s'", resp["commit"])
	}
}

func TestHealthHandlerMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/health", nil)
	rr := httptest.NewRecorder()

	handler := HealthHandler("dev", "none", "unknown")
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rr.Code)
	}
}
