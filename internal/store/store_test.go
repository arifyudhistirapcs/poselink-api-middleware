package store

import (
	"sync"
	"testing"
	"time"

	"payment-middleware/internal/models"
)

func TestSyncMapStore_Store(t *testing.T) {
	store := NewSyncMapStore()

	t.Run("store valid transaction", func(t *testing.T) {
		tx := &models.Transaction{
			TrxID:  "test-123",
			Status: models.StatusPending,
		}

		err := store.Store("test-123", tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Verify CreatedAt was set
		if tx.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be set")
		}

		// Verify UpdatedAt was set
		if tx.UpdatedAt.IsZero() {
			t.Error("expected UpdatedAt to be set")
		}
	})

	t.Run("store with empty trxID", func(t *testing.T) {
		tx := &models.Transaction{
			Status: models.StatusPending,
		}

		err := store.Store("", tx)
		if err == nil {
			t.Error("expected error for empty trxID")
		}
	})

	t.Run("store nil transaction", func(t *testing.T) {
		err := store.Store("test-456", nil)
		if err == nil {
			t.Error("expected error for nil transaction")
		}
	})

	t.Run("store preserves existing timestamps", func(t *testing.T) {
		createdAt := time.Now().Add(-1 * time.Hour)
		updatedAt := time.Now().Add(-30 * time.Minute)
		
		tx := &models.Transaction{
			TrxID:     "test-789",
			Status:    models.StatusPending,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}

		err := store.Store("test-789", tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Verify timestamps were preserved
		if !tx.CreatedAt.Equal(createdAt) {
			t.Error("expected CreatedAt to be preserved")
		}
		if !tx.UpdatedAt.Equal(updatedAt) {
			t.Error("expected UpdatedAt to be preserved")
		}
	})
}

func TestSyncMapStore_Load(t *testing.T) {
	store := NewSyncMapStore()

	t.Run("load existing transaction", func(t *testing.T) {
		tx := &models.Transaction{
			TrxID:  "test-load-1",
			Status: models.StatusPending,
		}
		store.Store("test-load-1", tx)

		loaded, ok := store.Load("test-load-1")
		if !ok {
			t.Error("expected transaction to be found")
		}
		if loaded.TrxID != "test-load-1" {
			t.Errorf("expected TrxID test-load-1, got %s", loaded.TrxID)
		}
		if loaded.Status != models.StatusPending {
			t.Errorf("expected status PENDING, got %s", loaded.Status)
		}
	})

	t.Run("load non-existent transaction", func(t *testing.T) {
		loaded, ok := store.Load("non-existent")
		if ok {
			t.Error("expected transaction not to be found")
		}
		if loaded != nil {
			t.Error("expected nil transaction")
		}
	})
}

func TestSyncMapStore_Update(t *testing.T) {
	store := NewSyncMapStore()

	t.Run("update existing transaction", func(t *testing.T) {
		tx := &models.Transaction{
			TrxID:  "test-update-1",
			Status: models.StatusPending,
		}
		store.Store("test-update-1", tx)

		// Wait a bit to ensure timestamp difference
		time.Sleep(10 * time.Millisecond)

		err := store.Update("test-update-1", func(t *models.Transaction) {
			t.Status = models.StatusSuccess
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Verify the update
		updated, ok := store.Load("test-update-1")
		if !ok {
			t.Error("expected transaction to be found")
		}
		if updated.Status != models.StatusSuccess {
			t.Errorf("expected status SUCCESS, got %s", updated.Status)
		}

		// Verify UpdatedAt was updated
		if !updated.UpdatedAt.After(updated.CreatedAt) {
			t.Error("expected UpdatedAt to be after CreatedAt")
		}
	})

	t.Run("update non-existent transaction", func(t *testing.T) {
		err := store.Update("non-existent", func(t *models.Transaction) {
			t.Status = models.StatusSuccess
		})

		if err == nil {
			t.Error("expected error for non-existent transaction")
		}
	})

	t.Run("update with empty trxID", func(t *testing.T) {
		err := store.Update("", func(t *models.Transaction) {
			t.Status = models.StatusSuccess
		})

		if err == nil {
			t.Error("expected error for empty trxID")
		}
	})

	t.Run("update with nil updateFn", func(t *testing.T) {
		tx := &models.Transaction{
			TrxID:  "test-update-2",
			Status: models.StatusPending,
		}
		store.Store("test-update-2", tx)

		err := store.Update("test-update-2", nil)
		if err == nil {
			t.Error("expected error for nil updateFn")
		}
	})
}

func TestSyncMapStore_ConcurrentAccess(t *testing.T) {
	store := NewSyncMapStore()

	t.Run("concurrent stores", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 20

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				tx := &models.Transaction{
					TrxID:  string(rune(id)),
					Status: models.StatusPending,
				}
				store.Store(string(rune(id)), tx)
			}(i)
		}

		wg.Wait()

		// Verify all transactions were stored
		count := 0
		store.data.Range(func(key, value interface{}) bool {
			count++
			return true
		})

		if count != numGoroutines {
			t.Errorf("expected %d transactions, got %d", numGoroutines, count)
		}
	})

	t.Run("concurrent updates on same transaction", func(t *testing.T) {
		tx := &models.Transaction{
			TrxID:  "concurrent-test",
			Status: models.StatusPending,
		}
		store.Store("concurrent-test", tx)

		var wg sync.WaitGroup
		numUpdates := 20

		for i := 0; i < numUpdates; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				store.Update("concurrent-test", func(t *models.Transaction) {
					// Simulate some work
					time.Sleep(1 * time.Millisecond)
				})
			}()
		}

		wg.Wait()

		// Verify transaction still exists and is valid
		updated, ok := store.Load("concurrent-test")
		if !ok {
			t.Error("expected transaction to exist after concurrent updates")
		}
		if updated.TrxID != "concurrent-test" {
			t.Error("transaction data corrupted")
		}
	})

	t.Run("concurrent reads and writes", func(t *testing.T) {
		// Store initial transactions
		for i := 0; i < 10; i++ {
			tx := &models.Transaction{
				TrxID:  string(rune('A' + i)),
				Status: models.StatusPending,
			}
			store.Store(string(rune('A'+i)), tx)
		}

		var wg sync.WaitGroup
		numOperations := 10

		// Concurrent reads
		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				trxID := string(rune('A' + (id % 10)))
				store.Load(trxID)
			}(i)
		}

		// Concurrent writes
		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				trxID := string(rune('A' + (id % 10)))
				store.Update(trxID, func(t *models.Transaction) {
					t.Status = models.StatusSuccess
				})
			}(i)
		}

		wg.Wait()

		// Verify all transactions still exist
		for i := 0; i < 10; i++ {
			trxID := string(rune('A' + i))
			_, ok := store.Load(trxID)
			if !ok {
				t.Errorf("expected transaction %s to exist", trxID)
			}
		}
	})
}

func TestSyncMapStore_TimestampBehavior(t *testing.T) {
	store := NewSyncMapStore()

	t.Run("CreatedAt set on Store", func(t *testing.T) {
		before := time.Now()
		tx := &models.Transaction{
			TrxID:  "timestamp-test-1",
			Status: models.StatusPending,
		}
		store.Store("timestamp-test-1", tx)
		after := time.Now()

		if tx.CreatedAt.Before(before) || tx.CreatedAt.After(after) {
			t.Error("CreatedAt timestamp not within expected range")
		}
	})

	t.Run("UpdatedAt updated on Update", func(t *testing.T) {
		tx := &models.Transaction{
			TrxID:  "timestamp-test-2",
			Status: models.StatusPending,
		}
		store.Store("timestamp-test-2", tx)

		originalUpdatedAt := tx.UpdatedAt

		// Wait to ensure timestamp difference
		time.Sleep(10 * time.Millisecond)

		store.Update("timestamp-test-2", func(t *models.Transaction) {
			t.Status = models.StatusSuccess
		})

		updated, _ := store.Load("timestamp-test-2")
		if !updated.UpdatedAt.After(originalUpdatedAt) {
			t.Error("UpdatedAt should be after original timestamp")
		}
	})

	t.Run("UpdatedAt greater than or equal to CreatedAt", func(t *testing.T) {
		tx := &models.Transaction{
			TrxID:  "timestamp-test-3",
			Status: models.StatusPending,
		}
		store.Store("timestamp-test-3", tx)

		store.Update("timestamp-test-3", func(t *models.Transaction) {
			t.Status = models.StatusSuccess
		})

		updated, _ := store.Load("timestamp-test-3")
		if updated.UpdatedAt.Before(updated.CreatedAt) {
			t.Error("UpdatedAt should be >= CreatedAt")
		}
	})
}
