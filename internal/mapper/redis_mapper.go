package mapper

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisMIDTIDMapper implements MIDTIDMapper using Redis as the storage backend
type RedisMIDTIDMapper struct {
	client *redis.Client
}

// NewRedisMIDTIDMapper creates a new RedisMIDTIDMapper with the provided Redis client
func NewRedisMIDTIDMapper(client *redis.Client) *RedisMIDTIDMapper {
	return &RedisMIDTIDMapper{
		client: client,
	}
}

// GetSerialNumber retrieves the serial number for a given MID and TID combination from Redis
// Returns an error if the mapping is not found
func (m *RedisMIDTIDMapper) GetSerialNumber(mid, tid string) (string, error) {
	key := fmt.Sprintf("mapping:mid:%s:tid:%s", mid, tid)
	
	ctx := context.Background()
	serialNumber, err := m.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("unknown mid/tid combination: %s:%s", mid, tid)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get mapping from Redis: %w", err)
	}
	
	return serialNumber, nil
}

// SetMapping stores a MID/TID to serial number mapping in Redis without TTL (persistent)
func (m *RedisMIDTIDMapper) SetMapping(mid, tid, serialNumber string) error {
	key := fmt.Sprintf("mapping:mid:%s:tid:%s", mid, tid)
	
	ctx := context.Background()
	err := m.client.Set(ctx, key, serialNumber, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set mapping in Redis: %w", err)
	}
	
	return nil
}

// DeleteMapping removes a MID/TID mapping from Redis
func (m *RedisMIDTIDMapper) DeleteMapping(mid, tid string) error {
	key := fmt.Sprintf("mapping:mid:%s:tid:%s", mid, tid)
	
	ctx := context.Background()
	err := m.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete mapping from Redis: %w", err)
	}
	
	return nil
}
