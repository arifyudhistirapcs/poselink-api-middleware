package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	AblyAPIKey       string
	EncryptionSecret string
	ServerPort       int
	TimeoutDuration  time.Duration
	MIDTIDMappings   map[string]string // key: "mid:tid", value: serial_number
	
	// Redis configuration
	RedisHost         string
	RedisPort         int
	RedisPassword     string
	RedisDB           int
	RedisMinIdleConns int
	RedisMaxConns     int
}

// LoadConfig loads configuration from environment variables or uses defaults
// Environment variables:
//   - ABLY_API_KEY: Ably API key (required)
//   - ENCRYPTION_SECRET: AES encryption secret (default: "ECR2022secretKey")
//   - SERVER_PORT: HTTP server port (default: 8080)
//   - TIMEOUT_DURATION: Transaction timeout in seconds (default: 60)
//   - MIDTID_MAPPINGS: JSON string of MID/TID mappings (default: empty map)
//   - REDIS_HOST: Redis server host (default: "localhost")
//   - REDIS_PORT: Redis server port (default: 6379)
//   - REDIS_PASSWORD: Redis password (default: "")
//   - REDIS_DB: Redis database number (default: 0)
//   - REDIS_MIN_IDLE_CONNS: Minimum idle connections (default: 5)
//   - REDIS_MAX_CONNS: Maximum connections (default: 100)
func LoadConfig() (*Config, error) {
	config := &Config{
		EncryptionSecret:  "ECR2022secretKey",
		ServerPort:        8080,
		TimeoutDuration:   60 * time.Second,
		MIDTIDMappings:    make(map[string]string),
		RedisHost:         "localhost",
		RedisPort:         6379,
		RedisPassword:     "",
		RedisDB:           0,
		RedisMinIdleConns: 5,
		RedisMaxConns:     100,
	}

	// Load Ably API Key (required)
	config.AblyAPIKey = os.Getenv("ABLY_API_KEY")
	if config.AblyAPIKey == "" {
		return nil, fmt.Errorf("ABLY_API_KEY environment variable is required")
	}

	// Load Encryption Secret (optional, default: "ECR2022secretKey")
	if encryptionSecret := os.Getenv("ENCRYPTION_SECRET"); encryptionSecret != "" {
		config.EncryptionSecret = encryptionSecret
	}

	// Load Server Port (optional, default: 8080)
	if portStr := os.Getenv("SERVER_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid SERVER_PORT: %w", err)
		}
		if port <= 0 || port > 65535 {
			return nil, fmt.Errorf("SERVER_PORT must be between 1 and 65535, got %d", port)
		}
		config.ServerPort = port
	}

	// Load Timeout Duration (optional, default: 60s)
	if timeoutStr := os.Getenv("TIMEOUT_DURATION"); timeoutStr != "" {
		timeout, err := strconv.Atoi(timeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid TIMEOUT_DURATION: %w", err)
		}
		if timeout <= 0 {
			return nil, fmt.Errorf("TIMEOUT_DURATION must be positive, got %d", timeout)
		}
		config.TimeoutDuration = time.Duration(timeout) * time.Second
	}

	// Load MID/TID Mappings (optional, default: empty map)
	if mappingsJSON := os.Getenv("MIDTID_MAPPINGS"); mappingsJSON != "" {
		if err := json.Unmarshal([]byte(mappingsJSON), &config.MIDTIDMappings); err != nil {
			return nil, fmt.Errorf("invalid MIDTID_MAPPINGS JSON: %w", err)
		}
	}

	// Load Redis Host (optional, default: "localhost")
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		config.RedisHost = redisHost
	}

	// Load Redis Port (optional, default: 6379)
	if redisPortStr := os.Getenv("REDIS_PORT"); redisPortStr != "" {
		redisPort, err := strconv.Atoi(redisPortStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_PORT: %w", err)
		}
		if redisPort <= 0 || redisPort > 65535 {
			return nil, fmt.Errorf("REDIS_PORT must be between 1 and 65535, got %d", redisPort)
		}
		config.RedisPort = redisPort
	}

	// Load Redis Password (optional, default: "")
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		config.RedisPassword = redisPassword
	}

	// Load Redis DB (optional, default: 0)
	if redisDBStr := os.Getenv("REDIS_DB"); redisDBStr != "" {
		redisDB, err := strconv.Atoi(redisDBStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
		}
		if redisDB < 0 {
			return nil, fmt.Errorf("REDIS_DB must be non-negative, got %d", redisDB)
		}
		config.RedisDB = redisDB
	}

	// Load Redis Min Idle Conns (optional, default: 5)
	if redisMinIdleConnsStr := os.Getenv("REDIS_MIN_IDLE_CONNS"); redisMinIdleConnsStr != "" {
		redisMinIdleConns, err := strconv.Atoi(redisMinIdleConnsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_MIN_IDLE_CONNS: %w", err)
		}
		if redisMinIdleConns < 0 {
			return nil, fmt.Errorf("REDIS_MIN_IDLE_CONNS must be non-negative, got %d", redisMinIdleConns)
		}
		config.RedisMinIdleConns = redisMinIdleConns
	}

	// Load Redis Max Conns (optional, default: 100)
	if redisMaxConnsStr := os.Getenv("REDIS_MAX_CONNS"); redisMaxConnsStr != "" {
		redisMaxConns, err := strconv.Atoi(redisMaxConnsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_MAX_CONNS: %w", err)
		}
		if redisMaxConns <= 0 {
			return nil, fmt.Errorf("REDIS_MAX_CONNS must be positive, got %d", redisMaxConns)
		}
		config.RedisMaxConns = redisMaxConns
	}

	return config, nil
}
