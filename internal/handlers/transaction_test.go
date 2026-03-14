package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"payment-middleware/internal/models"
	"payment-middleware/internal/store"
)

// MockMapper implements mapper.MIDTIDMapper for testing
type MockMapper struct {
	mappings map[string]string
}

func NewMockMapper(mappings map[string]string) *MockMapper {
	return &MockMapper{mappings: mappings}
}

func (m *MockMapper) GetSerialNumber(mid, tid string) (string, error) {
	key := mid + ":" + tid
	if serial, ok := m.mappings[key]; ok {
		return serial, nil
	}
	return "", &mapperError{msg: "unknown mid/tid combination"}
}

type mapperError struct {
	msg string
}

func (e *mapperError) Error() string {
	return e.msg
}

// MockPublisher implements ably.AblyPublisher for testing
type MockPublisher struct {
	mu             sync.Mutex
	published      []PublishedMessage
	publishError   error
	subscribeError error
}

type PublishedMessage struct {
	SerialNumber string
	Token        string
}

func NewMockPublisher() *MockPublisher {
	return &MockPublisher{
		published: make([]PublishedMessage, 0),
	}
}

func (m *MockPublisher) PublishPaymentRequest(serialNumber, token, trxID string) error {
	if m.publishError != nil {
		return m.publishError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, PublishedMessage{
		SerialNumber: serialNumber,
		Token:        token,
	})
	return nil
}

func (m *MockPublisher) SubscribeToResponses(handler func(response models.EDCResponse)) error {
	return m.subscribeError
}

func TestHandleTransaction_Success(t *testing.T) {
	// Setup
	store := store.NewSyncMapStore()
	mapper := NewMockMapper(map[string]string{
		"M123:T456": "SN789",
	})
	publisher := NewMockPublisher()
	handler := NewTransactionHandler(store, mapper, publisher)

	// Create request
	reqBody := models.PaymentRequest{
		Token: "test_token",
		MID:   "M123",
		TID:   "T456",
		TrxID: "TRX001",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	// Simulate EDC response in a goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		store.Update("TRX001", func(tx *models.Transaction) {
			tx.Status = models.StatusSuccess
			tx.ResponseData = &models.EDCResponse{
				TrxID:    "TRX001",
				Status:   "success",
				Approval: "123456",
			}
		})
		// Load and signal
		if tx, ok := store.Load("TRX001"); ok && tx.NotifyChan != nil {
			close(tx.NotifyChan)
		}
	}()

	// Execute
	handler.HandleTransaction(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response models.EDCResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.TrxID != "TRX001" {
		t.Errorf("Expected TrxID TRX001, got %s", response.TrxID)
	}
	if response.Status != "success" {
		t.Errorf("Expected status success, got %s", response.Status)
	}
}

func TestHandleTransaction_MissingTrxID(t *testing.T) {
	// Setup
	store := store.NewSyncMapStore()
	mapper := NewMockMapper(map[string]string{})
	publisher := NewMockPublisher()
	handler := NewTransactionHandler(store, mapper, publisher)

	// Create request with empty trx_id
	reqBody := models.PaymentRequest{
		Token: "test_token",
		MID:   "M123",
		TID:   "T456",
		TrxID: "",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	// Execute
	handler.HandleTransaction(rec, req)

	// Assert
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var response models.ErrorResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.Error != "transaction_id is required" {
		t.Errorf("Expected error message 'transaction_id is required', got %s", response.Error)
	}
}

func TestHandleTransaction_WhitespaceOnlyTrxID(t *testing.T) {
	// Setup
	store := store.NewSyncMapStore()
	mapper := NewMockMapper(map[string]string{})
	publisher := NewMockPublisher()
	handler := NewTransactionHandler(store, mapper, publisher)

	// Create request with whitespace-only trx_id
	reqBody := models.PaymentRequest{
		Token: "test_token",
		MID:   "M123",
		TID:   "T456",
		TrxID: "   ",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	// Execute
	handler.HandleTransaction(rec, req)

	// Assert
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestHandleTransaction_UnknownMIDTID(t *testing.T) {
	// Setup
	store := store.NewSyncMapStore()
	mapper := NewMockMapper(map[string]string{
		"M123:T456": "SN789",
	})
	publisher := NewMockPublisher()
	handler := NewTransactionHandler(store, mapper, publisher)

	// Create request with unknown MID/TID
	reqBody := models.PaymentRequest{
		Token: "test_token",
		MID:   "M999",
		TID:   "T999",
		TrxID: "TRX001",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	// Execute
	handler.HandleTransaction(rec, req)

	// Assert
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}

	var response models.ErrorResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.Error != "unknown mid/tid combination" {
		t.Errorf("Expected error message 'unknown mid/tid combination', got %s", response.Error)
	}
}

func TestHandleTransaction_AblyPublishError(t *testing.T) {
	// Setup
	store := store.NewSyncMapStore()
	mapper := NewMockMapper(map[string]string{
		"M123:T456": "SN789",
	})
	publisher := NewMockPublisher()
	publisher.publishError = &ablyError{msg: "connection failed"}
	handler := NewTransactionHandler(store, mapper, publisher)

	// Create request
	reqBody := models.PaymentRequest{
		Token: "test_token",
		MID:   "M123",
		TID:   "T456",
		TrxID: "TRX001",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	// Execute
	handler.HandleTransaction(rec, req)

	// Assert
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", rec.Code)
	}

	var response models.ErrorResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if !contains(response.Error, "ably connection error") {
		t.Errorf("Expected error message to contain 'ably connection error', got %s", response.Error)
	}
}

type ablyError struct {
	msg string
}

func (e *ablyError) Error() string {
	return e.msg
}

func TestHandleTransaction_Timeout(t *testing.T) {
	// Skip this test by default as it takes 60 seconds
	// Run with: go test -v -run TestHandleTransaction_Timeout
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	// Setup
	store := store.NewSyncMapStore()
	mapper := NewMockMapper(map[string]string{
		"M123:T456": "SN789",
	})
	publisher := NewMockPublisher()
	handler := NewTransactionHandler(store, mapper, publisher)

	// Create request
	reqBody := models.PaymentRequest{
		Token: "test_token",
		MID:   "M123",
		TID:   "T456",
		TrxID: "TRX001",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	// Don't simulate EDC response - let it timeout
	// Note: This test would take 60 seconds in real scenario
	// For testing, we'll need to mock the timeout or use a shorter duration
	// For now, we'll skip the actual execution and just verify the logic

	// In a real test, you'd want to inject a configurable timeout
	// For this test, we'll just verify the transaction was stored
	handler.HandleTransaction(rec, req)

	// The handler will timeout after 60 seconds
	// In production tests, you'd want to make the timeout configurable
	if rec.Code != http.StatusRequestTimeout {
		t.Errorf("Expected status 408, got %d", rec.Code)
	}

	var response models.ErrorResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.Error != "transaction timeout" {
		t.Errorf("Expected error message 'transaction timeout', got %s", response.Error)
	}

	// Verify transaction status was updated to TIMEOUT
	tx, ok := store.Load("TRX001")
	if !ok {
		t.Error("Transaction should exist in store")
	}
	if tx.Status != models.StatusTimeout {
		t.Errorf("Expected status TIMEOUT, got %s", tx.Status)
	}
}

func TestHandleTransaction_InvalidJSON(t *testing.T) {
	// Setup
	store := store.NewSyncMapStore()
	mapper := NewMockMapper(map[string]string{})
	publisher := NewMockPublisher()
	handler := NewTransactionHandler(store, mapper, publisher)

	// Create request with invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", bytes.NewReader([]byte("invalid json")))
	rec := httptest.NewRecorder()

	// Execute
	handler.HandleTransaction(rec, req)

	// Assert
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var response models.ErrorResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.Error != "invalid JSON request body" {
		t.Errorf("Expected error message 'invalid JSON request body', got %s", response.Error)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestHandleTransactionStatus_Success tests successful retrieval of a transaction
func TestHandleTransactionStatus_Success(t *testing.T) {
	// Setup
	store := store.NewSyncMapStore()
	mapper := NewMockMapper(map[string]string{})
	publisher := NewMockPublisher()
	handler := NewTransactionHandler(store, mapper, publisher)

	// Store a transaction with SUCCESS status
	tx := &models.Transaction{
		TrxID:  "TRX001",
		Status: models.StatusSuccess,
		ResponseData: &models.EDCResponse{
			TrxID:    "TRX001",
			Status:   "success",
			Approval: "123456",
			Amount:   "100.00",
		},
	}
	store.Store("TRX001", tx)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transaction/status/TRX001", nil)
	req.SetPathValue("trx_id", "TRX001")
	rec := httptest.NewRecorder()

	// Execute
	handler.HandleTransactionStatus(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response models.StatusResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.Status != "SUCCESS" {
		t.Errorf("Expected status SUCCESS, got %s", response.Status)
	}
	if response.Data == nil {
		t.Error("Expected data to be included for SUCCESS status")
	}
	if response.Data.TrxID != "TRX001" {
		t.Errorf("Expected TrxID TRX001, got %s", response.Data.TrxID)
	}
}

// TestHandleTransactionStatus_Failed tests retrieval of a failed transaction
func TestHandleTransactionStatus_Failed(t *testing.T) {
	// Setup
	store := store.NewSyncMapStore()
	mapper := NewMockMapper(map[string]string{})
	publisher := NewMockPublisher()
	handler := NewTransactionHandler(store, mapper, publisher)

	// Store a transaction with FAILED status
	tx := &models.Transaction{
		TrxID:  "TRX002",
		Status: models.StatusFailed,
		ResponseData: &models.EDCResponse{
			TrxID:  "TRX002",
			Status: "failed",
			Msg:    "Card declined",
		},
	}
	store.Store("TRX002", tx)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transaction/status/TRX002", nil)
	req.SetPathValue("trx_id", "TRX002")
	rec := httptest.NewRecorder()

	// Execute
	handler.HandleTransactionStatus(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response models.StatusResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.Status != "FAILED" {
		t.Errorf("Expected status FAILED, got %s", response.Status)
	}
	if response.Data == nil {
		t.Error("Expected data to be included for FAILED status")
	}
}

// TestHandleTransactionStatus_Pending tests retrieval of a pending transaction
func TestHandleTransactionStatus_Pending(t *testing.T) {
	// Setup
	store := store.NewSyncMapStore()
	mapper := NewMockMapper(map[string]string{})
	publisher := NewMockPublisher()
	handler := NewTransactionHandler(store, mapper, publisher)

	// Store a transaction with PENDING status
	tx := &models.Transaction{
		TrxID:  "TRX003",
		Status: models.StatusPending,
	}
	store.Store("TRX003", tx)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transaction/status/TRX003", nil)
	req.SetPathValue("trx_id", "TRX003")
	rec := httptest.NewRecorder()

	// Execute
	handler.HandleTransactionStatus(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response models.StatusResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.Status != "PENDING" {
		t.Errorf("Expected status PENDING, got %s", response.Status)
	}
	if response.Data != nil {
		t.Error("Expected data to be nil for PENDING status")
	}
}

// TestHandleTransactionStatus_Timeout tests retrieval of a timed out transaction
func TestHandleTransactionStatus_Timeout(t *testing.T) {
	// Setup
	store := store.NewSyncMapStore()
	mapper := NewMockMapper(map[string]string{})
	publisher := NewMockPublisher()
	handler := NewTransactionHandler(store, mapper, publisher)

	// Store a transaction with TIMEOUT status
	tx := &models.Transaction{
		TrxID:  "TRX004",
		Status: models.StatusTimeout,
	}
	store.Store("TRX004", tx)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transaction/status/TRX004", nil)
	req.SetPathValue("trx_id", "TRX004")
	rec := httptest.NewRecorder()

	// Execute
	handler.HandleTransactionStatus(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response models.StatusResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.Status != "TIMEOUT" {
		t.Errorf("Expected status TIMEOUT, got %s", response.Status)
	}
	if response.Data != nil {
		t.Error("Expected data to be nil for TIMEOUT status")
	}
}

// TestHandleTransactionStatus_NotFound tests retrieval of a non-existent transaction
func TestHandleTransactionStatus_NotFound(t *testing.T) {
	// Setup
	store := store.NewSyncMapStore()
	mapper := NewMockMapper(map[string]string{})
	publisher := NewMockPublisher()
	handler := NewTransactionHandler(store, mapper, publisher)

	// Create request for non-existent transaction
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transaction/status/TRX999", nil)
	req.SetPathValue("trx_id", "TRX999")
	rec := httptest.NewRecorder()

	// Execute
	handler.HandleTransactionStatus(rec, req)

	// Assert
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}

	var response models.ErrorResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.Error != "transaction not found" {
		t.Errorf("Expected error message 'transaction not found', got %s", response.Error)
	}
}

// TestHandleTransactionStatus_AllStatuses tests that all status values are valid
func TestHandleTransactionStatus_AllStatuses(t *testing.T) {
	// Setup
	store := store.NewSyncMapStore()
	mapper := NewMockMapper(map[string]string{})
	publisher := NewMockPublisher()
	handler := NewTransactionHandler(store, mapper, publisher)

	// Test all valid status values
	statuses := []models.TransactionStatus{
		models.StatusPending,
		models.StatusSuccess,
		models.StatusFailed,
		models.StatusTimeout,
	}

	for i, status := range statuses {
		trxID := "TRX" + string(rune('0'+i))
		tx := &models.Transaction{
			TrxID:  trxID,
			Status: status,
		}
		if status == models.StatusSuccess || status == models.StatusFailed {
			tx.ResponseData = &models.EDCResponse{
				TrxID:  trxID,
				Status: "test",
			}
		}
		store.Store(trxID, tx)

		// Create request
		req := httptest.NewRequest(http.MethodGet, "/api/v1/transaction/status/"+trxID, nil)
		req.SetPathValue("trx_id", trxID)
		rec := httptest.NewRecorder()

		// Execute
		handler.HandleTransactionStatus(rec, req)

		// Assert
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for %s, got %d", status, rec.Code)
		}

		var response models.StatusResponse
		json.NewDecoder(rec.Body).Decode(&response)
		if response.Status != string(status) {
			t.Errorf("Expected status %s, got %s", status, response.Status)
		}

		// Verify data inclusion based on status
		if status == models.StatusSuccess || status == models.StatusFailed {
			if response.Data == nil {
				t.Errorf("Expected data to be included for %s status", status)
			}
		} else {
			if response.Data != nil {
				t.Errorf("Expected data to be nil for %s status", status)
			}
		}
	}
}
