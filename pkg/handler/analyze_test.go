package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/07Durgesh/commitlint-AI/pkg/model"
	"github.com/07Durgesh/commitlint-AI/pkg/service"
)

func TestAnalyzeHandler_MethodNotAllowed(t *testing.T) {
	svc := service.NewGeminiService("test-key")
	handler := AnalyzeHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/analyze", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed, got %d", rr.Code)
	}
}

func TestAnalyzeHandler_EmptyDiff(t *testing.T) {
	svc := service.NewGeminiService("test-key")
	handler := AnalyzeHandler(svc)

	body, _ := json.Marshal(model.AnalyzeRequest{
		Diff:  "   ",
		Style: "conventional",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rr.Code)
	}

	var errResp model.ErrorResponse
	json.Unmarshal(rr.Body.Bytes(), &errResp)
	if !strings.Contains(errResp.Error, "Diff cannot be empty") {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestAnalyzeHandler_InvalidStyle(t *testing.T) {
	svc := service.NewGeminiService("test-key")
	handler := AnalyzeHandler(svc)

	body, _ := json.Marshal(model.AnalyzeRequest{
		Diff:  "+ some change",
		Style: "unknown-style",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rr.Code)
	}
}

func TestAnalyzeHandler_DiffTooLarge(t *testing.T) {
	svc := service.NewGeminiService("test-key")
	handler := AnalyzeHandler(svc)

	hugeDiff := strings.Repeat("x", MaxDiffLength+10)
	body, _ := json.Marshal(model.AnalyzeRequest{
		Diff:  hugeDiff,
		Style: "conventional",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rr.Code)
	}
}

