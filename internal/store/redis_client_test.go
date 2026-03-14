package store

import (
	"context"
	"testing"
	"time"
)

func TestNewRedisClient_Success(t *testing.T) {
	// This test requires a running Redis instance
	// Skip if Redis is not available
	config := RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		MinIdleConns: 5,
		MaxConns:     100,
	}

	client, err := NewRedisClient(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer client.Close()

	// Verify client is not nil
	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	// Verify underlying client is accessible
	if client.Client() == nil {
		t.Fatal("Expected non-nil underlying redis.Client")
	}
}

func TestRedisClient_Ping(t *testing.T) {
	config := RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		MinIdleConns: 5,
		MaxConns:     100,
	}

	client, err := NewRedisClient(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = client.Ping(ctx)
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestRedisClient_Close(t *testing.T) {
	config := RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		MinIdleConns: 5,
		MaxConns:     100,
	}

	client, err := NewRedisClient(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify that operations fail after close
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = client.Ping(ctx)
	if err == nil {
		t.Error("Expected Ping to fail after Close, but it succeeded")
	}
}

func TestNewRedisClient_InvalidHost(t *testing.T) {
	config := RedisConfig{
		Host:         "invalid-host-that-does-not-exist",
		Port:         6379,
		Password:     "",
		DB:           0,
		MinIdleConns: 5,
		MaxConns:     100,
	}

	// This should fail after retries
	client, err := NewRedisClient(config)
	if err == nil {
		client.Close()
		t.Fatal("Expected error when connecting to invalid host")
	}

	if client != nil {
		t.Error("Expected nil client on connection failure")
	}
}

func TestNewRedisClient_ConnectionPoolConfig(t *testing.T) {
	config := RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		MinIdleConns: 10,
		MaxConns:     50,
	}

	client, err := NewRedisClient(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer client.Close()

	// Verify config is stored
	if client.config.MinIdleConns != 10 {
		t.Errorf("Expected MinIdleConns=10, got %d", client.config.MinIdleConns)
	}
	if client.config.MaxConns != 50 {
		t.Errorf("Expected MaxConns=50, got %d", client.config.MaxConns)
	}
}
