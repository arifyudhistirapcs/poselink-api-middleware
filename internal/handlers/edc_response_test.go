package handlers

import (
	"testing"
	"time"

	"payment-middleware/internal/models"
	"payment-middleware/internal/store"
)

func TestHandleEDCResponse_Success(t *testing.T) {
	// Setup
	txStore := store.NewSyncMapStore()
	handler := NewEDCResponseHandler(txStore)

	// Create a pending transaction
	notifyChan := make(chan struct{}, 1)
	tx := &models.Transaction{
		TrxID:      "test-trx-123",
		Status:     models.StatusPending,
		NotifyChan: notifyChan,
	}
	txStore.Store("test-trx-123", tx)

	// Create a successful EDC response
	edcResponse := models.EDCResponse{
		TrxID:           "test-trx-123",
		Status:          "success",
		Approval:        "123456",
		Amount:          "100.00",
		CardName:        "VISA",
		ReferenceNumber: "REF123",
	}

	// Act
	handler.HandleEDCResponse(edcResponse)

	// Assert - Check transaction was updated
	updatedTx, ok := txStore.Load("test-trx-123")
	if !ok {
		t.Fatal("Transaction not found after update")
	}

	if updatedTx.Status != models.StatusSuccess {
		t.Errorf("Expected status SUCCESS, got %s", updatedTx.Status)
	}

	if updatedTx.ResponseData == nil {
		t.Fatal("ResponseData should not be nil")
	}

	if updatedTx.ResponseData.TrxID != "test-trx-123" {
		t.Errorf("Expected TrxID test-trx-123, got %s", updatedTx.ResponseData.TrxID)
	}

	if updatedTx.ResponseData.Approval != "123456" {
		t.Errorf("Expected Approval 123456, got %s", updatedTx.ResponseData.Approval)
	}

	// Assert - Check notification channel was signaled
	select {
	case <-notifyChan:
		// Success - channel was signaled
	case <-time.After(100 * time.Millisecond):
		t.Error("Notification channel was not signaled")
	}
}

func TestHandleEDCResponse_Failure(t *testing.T) {
	// Setup
	txStore := store.NewSyncMapStore()
	handler := NewEDCResponseHandler(txStore)

	// Create a pending transaction
	notifyChan := make(chan struct{}, 1)
	tx := &models.Transaction{
		TrxID:      "test-trx-456",
		Status:     models.StatusPending,
		NotifyChan: notifyChan,
	}
	txStore.Store("test-trx-456", tx)

	// Create a failed EDC response
	edcResponse := models.EDCResponse{
		TrxID:  "test-trx-456",
		Status: "failed",
		Msg:    "Insufficient funds",
		RC:     "51",
	}

	// Act
	handler.HandleEDCResponse(edcResponse)

	// Assert - Check transaction was updated to FAILED
	updatedTx, ok := txStore.Load("test-trx-456")
	if !ok {
		t.Fatal("Transaction not found after update")
	}

	if updatedTx.Status != models.StatusFailed {
		t.Errorf("Expected status FAILED, got %s", updatedTx.Status)
	}

	if updatedTx.ResponseData == nil {
		t.Fatal("ResponseData should not be nil")
	}

	if updatedTx.ResponseData.Msg != "Insufficient funds" {
		t.Errorf("Expected Msg 'Insufficient funds', got %s", updatedTx.ResponseData.Msg)
	}

	// Assert - Check notification channel was signaled
	select {
	case <-notifyChan:
		// Success - channel was signaled
	case <-time.After(100 * time.Millisecond):
		t.Error("Notification channel was not signaled")
	}
}

func TestHandleEDCResponse_UppercaseStatus(t *testing.T) {
	// Setup
	txStore := store.NewSyncMapStore()
	handler := NewEDCResponseHandler(txStore)

	// Create a pending transaction
	notifyChan := make(chan struct{}, 1)
	tx := &models.Transaction{
		TrxID:      "test-trx-789",
		Status:     models.StatusPending,
		NotifyChan: notifyChan,
	}
	txStore.Store("test-trx-789", tx)

	// Create an EDC response with uppercase SUCCESS
	edcResponse := models.EDCResponse{
		TrxID:    "test-trx-789",
		Status:   "SUCCESS",
		Approval: "789012",
	}

	// Act
	handler.HandleEDCResponse(edcResponse)

	// Assert - Check transaction was updated to SUCCESS
	updatedTx, ok := txStore.Load("test-trx-789")
	if !ok {
		t.Fatal("Transaction not found after update")
	}

	if updatedTx.Status != models.StatusSuccess {
		t.Errorf("Expected status SUCCESS, got %s", updatedTx.Status)
	}
}

func TestHandleEDCResponse_TransactionNotFound(t *testing.T) {
	// Setup
	txStore := store.NewSyncMapStore()
	handler := NewEDCResponseHandler(txStore)

	// Create an EDC response for a non-existent transaction
	edcResponse := models.EDCResponse{
		TrxID:  "non-existent-trx",
		Status: "success",
	}

	// Act - should not panic, just log error
	handler.HandleEDCResponse(edcResponse)

	// Assert - transaction should still not exist
	_, ok := txStore.Load("non-existent-trx")
	if ok {
		t.Error("Transaction should not exist")
	}
}

func TestHandleEDCResponse_MissingTrxID(t *testing.T) {
	// Setup
	txStore := store.NewSyncMapStore()
	handler := NewEDCResponseHandler(txStore)

	// Create an EDC response with missing trx_id
	edcResponse := models.EDCResponse{
		Status:   "success",
		Approval: "123456",
	}

	// Act - should not panic, just log error
	handler.HandleEDCResponse(edcResponse)

	// No assertions needed - just verify it doesn't panic
}

func TestHandleEDCResponse_CompleteDataStorage(t *testing.T) {
	// Setup
	txStore := store.NewSyncMapStore()
	handler := NewEDCResponseHandler(txStore)

	// Create a pending transaction
	notifyChan := make(chan struct{}, 1)
	tx := &models.Transaction{
		TrxID:      "test-trx-complete",
		Status:     models.StatusPending,
		NotifyChan: notifyChan,
	}
	txStore.Store("test-trx-complete", tx)

	// Create a complete EDC response with all fields
	edcResponse := models.EDCResponse{
		TrxID:           "test-trx-complete",
		Status:          "success",
		AcqMID:          "MID123",
		AcqTID:          "TID456",
		Action:          "SALE",
		Amount:          "250.50",
		Approval:        "APP789",
		BatchNumber:     "BATCH001",
		CardCategory:    "CREDIT",
		CardName:        "MASTERCARD",
		CardType:        "DEBIT",
		EDCAddress:      "EDC_ADDR_1",
		IsCredit:        "true",
		IsOffUs:         "false",
		Method:          "CHIP",
		Msg:             "Transaction approved",
		PAN:             "****1234",
		Periode:         "12/25",
		Plan:            "REGULAR",
		POSAddress:      "POS_ADDR_1",
		RC:              "00",
		ReferenceNumber: "REF987654",
		TraceNumber:     "TRACE123",
		TransactionDate: "2024-01-15",
	}

	// Act
	handler.HandleEDCResponse(edcResponse)

	// Assert - Check all fields are preserved
	updatedTx, ok := txStore.Load("test-trx-complete")
	if !ok {
		t.Fatal("Transaction not found after update")
	}

	if updatedTx.ResponseData == nil {
		t.Fatal("ResponseData should not be nil")
	}

	resp := updatedTx.ResponseData

	// Verify all fields are stored correctly
	if resp.AcqMID != "MID123" {
		t.Errorf("Expected AcqMID MID123, got %s", resp.AcqMID)
	}
	if resp.Amount != "250.50" {
		t.Errorf("Expected Amount 250.50, got %s", resp.Amount)
	}
	if resp.CardName != "MASTERCARD" {
		t.Errorf("Expected CardName MASTERCARD, got %s", resp.CardName)
	}
	if resp.ReferenceNumber != "REF987654" {
		t.Errorf("Expected ReferenceNumber REF987654, got %s", resp.ReferenceNumber)
	}
}

func TestHandleEDCResponse_NotificationChannelNonBlocking(t *testing.T) {
	// Setup
	txStore := store.NewSyncMapStore()
	handler := NewEDCResponseHandler(txStore)

	// Create a pending transaction with a full channel (simulating timeout scenario)
	notifyChan := make(chan struct{}, 1)
	notifyChan <- struct{}{} // Fill the channel
	tx := &models.Transaction{
		TrxID:      "test-trx-nonblock",
		Status:     models.StatusPending,
		NotifyChan: notifyChan,
	}
	txStore.Store("test-trx-nonblock", tx)

	// Create an EDC response
	edcResponse := models.EDCResponse{
		TrxID:  "test-trx-nonblock",
		Status: "success",
	}

	// Act - should not block even though channel is full
	done := make(chan bool)
	go func() {
		handler.HandleEDCResponse(edcResponse)
		done <- true
	}()

	// Assert - handler should complete quickly without blocking
	select {
	case <-done:
		// Success - handler completed
	case <-time.After(100 * time.Millisecond):
		t.Error("Handler blocked on notification channel")
	}

	// Verify transaction was still updated
	updatedTx, ok := txStore.Load("test-trx-nonblock")
	if !ok {
		t.Fatal("Transaction not found after update")
	}
	if updatedTx.Status != models.StatusSuccess {
		t.Errorf("Expected status SUCCESS, got %s", updatedTx.Status)
	}
}
