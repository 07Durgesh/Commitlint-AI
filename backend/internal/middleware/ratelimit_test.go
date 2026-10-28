package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_EnforcesLimit(t *testing.T) {
	// 1 token per second, capacity of 2 tokens
	limiter := NewRateLimiter(1.0, 2.0)
	defer limiter.Stop()

	handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request 1: should pass (2 - 1 = 1 token left)
	req1 := httptest.NewRequest(http.MethodPost, "/api/analyze", nil)
	req1.RemoteAddr = "192.168.1.1:1234"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("request 1 expected 200, got %d", rr1.Code)
	}

	// Request 2: should pass (1 - 1 = 0 tokens left)
	req2 := httptest.NewRequest(http.MethodPost, "/api/analyze", nil)
	req2.RemoteAddr = "192.168.1.1:1234"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("request 2 expected 200, got %d", rr2.Code)
	}

	// Request 3: should fail with 429 (0 tokens available)
	req3 := httptest.NewRequest(http.MethodPost, "/api/analyze", nil)
	req3.RemoteAddr = "192.168.1.1:1234"
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusTooManyRequests {
		t.Fatalf("request 3 expected 429 Too Many Requests, got %d", rr3.Code)
	}
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	// 5 tokens per second, capacity of 1
	limiter := NewRateLimiter(10.0, 1.0)
	defer limiter.Stop()

	handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/analyze", nil)
	req.RemoteAddr = "10.0.0.1:5432"

	// 1st request succeeds
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr1.Code)
	}

	// Immediate 2nd request fails
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req)
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr2.Code)
	}

	// Wait 150ms (refills at 10/s => 1.5 tokens refilled)
	time.Sleep(150 * time.Millisecond)

	// 3rd request should succeed
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req)
	if rr3.Code != http.StatusOK {
		t.Fatalf("expected 200 after refill, got %d", rr3.Code)
	}
}

func TestRateLimiter_ConcurrencySafe(t *testing.T) {
	limiter := NewRateLimiter(100.0, 100.0)
	defer limiter.Stop()

	handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var wg sync.WaitGroup
	// Fire 50 concurrent requests simultaneously
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/analyze", nil)
			req.RemoteAddr = "172.16.0.1:9999"
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}()
	}

	wg.Wait()
	// If there were race conditions or panics, the test would crash
}

