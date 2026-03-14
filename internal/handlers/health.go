package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"payment-middleware/internal/store"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	redisClient *store.RedisClient
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(redisClient *store.RedisClient) *HealthHandler {
	return &HealthHandler{
		redisClient: redisClient,
	}
}

// CheckHealth handles GET /health
// Returns 200 if Redis is reachable, 503 otherwise
func (h *HealthHandler) CheckHealth(w http.ResponseWriter, r *http.Request) {
	// Create context with timeout for health check
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Ping Redis to verify connectivity
	err := h.redisClient.Ping(ctx)
	
	w.Header().Set("Content-Type", "application/json")
	
	if err != nil {
		// Redis is unavailable
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error":  "Redis unavailable",
		})
		return
	}

	// Redis is available
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}
