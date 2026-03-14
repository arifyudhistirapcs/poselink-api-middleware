package store

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"payment-middleware/internal/models"
)

func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, mr
}

func TestNewRedisTransactionStore(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	store := NewRedisTransactionStore(client)

	if store == nil {
		t.Fatal("Expected non-nil store")
	}

	if store.client != client {
		t.Error("Expected client to be set")
	}

	expectedTTL := 24 * time.Hour
	if store.ttl != expectedTTL {
		t.Errorf("Expected TTL to be %v, got %v", expectedTTL, store.ttl)
	}
}

func TestRedisTransactionStore_TransactionKey(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	store := NewRedisTransactionStore(client)

	tests := []struct {
		name     string
		trxID    string
		expected string
	}{
		{
			name:     "simple transaction ID",
			trxID:    "TRX123",
			expected: "transaction:TRX123",
		},
		{
			name:     "transaction ID with special characters",
			trxID:    "TRX-2024-001",
			expected: "transaction:TRX-2024-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := store.transactionKey(tt.trxID)
			if key != tt.expected {
				t.Errorf("Expected key %s, got %s", tt.expected, key)
			}
		})
	}
}

func TestRedisTransactionStore_Store(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	store := NewRedisTransactionStore(client)

	t.Run("store valid transaction", func(t *testing.T) {
		tx := &models.Transaction{
			TrxID:  "TRX123",
			Status: models.StatusPending,
			RequestData: models.PaymentRequest{
				TrxID: "TRX123",
				MID:   "M001",
				TID:   "T001",
				Token: "token123",
			},
		}

		err := store.Store("TRX123", tx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify timestamps were set
		if tx.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
		if tx.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}
	})

	t.Run("store with empty trxID", func(t *testing.T) {
		tx := &models.Transaction{
			TrxID:  "TRX456",
			Status: models.StatusPending,
		}

		err := store.Store("", tx)
		if err == nil {
			t.Fatal("Expected error for empty trxID")
		}
		if err.Error() != "trxID cannot be empty" {
			t.Errorf("Expected 'trxID cannot be empty' error, got %v", err)
		}
	})

	t.Run("store with nil transaction", func(t *testing.T) {
		err := store.Store("TRX789", nil)
		if err == nil {
			t.Fatal("Expected error for nil transaction")
		}
		if err.Error() != "transaction cannot be nil" {
			t.Errorf("Expected 'transaction cannot be nil' error, got %v", err)
		}
	})
}

func TestRedisTransactionStore_Load(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	store := NewRedisTransactionStore(client)

	t.Run("load existing transaction", func(t *testing.T) {
		// Store a transaction first
		tx := &models.Transaction{
			TrxID:  "TRX123",
			Status: models.StatusPending,
			RequestData: models.PaymentRequest{
				TrxID: "TRX123",
				MID:   "M001",
				TID:   "T001",
				Token: "token123",
			},
		}
		err := store.Store("TRX123", tx)
		if err != nil {
			t.Fatalf("Failed to store transaction: %v", err)
		}

		// Load it back
		loaded, ok := store.Load("TRX123")
		if !ok {
			t.Fatal("Expected transaction to be found")
		}
		if loaded == nil {
			t.Fatal("Expected non-nil transaction")
		}
		if loaded.TrxID != "TRX123" {
			t.Errorf("Expected TrxID TRX123, got %s", loaded.TrxID)
		}
		if loaded.Status != models.StatusPending {
			t.Errorf("Expected status PENDING, got %s", loaded.Status)
		}
	})

	t.Run("load non-existent transaction", func(t *testing.T) {
		loaded, ok := store.Load("NONEXISTENT")
		if ok {
			t.Error("Expected transaction not to be found")
		}
		if loaded != nil {
			t.Error("Expected nil transaction")
		}
	})
}

func TestRedisTransactionStore_Update(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	store := NewRedisTransactionStore(client)

	t.Run("update existing transaction", func(t *testing.T) {
		// Store a transaction first
		tx := &models.Transaction{
			TrxID:  "TRX123",
			Status: models.StatusPending,
			RequestData: models.PaymentRequest{
				TrxID: "TRX123",
				MID:   "M001",
				TID:   "T001",
				Token: "token123",
			},
		}
		err := store.Store("TRX123", tx)
		if err != nil {
			t.Fatalf("Failed to store transaction: %v", err)
		}

		originalUpdatedAt := tx.UpdatedAt
		time.Sleep(10 * time.Millisecond) // Ensure time difference

		// Update the transaction
		err = store.Update("TRX123", func(t *models.Transaction) {
			t.Status = models.StatusSuccess
		})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Load and verify
		loaded, ok := store.Load("TRX123")
		if !ok {
			t.Fatal("Expected transaction to be found")
		}
		if loaded.Status != models.StatusSuccess {
			t.Errorf("Expected status SUCCESS, got %s", loaded.Status)
		}
		if !loaded.UpdatedAt.After(originalUpdatedAt) {
			t.Error("Expected UpdatedAt to be updated")
		}
	})

	t.Run("update non-existent transaction", func(t *testing.T) {
		err := store.Update("NONEXISTENT", func(t *models.Transaction) {
			t.Status = models.StatusSuccess
		})
		if err == nil {
			t.Fatal("Expected error for non-existent transaction")
		}
	})

	t.Run("update with empty trxID", func(t *testing.T) {
		err := store.Update("", func(t *models.Transaction) {
			t.Status = models.StatusSuccess
		})
		if err == nil {
			t.Fatal("Expected error for empty trxID")
		}
		if err.Error() != "trxID cannot be empty" {
			t.Errorf("Expected 'trxID cannot be empty' error, got %v", err)
		}
	})

	t.Run("update with nil updateFn", func(t *testing.T) {
		err := store.Update("TRX123", nil)
		if err == nil {
			t.Fatal("Expected error for nil updateFn")
		}
		if err.Error() != "updateFn cannot be nil" {
			t.Errorf("Expected 'updateFn cannot be nil' error, got %v", err)
		}
	})
}
