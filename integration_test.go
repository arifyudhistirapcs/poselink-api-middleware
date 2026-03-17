package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"payment-middleware/internal/ably"
	"payment-middleware/internal/config"
	"payment-middleware/internal/handlers"
	"payment-middleware/internal/mapper"
	"payment-middleware/internal/models"
	"payment-middleware/internal/store"
)

// TestFullIntegration tests the complete wiring of all components
func TestFullIntegration(t *testing.T) {
	// Set up test configuration
	os.Setenv("ABLY_API_KEY", "test_api_key")
	os.Setenv("SERVER_PORT", "9999")
	os.Setenv("TIMEOUT_DURATION", "5")
	os.Setenv("MIDTID_MAPPINGS", `{"M001:T001":"SN12345"}`)
	defer func() {
		os.Unsetenv("ABLY_API_KEY")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("TIMEOUT_DURATION")
		os.Unsetenv("MIDTID_MAPPINGS")
	}()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Verify configuration
	if cfg.ServerPort != 9999 {
		t.Errorf("Expected ServerPort 9999, got %d", cfg.ServerPort)
	}
	if cfg.TimeoutDuration != 5*time.Second {
		t.Errorf("Expected TimeoutDuration 5s, got %v", cfg.TimeoutDuration)
	}
	if len(cfg.MIDTIDMappings) != 1 {
		t.Errorf("Expected 1 MID/TID mapping, got %d", len(cfg.MIDTIDMappings))
	}

	// Initialize components
	midtidMapper := mapper.NewInMemoryMapper(cfg.MIDTIDMappings)
	transactionStore := store.NewSyncMapStore()

	// Verify mapper works
	serialNumber, err := midtidMapper.GetSerialNumber("M001", "T001")
	if err != nil {
		t.Fatalf("Failed to get serial number: %v", err)
	}
	if serialNumber != "SN12345" {
		t.Errorf("Expected serial number SN12345, got %s", serialNumber)
	}

	// Create mock Ably client
	mockAbly := &mockAblyPublisher{
		publishFunc: func(serialNumber, token, trxID string) error {
			return nil
		},
		subscribeFunc: func(handler func(response models.EDCResponse)) error {
			return nil
		},
	}

	// Initialize handlers
	transactionHandler := handlers.NewTransactionHandler(transactionStore, midtidMapper, mockAbly, nil)
	edcHandler := handlers.NewEDCResponseHandler(transactionStore)

	// Create a simple health check handler for testing (no Redis dependency)
	simpleHealthCheck := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}

	// Set up router with middleware
	router := mux.NewRouter()
	router.Use(handlers.PanicRecoveryMiddleware)
	router.HandleFunc("/api/v1/transaction", transactionHandler.HandleTransaction).Methods("POST")
	router.HandleFunc("/api/v1/transaction/status/{trx_id}", transactionHandler.HandleTransactionStatus).Methods("GET")
	router.HandleFunc("/health", simpleHealthCheck).Methods("GET")

	// Test health check
	t.Run("HealthCheck", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	// Test transaction initiation with timeout
	t.Run("TransactionTimeout", func(t *testing.T) {
		// Skip this test in short mode as it takes 5+ seconds
		if testing.Short() {
			t.Skip("Skipping timeout test in short mode")
		}

		reqBody := models.PaymentRequest{
			Token: "test_token",
			MID:   "M001",
			TID:   "T001",
			TrxID: "TEST-TRX-001",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/transaction", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// This will timeout after 5 seconds
		router.ServeHTTP(w, req)

		if w.Code != http.StatusRequestTimeout {
			t.Errorf("Expected status 408, got %d", w.Code)
		}

		var errResp models.ErrorResponse
		json.NewDecoder(w.Body).Decode(&errResp)
		if errResp.Error != "transaction timeout" {
			t.Errorf("Expected 'transaction timeout' error, got %s", errResp.Error)
		}
	})

	// Test transaction status polling
	t.Run("TransactionStatusPolling", func(t *testing.T) {
		// Store a test transaction
		tx := &models.Transaction{
			TrxID:  "TEST-TRX-002",
			Status: models.StatusPending,
		}
		transactionStore.Store("TEST-TRX-002", tx)

		req := httptest.NewRequest("GET", "/api/v1/transaction/status/TEST-TRX-002", nil)
		req.SetPathValue("trx_id", "TEST-TRX-002") // Set path parameter for Go 1.22+
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var statusResp models.StatusResponse
		json.NewDecoder(w.Body).Decode(&statusResp)
		if statusResp.Status != "PENDING" {
			t.Errorf("Expected status PENDING, got %s", statusResp.Status)
		}
	})

	// Test EDC response handler
	t.Run("EDCResponseHandler", func(t *testing.T) {
		// Store a pending transaction
		notifyChan := make(chan struct{}, 1)
		tx := &models.Transaction{
			TrxID:      "TEST-TRX-003",
			Status:     models.StatusPending,
			NotifyChan: notifyChan,
		}
		transactionStore.Store("TEST-TRX-003", tx)

		// Simulate EDC response
		edcResponse := models.EDCResponse{
			TrxID:  "TEST-TRX-003",
			Status: "success",
			Amount: "100000",
		}

		edcHandler.HandleEDCResponse(edcResponse)

		// Verify transaction was updated
		updatedTx, ok := transactionStore.Load("TEST-TRX-003")
		if !ok {
			t.Fatal("Transaction not found after EDC response")
		}

		if updatedTx.Status != models.StatusSuccess {
			t.Errorf("Expected status SUCCESS, got %s", updatedTx.Status)
		}

		if updatedTx.ResponseData == nil {
			t.Error("Expected response data to be set")
		} else if updatedTx.ResponseData.Amount != "100000" {
			t.Errorf("Expected amount 100000, got %s", updatedTx.ResponseData.Amount)
		}
	})

	t.Log("Full integration test completed successfully")
}

// mockAblyPublisher is a mock implementation of ably.AblyPublisher for testing
type mockAblyPublisher struct {
	publishFunc   func(serialNumber, token, trxID string) error
	subscribeFunc func(handler func(response models.EDCResponse)) error
}

func (m *mockAblyPublisher) PublishPaymentRequest(serialNumber, token, trxID string) error {
	if m.publishFunc != nil {
		return m.publishFunc(serialNumber, token, trxID)
	}
	return nil
}

func (m *mockAblyPublisher) SubscribeToResponses(handler func(response models.EDCResponse)) error {
	if m.subscribeFunc != nil {
		return m.subscribeFunc(handler)
	}
	return nil
}

var _ ably.AblyPublisher = (*mockAblyPublisher)(nil)
