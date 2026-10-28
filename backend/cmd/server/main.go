package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/07Durgesh/commitlint-AI/internal/handler"
	"github.com/07Durgesh/commitlint-AI/internal/middleware"
	"github.com/07Durgesh/commitlint-AI/internal/service"
)

// Build metadata injected at compile time via -ldflags:
// go build -ldflags "-X main.Version=v1.0.0 -X main.Commit=$(git rev-parse --short HEAD)"
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// loadEnv reads key-value pairs from a .env file if it exists,
// setting them in the process environment without overwriting existing vars.
// This keeps our project zero-dependency while supporting local development.
func loadEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return // File does not exist, which is normal in production environments
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip blank lines and comments
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`) // Strip enclosing quotes if present

			// Only set if not already present in system environment
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
}

func main() {
	// 1. Attempt to load .env from current directory or project root
	loadEnv(".env")
	loadEnv("../../.env")
	loadEnv("../.env")

	// 2. Retrieve required configuration
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("FATAL: GEMINI_API_KEY environment variable is not set. Please set it in .env or your environment.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 3. Initialize services (Dependency Injection)
	geminiService := service.NewGeminiService(apiKey)

	// 4. Set up routes with standard library ServeMux
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", handler.HealthHandler(Version, Commit, BuildDate))
	mux.HandleFunc("/api/analyze", handler.AnalyzeHandler(geminiService))

	// 5. Build Middleware Pipeline:
	// Order of execution for an incoming HTTP request:
	// 1. CORS       (handles preflight OPTIONS immediately without rate limiting)
	// 2. Logger     (attaches X-Request-ID, measures request duration, logs status)
	// 3. Recoverer  (intercepts runtime panics, logs stack, prevents server crash)
	// 4. RateLimit  (token bucket per client IP, rejects abuse with 429)
	// 5. ServeMux   (routes to handler.AnalyzeHandler or handler.HealthHandler)
	rateLimiter := middleware.NewRateLimiter(5.0, 10.0) // 5 requests/sec replenishment, max burst of 10
	defer rateLimiter.Stop()

	var appHandler http.Handler = mux
	appHandler = rateLimiter.Limit(appHandler)
	appHandler = middleware.Recoverer(appHandler)
	appHandler = middleware.Logger(appHandler)
	appHandler = middleware.CORS(appHandler)

	addr := fmt.Sprintf(":%s", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      appHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 45 * time.Second, // Allow enough time for LLM response
		IdleTimeout:  60 * time.Second,
	}

	// 6. Start server in a background goroutine so main can listen for shutdown signals
	go func() {
		log.Printf("⚡ CommitLint-AI backend [%s (%s)] listening on http://localhost%s", Version, Commit, addr)
		log.Printf("⚡ Health check: http://localhost%s/api/health", addr)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// 7. Graceful Shutdown on SIGINT (Ctrl+C) or SIGTERM (system/container termination)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("\nShutdown signal received. Shutting down gracefully...")

	// Create a deadline to allow in-flight requests to complete
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited cleanly. Goodbye!")
}
