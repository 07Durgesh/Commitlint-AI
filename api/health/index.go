package handler

import (
	"net/http"

	"github.com/07Durgesh/commitlint-AI/internal/handler"
	"github.com/07Durgesh/commitlint-AI/internal/middleware"
)

var healthHandler http.Handler

func init() {
	h := handler.HealthHandler("v1.0.0 (Vercel Serverless)", "live", "production")
	healthHandler = middleware.CORS(h)
}

// Handler is the Vercel serverless entrypoint for GET /api/health
func Handler(w http.ResponseWriter, r *http.Request) {
	healthHandler.ServeHTTP(w, r)
}
