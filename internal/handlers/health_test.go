package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"payment-middleware/internal/store"

	"github.com/alicebob/miniredis/v2"
)

func TestHealthHandler_CheckHealth_Success(t *testing.T) {
	// Setup miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	// Create Redis client
	config := store.RedisConfig{
		Host:         mr.Host(),
		Port:         mr.Server().Addr().Port,
		Password:     "",
		DB:           0,
		MinIdleConns: 1,
		MaxConns:     10,
	}
	redisClient, err := store.NewRedisClient(config)
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redisClient.Close()

	// Create health handler
	handler := NewHealthHandler(redisClient)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.CheckHealth(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Parse response body
	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response["status"])
	}
}

func TestHealthHandler_CheckHealth_RedisUnavailable(t *testing.T) {
	// Setup miniredis
	mr := miniredis.RunT(t)
	config := store.RedisConfig{
		Host:         mr.Host(),
		Port:         mr.Server().Addr().Port,
		Password:     "",
		DB:           0,
		MinIdleConns: 1,
		MaxConns:     10,
	}
	redisClient, err := store.NewRedisClient(config)
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redisClient.Close()
	
	// Close miniredis to simulate connection loss
	mr.Close()

	// Create health handler
	handler := NewHealthHandler(redisClient)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.CheckHealth(w, req)

	// Verify response
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	// Parse response body
	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", response["status"])
	}

	if response["error"] != "Redis unavailable" {
		t.Errorf("Expected error 'Redis unavailable', got '%s'", response["error"])
	}
}

func TestHealthHandler_CheckHealth_PingTimeout(t *testing.T) {
	// Setup miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	// Create Redis client
	config := store.RedisConfig{
		Host:         mr.Host(),
		Port:         mr.Server().Addr().Port,
		Password:     "",
		DB:           0,
		MinIdleConns: 1,
		MaxConns:     10,
	}
	redisClient, err := store.NewRedisClient(config)
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redisClient.Close()

	// Verify Ping works before closing
	ctx := context.Background()
	if err := redisClient.Ping(ctx); err != nil {
		t.Fatalf("Initial ping failed: %v", err)
	}

	// Close miniredis to simulate timeout
	mr.Close()

	// Create health handler
	handler := NewHealthHandler(redisClient)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.CheckHealth(w, req)

	// Verify response - should be 503 since Redis is unavailable
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	// Parse response body
	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", response["status"])
	}
}
