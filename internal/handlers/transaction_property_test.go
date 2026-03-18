package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"payment-middleware/internal/models"
	"payment-middleware/internal/store"
)

// Feature: payment-middleware, Property 1: Valid POS Request Parsing
// Validates: Requirements 1.2, 9.1
func TestProperty_ValidPOSRequestParsing(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)
	properties.Property("valid POS requests should parse successfully", prop.ForAll(
		func(token, mid, tid, trxID string) bool {
			// Skip empty trx_id as that's tested separately
			if strings.TrimSpace(trxID) == "" {
				return true
			}

			req := models.PaymentRequest{
				Token: token,
				MID:   mid,
				TID:   tid,
				TrxID: trxID,
			}

			// Marshal and unmarshal to verify JSON handling
			data, err := json.Marshal(req)
			if err != nil {
				return false
			}

			var parsed models.PaymentRequest
			err = json.Unmarshal(data, &parsed)
			if err != nil {
				return false
			}

			// Verify all fields are preserved
			return parsed.Token == token &&
				parsed.MID == mid &&
				parsed.TID == tid &&
				parsed.TrxID == trxID
		},
		gen.AnyString(),
		gen.AnyString(),
		gen.AnyString(),
		gen.Identifier(), // Use identifier to ensure non-empty trx_id
	))

	properties.TestingRun(t)
}

// Feature: payment-middleware, Property 2: Missing Transaction ID Rejection
// Validates: Requirements 1.3, 8.2
func TestProperty_MissingTransactionIDRejection(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)
	properties.Property("requests with empty trx_id should return 400", prop.ForAll(
		func(token, mid, tid string) bool {
			mockAbly := &MockAblyPublisher{}
			mockMapper := &MockMIDTIDMapper{
				mappings: map[string]string{"M001:T001": "SN12345"},
			}
			transactionStore := store.NewSyncMapStore()
			handler := NewTransactionHandler(transactionStore, mockMapper, mockAbly, nil, 2*time.Second)

			// Test with empty string
			req := models.PaymentRequest{
				Token: token,
				MID:   mid,
				TID:   tid,
				TrxID: "",
			}

			body, _ := json.Marshal(req)
			httpReq := httptest.NewRequest("POST", "/api/v1/transaction", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.HandleTransaction(w, httpReq)

			return w.Code == http.StatusBadRequest
		},
		gen.AnyString(),
		gen.AnyString(),
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// Feature: payment-middleware, Property 5: Transaction Creation with Pending State
// Validates: Requirements 1.6, 6.2
func TestProperty_TransactionCreationWithPendingState(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)
	properties.Property("valid transactions should be stored with PENDING status", prop.ForAll(
		func(token, trxID string) bool {
			if strings.TrimSpace(trxID) == "" {
				return true // Skip empty trx_id
			}

			mockAbly := &MockAblyPublisher{
				publishDelay: 100 * time.Millisecond, // Small delay to check state
			}
			mockMapper := &MockMIDTIDMapper{
				mappings: map[string]string{"M001:T001": "SN12345"},
			}
			transactionStore := store.NewSyncMapStore()
			handler := NewTransactionHandler(transactionStore, mockMapper, mockAbly, nil, 2*time.Second)

			req := models.PaymentRequest{
				Token: token,
				MID:   "M001",
				TID:   "T001",
				TrxID: trxID,
			}

			body, _ := json.Marshal(req)
			httpReq := httptest.NewRequest("POST", "/api/v1/transaction", bytes.NewReader(body))

			// Start request in goroutine
			go func() {
				w := httptest.NewRecorder()
				handler.HandleTransaction(w, httpReq)
			}()

			// Give it a moment to store the transaction
			time.Sleep(50 * time.Millisecond)

			// Check if transaction exists with PENDING status
			tx, exists := transactionStore.Load(trxID)
			if !exists {
				return false
			}

			return tx.Status == models.StatusPending
		},
		gen.AnyString(),
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// Feature: payment-middleware, Property 17: Status Value Constraints
// Validates: Requirements 4.5
func TestProperty_StatusValueConstraints(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)
	properties.Property("transaction status should always be one of the valid values", prop.ForAll(
		func(trxID string) bool {
			if strings.TrimSpace(trxID) == "" {
				return true
			}

			transactionStore := store.NewSyncMapStore()

			// Test all valid status values
			validStatuses := []models.TransactionStatus{
				models.StatusPending,
				models.StatusSuccess,
				models.StatusFailed,
				models.StatusTimeout,
			}

			for _, status := range validStatuses {
				tx := &models.Transaction{
					TrxID:     trxID,
					Status:    status,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				transactionStore.Store(trxID, tx)

				loaded, exists := transactionStore.Load(trxID)
				if !exists {
					return false
				}

				// Verify status is one of the valid values
				isValid := loaded.Status == models.StatusPending ||
					loaded.Status == models.StatusSuccess ||
					loaded.Status == models.StatusFailed ||
					loaded.Status == models.StatusTimeout

				if !isValid {
					return false
				}
			}

			return true
		},
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// Feature: payment-middleware, Property 22: Error Response Format
// Validates: Requirements 8.1, 8.6
func TestProperty_ErrorResponseFormat(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)
	properties.Property("all error responses should have valid JSON with error field", prop.ForAll(
		func(statusCode int, message string) bool {
			// Only test error status codes
			if statusCode < 400 || statusCode >= 600 {
				return true
			}

			w := httptest.NewRecorder()
			WriteErrorResponse(w, statusCode, message)

			// Verify status code
			if w.Code != statusCode {
				return false
			}

			// Verify JSON format
			var errResp models.ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &errResp)
			if err != nil {
				return false
			}

			// Verify error field exists and matches
			return errResp.Error == message
		},
		gen.IntRange(400, 599),
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// Feature: payment-middleware, Property 21: Timestamp Update on State Change
// Validates: Requirements 6.6
func TestProperty_TimestampUpdateOnStateChange(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)
	properties.Property("updated_at should be >= created_at after state change", prop.ForAll(
		func(trxID string) bool {
			if strings.TrimSpace(trxID) == "" {
				return true
			}

			transactionStore := store.NewSyncMapStore()

			// Create initial transaction
			createdAt := time.Now()
			tx := &models.Transaction{
				TrxID:     trxID,
				Status:    models.StatusPending,
				CreatedAt: createdAt,
				UpdatedAt: createdAt,
			}

			transactionStore.Store(trxID, tx)

			// Wait a bit to ensure time difference
			time.Sleep(10 * time.Millisecond)

			// Update transaction state
			transactionStore.Update(trxID, func(t *models.Transaction) {
				t.Status = models.StatusSuccess
				t.UpdatedAt = time.Now()
			})

			// Load and verify
			updated, exists := transactionStore.Load(trxID)
			if !exists {
				return false
			}

			// UpdatedAt should be >= CreatedAt
			return !updated.UpdatedAt.Before(updated.CreatedAt)
		},
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// MockAblyPublisher for property tests
type MockAblyPublisher struct {
	publishError error
	publishDelay time.Duration
}

func (m *MockAblyPublisher) PublishPaymentRequest(serialNumber, token, trxID string) error {
	if m.publishDelay > 0 {
		time.Sleep(m.publishDelay)
	}
	return m.publishError
}

func (m *MockAblyPublisher) SubscribeToResponses(handler func(response models.EDCResponse)) error {
	return nil
}

// MockMIDTIDMapper for property tests
type MockMIDTIDMapper struct {
	mappings map[string]string
}

func (m *MockMIDTIDMapper) GetSerialNumber(mid, tid string) (string, error) {
	key := fmt.Sprintf("%s:%s", mid, tid)
	if serial, ok := m.mappings[key]; ok {
		return serial, nil
	}
	return "", fmt.Errorf("unknown mid/tid combination")
}
