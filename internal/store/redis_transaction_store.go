package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"payment-middleware/internal/models"
)

// RedisTransactionStore implements TransactionStore interface with Redis backend
type RedisTransactionStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisTransactionStore creates a new RedisTransactionStore with 24-hour TTL
func NewRedisTransactionStore(client *redis.Client) *RedisTransactionStore {
	return &RedisTransactionStore{
		client: client,
		ttl:    24 * time.Hour, // 86400 seconds
	}
}

// transactionKey generates the Redis key for a transaction
// Pattern: "transaction:{trx_id}"
func (s *RedisTransactionStore) transactionKey(trxID string) string {
	return fmt.Sprintf("transaction:%s", trxID)
}

// Store saves a transaction to Redis with 24-hour TTL
func (s *RedisTransactionStore) Store(trxID string, tx *models.Transaction) error {
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

	// Serialize transaction to JSON
	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to serialize transaction %s: %w", trxID, err)
	}

	// Store in Redis with TTL
	ctx := context.Background()
	key := s.transactionKey(trxID)
	err = s.client.Set(ctx, key, data, s.ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to store transaction %s in Redis: %w", trxID, err)
	}

	return nil
}

// Load retrieves a transaction from Redis
func (s *RedisTransactionStore) Load(trxID string) (*models.Transaction, bool) {
	ctx := context.Background()
	key := s.transactionKey(trxID)

	// Get from Redis
	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// Key doesn't exist or has expired
		return nil, false
	}
	if err != nil {
		// Other Redis error
		return nil, false
	}

	// Deserialize from JSON
	var tx models.Transaction
	err = json.Unmarshal([]byte(data), &tx)
	if err != nil {
		// Invalid JSON data
		return nil, false
	}

	return &tx, true
}

// Update performs a thread-safe update on a transaction using optimistic locking
func (s *RedisTransactionStore) Update(trxID string, updateFn func(*models.Transaction)) error {
	if trxID == "" {
		return fmt.Errorf("trxID cannot be empty")
	}
	if updateFn == nil {
		return fmt.Errorf("updateFn cannot be nil")
	}

	ctx := context.Background()
	key := s.transactionKey(trxID)
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Watch the key for concurrent modifications
		err := s.client.Watch(ctx, func(tx *redis.Tx) error {
			// Load current transaction
			data, err := tx.Get(ctx, key).Result()
			if err == redis.Nil {
				return fmt.Errorf("transaction not found: %s", trxID)
			}
			if err != nil {
				return fmt.Errorf("failed to load transaction %s: %w", trxID, err)
			}

			// Deserialize
			var transaction models.Transaction
			err = json.Unmarshal([]byte(data), &transaction)
			if err != nil {
				return fmt.Errorf("failed to deserialize transaction %s: %w", trxID, err)
			}

			// Apply update function
			updateFn(&transaction)

			// Update timestamp
			transaction.UpdatedAt = time.Now()

			// Serialize updated transaction
			updatedData, err := json.Marshal(&transaction)
			if err != nil {
				return fmt.Errorf("failed to serialize updated transaction %s: %w", trxID, err)
			}

			// Execute transaction with MULTI/EXEC
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, key, updatedData, s.ttl)
				return nil
			})
			return err
		}, key)

		if err == nil {
			// Success
			return nil
		}

		// Check if it's a transaction conflict (optimistic lock failure)
		if err == redis.TxFailedErr {
			// Retry on conflict
			continue
		}

		// Other error, don't retry
		return err
	}

	return fmt.Errorf("failed to update transaction after %d retries", maxRetries)
}
