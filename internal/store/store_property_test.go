package store

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"payment-middleware/internal/models"
)

// Feature: payment-middleware, Property 12: Concurrent Transaction Isolation
// Validates: Requirements 6.4, 10.1, 10.2, 10.4
func TestProperty_ConcurrentTransactionIsolation(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)
	properties.Property("concurrent operations should maintain data integrity", prop.ForAll(
		func(trxIDs []string) bool {
			if len(trxIDs) == 0 {
				return true
			}

			// Filter out empty IDs
			validIDs := make([]string, 0)
			for _, id := range trxIDs {
				if strings.TrimSpace(id) != "" {
					validIDs = append(validIDs, id)
				}
			}

			if len(validIDs) == 0 {
				return true
			}

			store := NewSyncMapStore()
			var wg sync.WaitGroup

			// Concurrently store transactions
			for _, trxID := range validIDs {
				wg.Add(1)
				go func(id string) {
					defer wg.Done()
					tx := &models.Transaction{
						TrxID:     id,
						Status:    models.StatusPending,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}
					store.Store(id, tx)
				}(trxID)
			}

			wg.Wait()

			// Verify all transactions were stored correctly
			for _, trxID := range validIDs {
				tx, exists := store.Load(trxID)
				if !exists {
					return false
				}
				if tx.TrxID != trxID {
					return false
				}
			}

			return true
		},
		gen.SliceOf(gen.Identifier()),
	))

	properties.TestingRun(t)
}

// Feature: payment-middleware, Property 20: Transaction Structure Completeness
// Validates: Requirements 6.3, 6.5
func TestProperty_TransactionStructureCompleteness(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)
	properties.Property("stored transactions should have all required fields", prop.ForAll(
		func(trxID string) bool {
			if strings.TrimSpace(trxID) == "" {
				return true
			}

			store := NewSyncMapStore()

			createdAt := time.Now()
			tx := &models.Transaction{
				TrxID:     trxID,
				Status:    models.StatusPending,
				CreatedAt: createdAt,
				UpdatedAt: createdAt,
				RequestData: models.PaymentRequest{
					Token: "test_token",
					MID:   "M001",
					TID:   "T001",
					TrxID: trxID,
				},
			}

			store.Store(trxID, tx)

			loaded, exists := store.Load(trxID)
			if !exists {
				return false
			}

			// Verify all required fields are present
			return loaded.TrxID == trxID &&
				loaded.Status != "" &&
				!loaded.CreatedAt.IsZero() &&
				!loaded.UpdatedAt.IsZero() &&
				loaded.RequestData.TrxID == trxID
		},
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// Feature: payment-middleware, Property 23: EDC Response Round-Trip Preservation
// Validates: Requirements 9.3
func TestProperty_EDCResponseRoundTripPreservation(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)
	properties.Property("EDC response data should be preserved exactly", prop.ForAll(
		func(trxID, amount, approval, status string) bool {
			if strings.TrimSpace(trxID) == "" {
				return true
			}

			store := NewSyncMapStore()

			edcResponse := &models.EDCResponse{
				TrxID:    trxID,
				Amount:   amount,
				Approval: approval,
				Status:   status,
			}

			tx := &models.Transaction{
				TrxID:        trxID,
				Status:       models.StatusSuccess,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
				ResponseData: edcResponse,
			}

			store.Store(trxID, tx)

			loaded, exists := store.Load(trxID)
			if !exists {
				return false
			}

			if loaded.ResponseData == nil {
				return false
			}

			// Verify all fields are preserved
			return loaded.ResponseData.TrxID == trxID &&
				loaded.ResponseData.Amount == amount &&
				loaded.ResponseData.Approval == approval &&
				loaded.ResponseData.Status == status
		},
		gen.Identifier(),
		gen.AnyString(),
		gen.AnyString(),
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// Feature: payment-middleware, Property 12 Extended: Concurrent Updates
// Validates: Requirements 6.4, 10.4
func TestProperty_ConcurrentUpdates(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)
	properties.Property("concurrent updates should be thread-safe", prop.ForAll(
		func(trxID string, updateCount int) bool {
			if strings.TrimSpace(trxID) == "" || updateCount <= 0 || updateCount > 100 {
				return true
			}

			store := NewSyncMapStore()

			// Store initial transaction
			tx := &models.Transaction{
				TrxID:     trxID,
				Status:    models.StatusPending,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			store.Store(trxID, tx)

			var wg sync.WaitGroup

			// Concurrently update the transaction
			for i := 0; i < updateCount; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					store.Update(trxID, func(t *models.Transaction) {
						t.UpdatedAt = time.Now()
					})
				}()
			}

			wg.Wait()

			// Verify transaction still exists and is valid
			loaded, exists := store.Load(trxID)
			if !exists {
				return false
			}

			return loaded.TrxID == trxID && !loaded.UpdatedAt.Before(loaded.CreatedAt)
		},
		gen.Identifier(),
		gen.IntRange(1, 50),
	))

	properties.TestingRun(t)
}
