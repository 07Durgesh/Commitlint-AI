package handler

import (
	"net/http"
	"os"

	"github.com/07Durgesh/commitlint-AI/pkg/handler"
	"github.com/07Durgesh/commitlint-AI/pkg/middleware"
	"github.com/07Durgesh/commitlint-AI/pkg/service"
)

var analyzeHandler http.Handler

func init() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	svc := service.NewGeminiService(apiKey)

	// Wrap in CORS and Panic Recovery
	h := handler.AnalyzeHandler(svc)
	analyzeHandler = middleware.CORS(middleware.Recoverer(h))
}

// Handler is the Vercel serverless entrypoint for POST /api/analyze
func Handler(w http.ResponseWriter, r *http.Request) {
	analyzeHandler.ServeHTTP(w, r)
}

