package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Set required env var
	os.Setenv("ABLY_API_KEY", "test-api-key")
	defer os.Unsetenv("ABLY_API_KEY")

	// Clear optional env vars
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("TIMEOUT_DURATION")
	os.Unsetenv("MIDTID_MAPPINGS")

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.AblyAPIKey != "test-api-key" {
		t.Errorf("Expected AblyAPIKey 'test-api-key', got '%s'", config.AblyAPIKey)
	}

	if config.ServerPort != 8080 {
		t.Errorf("Expected default ServerPort 8080, got %d", config.ServerPort)
	}

	if config.TimeoutDuration != 60*time.Second {
		t.Errorf("Expected default TimeoutDuration 60s, got %v", config.TimeoutDuration)
	}

	if len(config.MIDTIDMappings) != 0 {
		t.Errorf("Expected empty MIDTIDMappings, got %v", config.MIDTIDMappings)
	}
}

func TestLoadConfig_MissingAblyAPIKey(t *testing.T) {
	os.Unsetenv("ABLY_API_KEY")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error when ABLY_API_KEY is missing")
	}

	expectedMsg := "ABLY_API_KEY environment variable is required"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestLoadConfig_CustomServerPort(t *testing.T) {
	os.Setenv("ABLY_API_KEY", "test-api-key")
	os.Setenv("SERVER_PORT", "9000")
	defer func() {
		os.Unsetenv("ABLY_API_KEY")
		os.Unsetenv("SERVER_PORT")
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.ServerPort != 9000 {
		t.Errorf("Expected ServerPort 9000, got %d", config.ServerPort)
	}
}

func TestLoadConfig_InvalidServerPort(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"non-numeric", "abc"},
		{"negative", "-1"},
		{"zero", "0"},
		{"too large", "70000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ABLY_API_KEY", "test-api-key")
			os.Setenv("SERVER_PORT", tt.value)
			defer func() {
				os.Unsetenv("ABLY_API_KEY")
				os.Unsetenv("SERVER_PORT")
			}()

			_, err := LoadConfig()
			if err == nil {
				t.Errorf("Expected error for SERVER_PORT='%s'", tt.value)
			}
		})
	}
}

func TestLoadConfig_CustomTimeoutDuration(t *testing.T) {
	os.Setenv("ABLY_API_KEY", "test-api-key")
	os.Setenv("TIMEOUT_DURATION", "120")
	defer func() {
		os.Unsetenv("ABLY_API_KEY")
		os.Unsetenv("TIMEOUT_DURATION")
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.TimeoutDuration != 120*time.Second {
		t.Errorf("Expected TimeoutDuration 120s, got %v", config.TimeoutDuration)
	}
}

func TestLoadConfig_InvalidTimeoutDuration(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"non-numeric", "xyz"},
		{"negative", "-10"},
		{"zero", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ABLY_API_KEY", "test-api-key")
			os.Setenv("TIMEOUT_DURATION", tt.value)
			defer func() {
				os.Unsetenv("ABLY_API_KEY")
				os.Unsetenv("TIMEOUT_DURATION")
			}()

			_, err := LoadConfig()
			if err == nil {
				t.Errorf("Expected error for TIMEOUT_DURATION='%s'", tt.value)
			}
		})
	}
}

func TestLoadConfig_MIDTIDMappings(t *testing.T) {
	os.Setenv("ABLY_API_KEY", "test-api-key")
	os.Setenv("MIDTID_MAPPINGS", `{"M001:T001":"SN12345","M002:T002":"SN67890"}`)
	defer func() {
		os.Unsetenv("ABLY_API_KEY")
		os.Unsetenv("MIDTID_MAPPINGS")
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if len(config.MIDTIDMappings) != 2 {
		t.Errorf("Expected 2 mappings, got %d", len(config.MIDTIDMappings))
	}

	if config.MIDTIDMappings["M001:T001"] != "SN12345" {
		t.Errorf("Expected mapping M001:T001 -> SN12345, got %s", config.MIDTIDMappings["M001:T001"])
	}

	if config.MIDTIDMappings["M002:T002"] != "SN67890" {
		t.Errorf("Expected mapping M002:T002 -> SN67890, got %s", config.MIDTIDMappings["M002:T002"])
	}
}

func TestLoadConfig_InvalidMIDTIDMappingsJSON(t *testing.T) {
	os.Setenv("ABLY_API_KEY", "test-api-key")
	os.Setenv("MIDTID_MAPPINGS", `{invalid json}`)
	defer func() {
		os.Unsetenv("ABLY_API_KEY")
		os.Unsetenv("MIDTID_MAPPINGS")
	}()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for invalid MIDTID_MAPPINGS JSON")
	}
}

func TestLoadConfig_EmptyMIDTIDMappings(t *testing.T) {
	os.Setenv("ABLY_API_KEY", "test-api-key")
	os.Setenv("MIDTID_MAPPINGS", `{}`)
	defer func() {
		os.Unsetenv("ABLY_API_KEY")
		os.Unsetenv("MIDTID_MAPPINGS")
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if len(config.MIDTIDMappings) != 0 {
		t.Errorf("Expected empty MIDTIDMappings, got %v", config.MIDTIDMappings)
	}
}

func TestLoadConfig_AllCustomValues(t *testing.T) {
	os.Setenv("ABLY_API_KEY", "custom-key-123")
	os.Setenv("SERVER_PORT", "3000")
	os.Setenv("TIMEOUT_DURATION", "45")
	os.Setenv("MIDTID_MAPPINGS", `{"M100:T100":"SN999"}`)
	defer func() {
		os.Unsetenv("ABLY_API_KEY")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("TIMEOUT_DURATION")
		os.Unsetenv("MIDTID_MAPPINGS")
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.AblyAPIKey != "custom-key-123" {
		t.Errorf("Expected AblyAPIKey 'custom-key-123', got '%s'", config.AblyAPIKey)
	}

	if config.ServerPort != 3000 {
		t.Errorf("Expected ServerPort 3000, got %d", config.ServerPort)
	}

	if config.TimeoutDuration != 45*time.Second {
		t.Errorf("Expected TimeoutDuration 45s, got %v", config.TimeoutDuration)
	}

	if config.MIDTIDMappings["M100:T100"] != "SN999" {
		t.Errorf("Expected mapping M100:T100 -> SN999, got %s", config.MIDTIDMappings["M100:T100"])
	}
}

func TestLoadConfig_RedisDefaults(t *testing.T) {
	os.Setenv("ABLY_API_KEY", "test-api-key")
	defer os.Unsetenv("ABLY_API_KEY")

	// Clear Redis env vars
	os.Unsetenv("REDIS_HOST")
	os.Unsetenv("REDIS_PORT")
	os.Unsetenv("REDIS_PASSWORD")
	os.Unsetenv("REDIS_DB")
	os.Unsetenv("REDIS_MIN_IDLE_CONNS")
	os.Unsetenv("REDIS_MAX_CONNS")

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.RedisHost != "localhost" {
		t.Errorf("Expected default RedisHost 'localhost', got '%s'", config.RedisHost)
	}

	if config.RedisPort != 6379 {
		t.Errorf("Expected default RedisPort 6379, got %d", config.RedisPort)
	}

	if config.RedisPassword != "" {
		t.Errorf("Expected default RedisPassword '', got '%s'", config.RedisPassword)
	}

	if config.RedisDB != 0 {
		t.Errorf("Expected default RedisDB 0, got %d", config.RedisDB)
	}

	if config.RedisMinIdleConns != 5 {
		t.Errorf("Expected default RedisMinIdleConns 5, got %d", config.RedisMinIdleConns)
	}

	if config.RedisMaxConns != 100 {
		t.Errorf("Expected default RedisMaxConns 100, got %d", config.RedisMaxConns)
	}
}

func TestLoadConfig_CustomRedisHost(t *testing.T) {
	os.Setenv("ABLY_API_KEY", "test-api-key")
	os.Setenv("REDIS_HOST", "redis.example.com")
	defer func() {
		os.Unsetenv("ABLY_API_KEY")
		os.Unsetenv("REDIS_HOST")
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.RedisHost != "redis.example.com" {
		t.Errorf("Expected RedisHost 'redis.example.com', got '%s'", config.RedisHost)
	}
}

func TestLoadConfig_CustomRedisPort(t *testing.T) {
	os.Setenv("ABLY_API_KEY", "test-api-key")
	os.Setenv("REDIS_PORT", "6380")
	defer func() {
		os.Unsetenv("ABLY_API_KEY")
		os.Unsetenv("REDIS_PORT")
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.RedisPort != 6380 {
		t.Errorf("Expected RedisPort 6380, got %d", config.RedisPort)
	}
}

func TestLoadConfig_InvalidRedisPort(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"non-numeric", "abc"},
		{"negative", "-1"},
		{"zero", "0"},
		{"too large", "70000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ABLY_API_KEY", "test-api-key")
			os.Setenv("REDIS_PORT", tt.value)
			defer func() {
				os.Unsetenv("ABLY_API_KEY")
				os.Unsetenv("REDIS_PORT")
			}()

			_, err := LoadConfig()
			if err == nil {
				t.Errorf("Expected error for REDIS_PORT='%s'", tt.value)
			}
		})
	}
}

func TestLoadConfig_CustomRedisPassword(t *testing.T) {
	os.Setenv("ABLY_API_KEY", "test-api-key")
	os.Setenv("REDIS_PASSWORD", "secret123")
	defer func() {
		os.Unsetenv("ABLY_API_KEY")
		os.Unsetenv("REDIS_PASSWORD")
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.RedisPassword != "secret123" {
		t.Errorf("Expected RedisPassword 'secret123', got '%s'", config.RedisPassword)
	}
}

func TestLoadConfig_CustomRedisDB(t *testing.T) {
	os.Setenv("ABLY_API_KEY", "test-api-key")
	os.Setenv("REDIS_DB", "2")
	defer func() {
		os.Unsetenv("ABLY_API_KEY")
		os.Unsetenv("REDIS_DB")
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.RedisDB != 2 {
		t.Errorf("Expected RedisDB 2, got %d", config.RedisDB)
	}
}

func TestLoadConfig_InvalidRedisDB(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"non-numeric", "xyz"},
		{"negative", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ABLY_API_KEY", "test-api-key")
			os.Setenv("REDIS_DB", tt.value)
			defer func() {
				os.Unsetenv("ABLY_API_KEY")
				os.Unsetenv("REDIS_DB")
			}()

			_, err := LoadConfig()
			if err == nil {
				t.Errorf("Expected error for REDIS_DB='%s'", tt.value)
			}
		})
	}
}

func TestLoadConfig_CustomRedisMinIdleConns(t *testing.T) {
	os.Setenv("ABLY_API_KEY", "test-api-key")
	os.Setenv("REDIS_MIN_IDLE_CONNS", "10")
	defer func() {
		os.Unsetenv("ABLY_API_KEY")
		os.Unsetenv("REDIS_MIN_IDLE_CONNS")
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.RedisMinIdleConns != 10 {
		t.Errorf("Expected RedisMinIdleConns 10, got %d", config.RedisMinIdleConns)
	}
}

func TestLoadConfig_InvalidRedisMinIdleConns(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"non-numeric", "abc"},
		{"negative", "-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ABLY_API_KEY", "test-api-key")
			os.Setenv("REDIS_MIN_IDLE_CONNS", tt.value)
			defer func() {
				os.Unsetenv("ABLY_API_KEY")
				os.Unsetenv("REDIS_MIN_IDLE_CONNS")
			}()

			_, err := LoadConfig()
			if err == nil {
				t.Errorf("Expected error for REDIS_MIN_IDLE_CONNS='%s'", tt.value)
			}
		})
	}
}

func TestLoadConfig_CustomRedisMaxConns(t *testing.T) {
	os.Setenv("ABLY_API_KEY", "test-api-key")
	os.Setenv("REDIS_MAX_CONNS", "200")
	defer func() {
		os.Unsetenv("ABLY_API_KEY")
		os.Unsetenv("REDIS_MAX_CONNS")
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.RedisMaxConns != 200 {
		t.Errorf("Expected RedisMaxConns 200, got %d", config.RedisMaxConns)
	}
}

func TestLoadConfig_InvalidRedisMaxConns(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"non-numeric", "xyz"},
		{"negative", "-10"},
		{"zero", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ABLY_API_KEY", "test-api-key")
			os.Setenv("REDIS_MAX_CONNS", tt.value)
			defer func() {
				os.Unsetenv("ABLY_API_KEY")
				os.Unsetenv("REDIS_MAX_CONNS")
			}()

			_, err := LoadConfig()
			if err == nil {
				t.Errorf("Expected error for REDIS_MAX_CONNS='%s'", tt.value)
			}
		})
	}
}

func TestLoadConfig_AllRedisCustomValues(t *testing.T) {
	os.Setenv("ABLY_API_KEY", "test-api-key")
	os.Setenv("REDIS_HOST", "redis-prod.example.com")
	os.Setenv("REDIS_PORT", "6380")
	os.Setenv("REDIS_PASSWORD", "prod-secret")
	os.Setenv("REDIS_DB", "3")
	os.Setenv("REDIS_MIN_IDLE_CONNS", "15")
	os.Setenv("REDIS_MAX_CONNS", "250")
	defer func() {
		os.Unsetenv("ABLY_API_KEY")
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_PORT")
		os.Unsetenv("REDIS_PASSWORD")
		os.Unsetenv("REDIS_DB")
		os.Unsetenv("REDIS_MIN_IDLE_CONNS")
		os.Unsetenv("REDIS_MAX_CONNS")
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.RedisHost != "redis-prod.example.com" {
		t.Errorf("Expected RedisHost 'redis-prod.example.com', got '%s'", config.RedisHost)
	}

	if config.RedisPort != 6380 {
		t.Errorf("Expected RedisPort 6380, got %d", config.RedisPort)
	}

	if config.RedisPassword != "prod-secret" {
		t.Errorf("Expected RedisPassword 'prod-secret', got '%s'", config.RedisPassword)
	}

	if config.RedisDB != 3 {
		t.Errorf("Expected RedisDB 3, got %d", config.RedisDB)
	}

	if config.RedisMinIdleConns != 15 {
		t.Errorf("Expected RedisMinIdleConns 15, got %d", config.RedisMinIdleConns)
	}

	if config.RedisMaxConns != 250 {
		t.Errorf("Expected RedisMaxConns 250, got %d", config.RedisMaxConns)
	}
}
