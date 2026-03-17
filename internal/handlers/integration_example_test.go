package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"payment-middleware/internal/models"
	"payment-middleware/internal/store"
)

// TestIntegration_TransactionWithEDCResponse demonstrates the full flow
// from transaction initiation to EDC response handling
func TestIntegration_TransactionWithEDCResponse(t *testing.T) {
	// Setup
	txStore := store.NewSyncMapStore()
	
	// Create mock mapper
	mockMapper := &MockMapper{
		mappings: map[string]string{
			"M123:T456": "SN789",
		},
	}
	
	// Create mock Ably publisher
	mockPublisher := &MockPublisher{
		published: make([]PublishedMessage, 0),
	}
	
	// Create handlers
	transactionHandler := NewTransactionHandler(txStore, mockMapper, mockPublisher, nil)
	edcResponseHandler := NewEDCResponseHandler(txStore)
	
	// Prepare transaction request
	reqBody := models.PaymentRequest{
		Token: "payment-token-123",
		MID:   "M123",
		TID:   "T456",
		TrxID: "integration-test-trx",
	}
	reqJSON, _ := json.Marshal(reqBody)
	
	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", bytes.NewReader(reqJSON))
	rec := httptest.NewRecorder()
	
	// Start transaction handler in goroutine (it will wait for response)
	done := make(chan bool)
	go func() {
		transactionHandler.HandleTransaction(rec, req)
		done <- true
	}()
	
	// Wait a bit to ensure transaction is stored
	time.Sleep(50 * time.Millisecond)
	
	// Verify transaction is in PENDING state
	tx, ok := txStore.Load("integration-test-trx")
	if !ok {
		t.Fatal("Transaction not found in store")
	}
	if tx.Status != models.StatusPending {
		t.Errorf("Expected status PENDING, got %s", tx.Status)
	}
	
	// Simulate EDC response
	edcResponse := models.EDCResponse{
		TrxID:           "integration-test-trx",
		Status:          "success",
		Approval:        "APP123",
		Amount:          "100.00",
		CardName:        "VISA",
		ReferenceNumber: "REF456",
	}
	
	// Process EDC response
	edcResponseHandler.HandleEDCResponse(edcResponse)
	
	// Wait for transaction handler to complete
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Transaction handler did not complete")
	}
	
	// Verify HTTP response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	
	// Parse response
	var responseData models.EDCResponse
	if err := json.NewDecoder(rec.Body).Decode(&responseData); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Verify response data
	if responseData.TrxID != "integration-test-trx" {
		t.Errorf("Expected TrxID integration-test-trx, got %s", responseData.TrxID)
	}
	if responseData.Approval != "APP123" {
		t.Errorf("Expected Approval APP123, got %s", responseData.Approval)
	}
	
	// Verify final transaction state
	finalTx, ok := txStore.Load("integration-test-trx")
	if !ok {
		t.Fatal("Transaction not found after completion")
	}
	if finalTx.Status != models.StatusSuccess {
		t.Errorf("Expected final status SUCCESS, got %s", finalTx.Status)
	}
}

// TestIntegration_MultipleTransactionsConcurrent tests concurrent transaction handling
func TestIntegration_MultipleTransactionsConcurrent(t *testing.T) {
	// Setup
	txStore := store.NewSyncMapStore()
	
	mockMapper := &MockMapper{
		mappings: map[string]string{
			"M123:T456": "SN789",
		},
	}
	
	mockPublisher := &MockPublisher{
		published: make([]PublishedMessage, 0),
	}
	
	transactionHandler := NewTransactionHandler(txStore, mockMapper, mockPublisher, nil)
	edcResponseHandler := NewEDCResponseHandler(txStore)
	
	// Start 3 concurrent transactions
	numTransactions := 3
	done := make(chan bool, numTransactions)
	
	for i := 0; i < numTransactions; i++ {
		trxID := fmt.Sprintf("concurrent-trx-%d", i)
		
		go func(id string) {
			reqBody := models.PaymentRequest{
				Token: "token-" + id,
				MID:   "M123",
				TID:   "T456",
				TrxID: id,
			}
			reqJSON, _ := json.Marshal(reqBody)
			
			req := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", bytes.NewReader(reqJSON))
			rec := httptest.NewRecorder()
			
			transactionHandler.HandleTransaction(rec, req)
			done <- true
		}(trxID)
	}
	
	// Wait for all transactions to be stored
	time.Sleep(100 * time.Millisecond)
	
	// Send EDC responses for all transactions
	for i := 0; i < numTransactions; i++ {
		trxID := fmt.Sprintf("concurrent-trx-%d", i)
		edcResponse := models.EDCResponse{
			TrxID:    trxID,
			Status:   "success",
			Approval: fmt.Sprintf("APP%d", i),
		}
		edcResponseHandler.HandleEDCResponse(edcResponse)
	}
	
	// Wait for all handlers to complete
	for i := 0; i < numTransactions; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("Not all transaction handlers completed")
		}
	}
	
	// Verify all transactions are in SUCCESS state
	for i := 0; i < numTransactions; i++ {
		trxID := fmt.Sprintf("concurrent-trx-%d", i)
		tx, ok := txStore.Load(trxID)
		if !ok {
			t.Errorf("Transaction %s not found", trxID)
			continue
		}
		if tx.Status != models.StatusSuccess {
			t.Errorf("Transaction %s: expected status SUCCESS, got %s", trxID, tx.Status)
		}
	}
}
