package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"payment-middleware/internal/ably"
	"payment-middleware/internal/mapper"
	"payment-middleware/internal/models"
	"payment-middleware/internal/store"
)

// TransactionHandler handles payment transaction requests
type TransactionHandler struct {
	store     store.TransactionStore
	mapper    mapper.MIDTIDMapper
	publisher ably.AblyPublisher
	hold      *HoldHandler
}

// NewTransactionHandler creates a new TransactionHandler
func NewTransactionHandler(
	store store.TransactionStore,
	mapper mapper.MIDTIDMapper,
	publisher ably.AblyPublisher,
	hold *HoldHandler,
) *TransactionHandler {
	return &TransactionHandler{
		store:     store,
		mapper:    mapper,
		publisher: publisher,
		hold:      hold,
	}
}

// HandleTransaction processes POST /api/v1/transaction requests
func (h *TransactionHandler) HandleTransaction(w http.ResponseWriter, r *http.Request) {
	// Parse JSON request body
	var req models.PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		LogError("", "", "", fmt.Sprintf("invalid JSON request body: %v", err))
		writeErrorResponse(w, http.StatusBadRequest, "invalid JSON request body")
		return
	}

	// Validate trx_id is not empty (trim whitespace)
	req.TrxID = strings.TrimSpace(req.TrxID)
	if req.TrxID == "" {
		LogError("", req.MID, req.TID, "transaction_id is required")
		writeErrorResponse(w, http.StatusBadRequest, "transaction_id is required")
		return
	}

	// Call MIDTIDMapper to resolve serial number
	serialNumber, err := h.mapper.GetSerialNumber(req.MID, req.TID)
	if err != nil {
		LogError(req.TrxID, req.MID, req.TID, fmt.Sprintf("unknown mid/tid combination: %v", err))
		writeErrorResponse(w, http.StatusNotFound, "unknown mid/tid combination")
		return
	}

	// Create Transaction with PENDING status and notification channel
	notifyChan := make(chan struct{}, 1)
	tx := &models.Transaction{
		TrxID:       req.TrxID,
		Status:      models.StatusPending,
		RequestData: req,
		NotifyChan:  notifyChan,
	}

	// Store transaction in TransactionStore
	if err := h.store.Store(req.TrxID, tx); err != nil {
		LogError(req.TrxID, req.MID, req.TID, fmt.Sprintf("failed to store transaction: %v", err))
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to store transaction: %v", err))
		return
	}

	// Check if transaction hold is enabled (QA testing feature)
	if h.hold != nil && h.hold.IsHoldEnabled() {
		log.Printf("[HOLD] Transaction %s held — not forwarding to EDC (serial: %s)", req.TrxID, serialNumber)
	} else {
		// Publish payment token to Ably with trxID metadata
		if err := h.publisher.PublishPaymentRequest(serialNumber, req.Token, req.TrxID); err != nil {
			LogError(req.TrxID, req.MID, req.TID, fmt.Sprintf("ably connection error: %v", err))
			writeErrorResponse(w, http.StatusServiceUnavailable, fmt.Sprintf("ably connection error: %v", err))
			return
		}
	}

	// Wait for EDC response using polling strategy (Redis-compatible)
	// Poll every 500ms for up to 60 seconds
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Poll transaction store for updates
			updatedTx, ok := h.store.Load(req.TrxID)
			if !ok {
				LogError(req.TrxID, req.MID, req.TID, "transaction not found during polling")
				writeErrorResponse(w, http.StatusInternalServerError, "transaction not found")
				return
			}

			// Check if status has changed from PENDING
			if updatedTx.Status != models.StatusPending {
				// Response received, return result
				if updatedTx.ResponseData != nil {
					writeJSONResponse(w, http.StatusOK, updatedTx.ResponseData)
				} else {
					LogError(req.TrxID, req.MID, req.TID, "no response data available")
					writeErrorResponse(w, http.StatusInternalServerError, "no response data available")
				}
				return
			}

		case <-ctx.Done():
			// Timeout occurred, update transaction to TIMEOUT status
			err := h.store.Update(req.TrxID, func(tx *models.Transaction) {
				tx.Status = models.StatusTimeout
			})
			if err != nil {
				// Log error but still return timeout response
				LogError(req.TrxID, req.MID, req.TID, fmt.Sprintf("error updating transaction to TIMEOUT: %v", err))
			}

			// Return 408 with error
			LogError(req.TrxID, req.MID, req.TID, "transaction timeout")
			writeErrorResponse(w, http.StatusRequestTimeout, "transaction timeout")
			return
		}
	}
}

// HandleTransactionStatus processes GET /api/v1/transaction/status/{trx_id} requests
func (h *TransactionHandler) HandleTransactionStatus(w http.ResponseWriter, r *http.Request) {
	// Extract trx_id from URL path parameter using gorilla/mux
	vars := mux.Vars(r)
	trxID := vars["trx_id"]
	if trxID == "" {
		LogError("", "", "", "transaction_id is required in path")
		writeErrorResponse(w, http.StatusBadRequest, "transaction_id is required")
		return
	}

	// Look up transaction in TransactionStore
	tx, ok := h.store.Load(trxID)
	if !ok {
		// Transaction not found
		LogError(trxID, "", "", "transaction not found")
		writeErrorResponse(w, http.StatusNotFound, "transaction not found")
		return
	}

	// Build StatusResponse
	response := models.StatusResponse{
		Status: string(tx.Status),
	}

	// Include EDC response data only if status is SUCCESS or FAILED
	if tx.Status == models.StatusSuccess || tx.Status == models.StatusFailed {
		response.Data = tx.ResponseData
	}

	// Return 200 with StatusResponse
	writeJSONResponse(w, http.StatusOK, response)
}

// writeErrorResponse writes a JSON error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(models.ErrorResponse{Error: message})
}

// writeJSONResponse writes a JSON response with the given status code
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
