package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/redis/go-redis/v9"
)

const holdRedisKey = "qa:transaction_hold"

// HoldHandler handles the QA transaction hold toggle
type HoldHandler struct {
	redisClient *redis.Client
}

// NewHoldHandler creates a new HoldHandler
func NewHoldHandler(redisClient *redis.Client) *HoldHandler {
	return &HoldHandler{redisClient: redisClient}
}

// IsHoldEnabled checks if transaction hold is currently enabled
func (h *HoldHandler) IsHoldEnabled() bool {
	val, err := h.redisClient.Get(context.Background(), holdRedisKey).Result()
	if err != nil {
		return false
	}
	return val == "true"
}

// SetHold handles POST /api/v1/admin/hold
// Toggles the transaction hold on/off
func (h *HoldHandler) SetHold(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid JSON request body")
		return
	}

	val := "false"
	if req.Enabled {
		val = "true"
	}

	if err := h.redisClient.Set(context.Background(), holdRedisKey, val, 0).Err(); err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to set hold state")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "transaction hold updated",
		"enabled": req.Enabled,
	})
}

// GetHold handles GET /api/v1/admin/hold
// Returns the current hold state
func (h *HoldHandler) GetHold(w http.ResponseWriter, r *http.Request) {
	enabled := h.IsHoldEnabled()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled": enabled,
	})
}
