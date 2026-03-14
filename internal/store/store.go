package store

import (
	"fmt"
	"sync"
	"time"

	"payment-middleware/internal/models"
)

// TransactionStore defines the interface for storing and retrieving transactions
type TransactionStore interface {
	Store(trxID string, tx *models.Transaction) error
	Load(trxID string) (*models.Transaction, bool)
	Update(trxID string, updateFn func(*models.Transaction)) error
}

// SyncMapStore implements TransactionStore using sync.Map for thread-safe operations
type SyncMapStore struct {
	data    sync.Map
	mutexes sync.Map // Map of trxID -> *sync.Mutex for per-transaction locking
}

// NewSyncMapStore creates a new SyncMapStore instance
func NewSyncMapStore() *SyncMapStore {
	return &SyncMapStore{}
}

// Store saves a transaction with the given trxID as the key
// Sets CreatedAt timestamp if not already set
func (s *SyncMapStore) Store(trxID string, tx *models.Transaction) error {
	if trxID == "" {
		return fmt.Errorf("trxID cannot be empty")
	}
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	// Set CreatedAt timestamp if not already set
	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = time.Now()
	}
	
	// Set UpdatedAt to match CreatedAt on initial store
	if tx.UpdatedAt.IsZero() {
		tx.UpdatedAt = tx.CreatedAt
	}

	s.data.Store(trxID, tx)
	return nil
}

// Load retrieves a transaction by trxID
// Returns the transaction and true if found, nil and false otherwise
func (s *SyncMapStore) Load(trxID string) (*models.Transaction, bool) {
	value, ok := s.data.Load(trxID)
	if !ok {
		return nil, false
	}
	
	tx, ok := value.(*models.Transaction)
	if !ok {
		return nil, false
	}
	
	return tx, true
}

// Update performs a thread-safe update on a transaction
// The updateFn is called with the current transaction state and can modify it
// Updates the UpdatedAt timestamp after the update function executes
func (s *SyncMapStore) Update(trxID string, updateFn func(*models.Transaction)) error {
	if trxID == "" {
		return fmt.Errorf("trxID cannot be empty")
	}
	if updateFn == nil {
		return fmt.Errorf("updateFn cannot be nil")
	}

	// Get or create a mutex for this transaction
	mutexInterface, _ := s.mutexes.LoadOrStore(trxID, &sync.Mutex{})
	mutex := mutexInterface.(*sync.Mutex)

	// Lock for this specific transaction
	mutex.Lock()
	defer mutex.Unlock()

	// Load the transaction
	tx, ok := s.Load(trxID)
	if !ok {
		return fmt.Errorf("transaction not found: %s", trxID)
	}

	// Apply the update function
	updateFn(tx)

	// Update the timestamp
	tx.UpdatedAt = time.Now()

	// Store the updated transaction
	s.data.Store(trxID, tx)

	return nil
}
