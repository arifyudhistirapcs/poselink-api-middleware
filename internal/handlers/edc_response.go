package handlers

import (
	"fmt"
	"log"

	"payment-middleware/internal/models"
	"payment-middleware/internal/store"
)

// EDCResponseHandler handles incoming EDC device responses from Ably
type EDCResponseHandler struct {
	store store.TransactionStore
}

// NewEDCResponseHandler creates a new EDCResponseHandler
func NewEDCResponseHandler(store store.TransactionStore) *EDCResponseHandler {
	return &EDCResponseHandler{
		store: store,
	}
}

// HandleEDCResponse processes an EDC response message from Ably
// This function is designed to be passed as a callback to AblyPublisher.SubscribeToResponses
func (h *EDCResponseHandler) HandleEDCResponse(response models.EDCResponse) {
	// Extract trx_id from EDC response
	trxID := response.TrxID
	if trxID == "" {
		LogError("", "", "", "EDC response missing trx_id field")
		return
	}

	// Look up transaction in TransactionStore
	tx, ok := h.store.Load(trxID)
	if !ok {
		LogError(trxID, "", "", "transaction not found")
		return
	}

	// Determine success/failure based on EDC response status field
	// EDC may return "success", "SUCCESS", or "paid" (for QRIS transactions) as success indicators
	var newStatus models.TransactionStatus
	if response.Status == "success" || response.Status == "SUCCESS" || response.Status == "paid" {
		newStatus = models.StatusSuccess
	} else {
		newStatus = models.StatusFailed
	}

	// Update transaction state and store complete EDC response data
	err := h.store.Update(trxID, func(tx *models.Transaction) {
		tx.Status = newStatus
		tx.ResponseData = &response
	})

	if err != nil {
		// Use the loaded transaction data for logging
		mid := tx.RequestData.MID
		tid := tx.RequestData.TID
		LogError(trxID, mid, tid, fmt.Sprintf("error updating transaction: %v", err))
		return
	}

	// Signal notification channel to unblock waiting request handler
	// Use non-blocking send to avoid deadlock if no one is waiting
	if tx.NotifyChan != nil {
		select {
		case tx.NotifyChan <- struct{}{}:
			// Successfully notified
			log.Printf("Successfully processed EDC response for transaction %s with status %s", trxID, newStatus)
		default:
			// Channel full or no receiver, which is fine
			// This can happen if the request already timed out
			log.Printf("Processed EDC response for transaction %s with status %s (no waiting handler)", trxID, newStatus)
		}
	} else {
		log.Printf("Processed EDC response for transaction %s with status %s (no notification channel)", trxID, newStatus)
	}
}
