package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"payment-middleware/internal/mapper"
)

// AdminHandler handles administrative operations for mapping management
type AdminHandler struct {
	mapper      *mapper.RedisMIDTIDMapper
	redisClient *redis.Client
}

// NewAdminHandler creates a new AdminHandler
func NewAdminHandler(mapper *mapper.RedisMIDTIDMapper, redisClient *redis.Client) *AdminHandler {
	return &AdminHandler{
		mapper:      mapper,
		redisClient: redisClient,
	}
}

// MappingRequest represents the request body for creating/updating a mapping
type MappingRequest struct {
	MID          string `json:"mid"`
	TID          string `json:"tid"`
	SerialNumber string `json:"serial_number"`
}

// CreateOrUpdateMapping handles POST /api/v1/admin/mapping
// Creates or updates a MID/TID to serial number mapping
func (h *AdminHandler) CreateOrUpdateMapping(w http.ResponseWriter, r *http.Request) {
	// Parse JSON request body
	var req MappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		LogError("", "", "", "invalid JSON request body")
		WriteErrorResponse(w, http.StatusBadRequest, "invalid JSON request body")
		return
	}

	// Validate required fields
	if req.MID == "" {
		LogError("", "", "", "mid is required")
		WriteErrorResponse(w, http.StatusBadRequest, "mid is required")
		return
	}
	if req.TID == "" {
		LogError("", req.MID, "", "tid is required")
		WriteErrorResponse(w, http.StatusBadRequest, "tid is required")
		return
	}
	if req.SerialNumber == "" {
		LogError("", req.MID, req.TID, "serial_number is required")
		WriteErrorResponse(w, http.StatusBadRequest, "serial_number is required")
		return
	}

	// Store mapping in Redis
	if err := h.mapper.SetMapping(req.MID, req.TID, req.SerialNumber); err != nil {
		LogError("", req.MID, req.TID, "failed to store mapping")
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to store mapping")
		return
	}

	// Return 200 success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "mapping created/updated successfully",
	})
}

// DeleteMapping handles DELETE /api/v1/admin/mapping
// Deletes a MID/TID mapping using query parameters
func (h *AdminHandler) DeleteMapping(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	mid := r.URL.Query().Get("mid")
	tid := r.URL.Query().Get("tid")

	// Validate required parameters
	if mid == "" {
		LogError("", "", "", "mid query parameter is required")
		WriteErrorResponse(w, http.StatusBadRequest, "mid query parameter is required")
		return
	}
	if tid == "" {
		LogError("", mid, "", "tid query parameter is required")
		WriteErrorResponse(w, http.StatusBadRequest, "tid query parameter is required")
		return
	}

	// Delete mapping from Redis
	if err := h.mapper.DeleteMapping(mid, tid); err != nil {
		LogError("", mid, tid, "failed to delete mapping")
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to delete mapping")
		return
	}

	// Return 200 success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "mapping deleted successfully",
	})
}

// MigrateRequest represents the request body for migration endpoint
type MigrateRequest struct {
	Force bool `json:"force"`
}

// MigrateMapping handles POST /api/v1/admin/migrate
// Migrates mappings from MIDTID_MAPPINGS config to Redis
func (h *AdminHandler) MigrateMapping(w http.ResponseWriter, r *http.Request, mappings map[string]string) {
	// Parse JSON request body for force flag
	var req MigrateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If body is empty or invalid, default to force=false
		req.Force = false
	}

	successCount := 0
	errorCount := 0
	var errors []string

	// Iterate through mappings from config
	for key, serialNumber := range mappings {
		// Parse key format "mid:tid"
		parts := strings.Split(key, ":")
		if len(parts) != 2 {
			errorMsg := fmt.Sprintf("invalid mapping key format: %s (expected 'mid:tid')", key)
			errors = append(errors, errorMsg)
			errorCount++
			continue
		}

		mid := parts[0]
		tid := parts[1]

		// Check if mapping already exists (unless force flag is set)
		if !req.Force {
			_, err := h.mapper.GetSerialNumber(mid, tid)
			if err == nil {
				// Mapping exists, skip
				continue
			}
		}

		// Store mapping in Redis
		if err := h.mapper.SetMapping(mid, tid, serialNumber); err != nil {
			errorMsg := fmt.Sprintf("failed to migrate mapping %s:%s: %v", mid, tid, err)
			errors = append(errors, errorMsg)
			errorCount++
			continue
		}

		successCount++
	}

	// Return response with migration results
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       fmt.Sprintf("migration completed: %d successful, %d errors", successCount, errorCount),
		"success_count": successCount,
		"error_count":   errorCount,
		"errors":        errors,
	})
}

// GetTransactionTTL handles GET /api/v1/admin/transaction/{trx_id}/ttl
// Returns the remaining TTL for a transaction in seconds
func (h *AdminHandler) GetTransactionTTL(w http.ResponseWriter, r *http.Request) {
	// Extract trx_id from URL path
	// Expected format: /api/v1/admin/transaction/{trx_id}/ttl
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 6 {
		LogError("", "", "", "invalid URL path for TTL endpoint")
		WriteErrorResponse(w, http.StatusBadRequest, "invalid URL path")
		return
	}
	trxID := pathParts[5]

	if trxID == "" {
		LogError("", "", "", "trx_id is required")
		WriteErrorResponse(w, http.StatusBadRequest, "trx_id is required")
		return
	}

	// Get TTL from Redis
	ctx := context.Background()
	key := fmt.Sprintf("transaction:%s", trxID)
	ttl, err := h.redisClient.TTL(ctx, key).Result()
	if err != nil {
		LogError(trxID, "", "", fmt.Sprintf("failed to get TTL: %v", err))
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to get TTL")
		return
	}

	// Check if key exists
	if ttl == -2 {
		// Key doesn't exist
		LogError(trxID, "", "", "transaction not found")
		WriteErrorResponse(w, http.StatusNotFound, "transaction not found")
		return
	}

	// Return TTL in seconds
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"trx_id":            trxID,
		"ttl_seconds":       int64(ttl.Seconds()),
		"has_expiration":    ttl > 0,
	})
}

// ExtendTTLRequest represents the request body for extending TTL
type ExtendTTLRequest struct {
	DurationSeconds int64 `json:"duration_seconds"`
}

// ExtendTransactionTTL handles POST /api/v1/admin/transaction/{trx_id}/extend-ttl
// Extends the TTL for a transaction by the specified duration
func (h *AdminHandler) ExtendTransactionTTL(w http.ResponseWriter, r *http.Request) {
	// Extract trx_id from URL path
	// Expected format: /api/v1/admin/transaction/{trx_id}/extend-ttl
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 6 {
		LogError("", "", "", "invalid URL path for extend-ttl endpoint")
		WriteErrorResponse(w, http.StatusBadRequest, "invalid URL path")
		return
	}
	trxID := pathParts[5]

	if trxID == "" {
		LogError("", "", "", "trx_id is required")
		WriteErrorResponse(w, http.StatusBadRequest, "trx_id is required")
		return
	}

	// Parse JSON request body
	var req ExtendTTLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		LogError(trxID, "", "", "invalid JSON request body")
		WriteErrorResponse(w, http.StatusBadRequest, "invalid JSON request body")
		return
	}

	// Validate duration
	if req.DurationSeconds <= 0 {
		LogError(trxID, "", "", "duration_seconds must be positive")
		WriteErrorResponse(w, http.StatusBadRequest, "duration_seconds must be positive")
		return
	}

	// Check if transaction exists
	ctx := context.Background()
	key := fmt.Sprintf("transaction:%s", trxID)
	exists, err := h.redisClient.Exists(ctx, key).Result()
	if err != nil {
		LogError(trxID, "", "", fmt.Sprintf("failed to check transaction existence: %v", err))
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to check transaction existence")
		return
	}

	if exists == 0 {
		LogError(trxID, "", "", "transaction not found")
		WriteErrorResponse(w, http.StatusNotFound, "transaction not found")
		return
	}

	// Extend TTL using EXPIRE command
	err = h.redisClient.Expire(ctx, key, time.Duration(req.DurationSeconds)*time.Second).Err()
	if err != nil {
		LogError(trxID, "", "", fmt.Sprintf("failed to extend TTL: %v", err))
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to extend TTL")
		return
	}

	// Get new TTL to confirm
	newTTL, err := h.redisClient.TTL(ctx, key).Result()
	if err != nil {
		LogError(trxID, "", "", fmt.Sprintf("failed to get new TTL: %v", err))
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to get new TTL")
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":           "TTL extended successfully",
		"trx_id":            trxID,
		"new_ttl_seconds":   int64(newTTL.Seconds()),
		"extended_by":       req.DurationSeconds,
	})
}
