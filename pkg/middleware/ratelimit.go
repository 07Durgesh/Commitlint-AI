package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/07Durgesh/commitlint-AI/pkg/model"
)

type visitor struct {
	tokens     float64   // Available tokens in the bucket
	lastSeen   time.Time // Timestamp of last token replenishment
}

// RateLimiter implements a concurrent-safe Token Bucket rate limiter per IP address.
type RateLimiter struct {
	mu         sync.Mutex
	visitors   map[string]*visitor
	rate       float64       // Tokens replenished per second
	capacity   float64       // Maximum burst capacity (bucket size)
	cleanupInt time.Duration // Interval between cleanup runs
	stopCh     chan struct{} // Channel to stop background cleanup (useful in tests)
}

// NewRateLimiter creates a new rate limiter allowing 'capacity' bursts,
// replenishing at 'ratePerSecond', with periodic memory cleanup.
func NewRateLimiter(ratePerSecond float64, capacity float64) *RateLimiter {
	limiter := &RateLimiter{
		visitors:   make(map[string]*visitor),
		rate:       ratePerSecond,
		capacity:   capacity,
		cleanupInt: 3 * time.Minute,
		stopCh:     make(chan struct{}),
	}

	// Start background cleanup goroutine to prevent memory leaks from inactive IPs
	go limiter.cleanupLoop()

	return limiter
}

// Stop shuts down the background cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

// extractIP obtains the client IP, inspecting X-Forwarded-For if deployed behind a proxy.
func extractIP(r *http.Request) string {
	// If behind a proxy/CDN (Render, Cloudflare, etc.)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}

	// Direct IP:port
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// allow checks whether a request from the given IP is allowed under the token bucket algorithm.
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[ip]

	if !exists {
		// New visitor gets full bucket minus 1 token for the current request
		rl.visitors[ip] = &visitor{
			tokens:   rl.capacity - 1,
			lastSeen: now,
		}
		return true
	}

	// Calculate tokens replenished since last request
	elapsed := now.Sub(v.lastSeen).Seconds()
	v.tokens += elapsed * rl.rate
	if v.tokens > rl.capacity {
		v.tokens = rl.capacity
	}
	v.lastSeen = now

	// Check if at least 1 token is available
	if v.tokens >= 1.0 {
		v.tokens -= 1.0
		return true
	}

	return false
}

// Limit wraps an http.Handler with rate limiting.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)

		if !rl.allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "5")
			w.WriteHeader(http.StatusTooManyRequests) // 429
			json.NewEncoder(w).Encode(model.ErrorResponse{
				Error: "Too many requests. Please wait a moment before trying again.",
				Code:  http.StatusTooManyRequests,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// cleanupLoop periodically sweeps through the map and deletes inactive IPs.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupInt)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			cutoff := time.Now().Add(-5 * time.Minute)
			for ip, v := range rl.visitors {
				if v.lastSeen.Before(cutoff) {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}

