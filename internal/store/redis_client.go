package store

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	MinIdleConns int
	MaxConns     int
}

// RedisClient wraps the go-redis client with connection management
type RedisClient struct {
	client *redis.Client
	config RedisConfig
}

// NewRedisClient creates a new RedisClient with connection retry logic
// Retries up to 5 times with exponential backoff: 1s, 2s, 4s, 8s, 16s
func NewRedisClient(config RedisConfig) (*RedisClient, error) {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	
	options := &redis.Options{
		Addr:           addr,
		Password:       config.Password,
		DB:             config.DB,
		MinIdleConns:   config.MinIdleConns,
		MaxActiveConns: config.MaxConns,
	}

	client := redis.NewClient(options)
	
	// Retry connection with exponential backoff
	maxRetries := 5
	backoffDurations := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
	}
	
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := client.Ping(ctx).Err()
		cancel()
		
		if err == nil {
			log.Printf("Successfully connected to Redis at %s", addr)
			return &RedisClient{
				client: client,
				config: config,
			}, nil
		}
		
		lastErr = err
		log.Printf("Redis connection attempt %d/%d failed: %v", attempt+1, maxRetries, err)
		
		if attempt < maxRetries-1 {
			backoff := backoffDurations[attempt]
			log.Printf("Retrying in %v...", backoff)
			time.Sleep(backoff)
		}
	}
	
	client.Close()
	return nil, fmt.Errorf("failed to connect to Redis after %d attempts: %w", maxRetries, lastErr)
}

// Ping checks Redis connectivity using PING command
func (rc *RedisClient) Ping(ctx context.Context) error {
	return rc.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (rc *RedisClient) Close() error {
	return rc.client.Close()
}

// Client returns the underlying redis.Client for direct operations
func (rc *RedisClient) Client() *redis.Client {
	return rc.client
}
