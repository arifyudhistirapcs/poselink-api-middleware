package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"payment-middleware/internal/models"
)

func TestWriteErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
	}{
		{
			name:       "400 Bad Request",
			statusCode: http.StatusBadRequest,
			message:    "transaction_id is required",
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			message:    "transaction not found",
		},
		{
			name:       "408 Request Timeout",
			statusCode: http.StatusRequestTimeout,
			message:    "transaction timeout",
		},
		{
			name:       "503 Service Unavailable",
			statusCode: http.StatusServiceUnavailable,
			message:    "ably connection error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteErrorResponse(w, tt.statusCode, tt.message)

			// Check status code
			if w.Code != tt.statusCode {
				t.Errorf("expected status code %d, got %d", tt.statusCode, w.Code)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			// Check response body
			var response models.ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.Error != tt.message {
				t.Errorf("expected error message %q, got %q", tt.message, response.Error)
			}
		})
	}
}

func TestLogError(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	tests := []struct {
		name     string
		trxID    string
		mid      string
		tid      string
		errorMsg string
	}{
		{
			name:     "All fields present",
			trxID:    "TRX123",
			mid:      "MID001",
			tid:      "TID001",
			errorMsg: "test error message",
		},
		{
			name:     "Missing trx_id",
			trxID:    "",
			mid:      "MID001",
			tid:      "TID001",
			errorMsg: "validation error",
		},
		{
			name:     "Missing mid and tid",
			trxID:    "TRX456",
			mid:      "",
			tid:      "",
			errorMsg: "transaction not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			LogError(tt.trxID, tt.mid, tt.tid, tt.errorMsg)

			logOutput := buf.String()

			// Check that log contains expected fields
			if !strings.Contains(logOutput, "Error") {
				t.Error("log output should contain 'Error'")
			}
			if !strings.Contains(logOutput, tt.errorMsg) {
				t.Errorf("log output should contain error message %q", tt.errorMsg)
			}
			if tt.trxID != "" && !strings.Contains(logOutput, tt.trxID) {
				t.Errorf("log output should contain trx_id %q", tt.trxID)
			}
			if tt.mid != "" && !strings.Contains(logOutput, tt.mid) {
				t.Errorf("log output should contain mid %q", tt.mid)
			}
			if tt.tid != "" && !strings.Contains(logOutput, tt.tid) {
				t.Errorf("log output should contain tid %q", tt.tid)
			}
		})
	}
}

func TestLogErrorWithDetails(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	details := map[string]interface{}{
		"serial_number": "SN123",
		"attempt":       3,
	}

	LogErrorWithDetails("TRX789", "MID002", "TID002", "publish failed", details)

	logOutput := buf.String()

	// Check that log contains expected fields
	if !strings.Contains(logOutput, "Error") {
		t.Error("log output should contain 'Error'")
	}
	if !strings.Contains(logOutput, "TRX789") {
		t.Error("log output should contain trx_id")
	}
	if !strings.Contains(logOutput, "MID002") {
		t.Error("log output should contain mid")
	}
	if !strings.Contains(logOutput, "TID002") {
		t.Error("log output should contain tid")
	}
	if !strings.Contains(logOutput, "publish failed") {
		t.Error("log output should contain error message")
	}
	if !strings.Contains(logOutput, "details") {
		t.Error("log output should contain details")
	}
}

func TestPanicRecoveryMiddleware(t *testing.T) {
	tests := []struct {
		name          string
		handler       http.HandlerFunc
		shouldPanic   bool
		expectedCode  int
		expectedError string
	}{
		{
			name: "No panic",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			},
			shouldPanic:  false,
			expectedCode: http.StatusOK,
		},
		{
			name: "Panic with string",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("something went wrong")
			},
			shouldPanic:   true,
			expectedCode:  http.StatusInternalServerError,
			expectedError: "internal server error",
		},
		{
			name: "Panic with error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic(http.ErrAbortHandler)
			},
			shouldPanic:   true,
			expectedCode:  http.StatusInternalServerError,
			expectedError: "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr)

			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			// Wrap handler with panic recovery middleware
			wrappedHandler := PanicRecoveryMiddleware(tt.handler)
			wrappedHandler.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedCode {
				t.Errorf("expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			// If panic was expected, check error response
			if tt.shouldPanic {
				var response models.ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}

				if response.Error != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, response.Error)
				}

				// Check that panic was logged
				logOutput := buf.String()
				if !strings.Contains(logOutput, "Panic recovered") {
					t.Error("log should contain 'Panic recovered'")
				}
				if !strings.Contains(logOutput, "Stack trace") {
					t.Error("log should contain stack trace")
				}
			}
		})
	}
}

func TestPanicRecoveryMiddleware_Integration(t *testing.T) {
	// Test that middleware allows normal requests to pass through
	normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler := PanicRecoveryMiddleware(normalHandler)
	wrappedHandler.ServeHTTP(w, req)

	// Check that response is unmodified
	if w.Code != http.StatusCreated {
		t.Errorf("expected status code %d, got %d", http.StatusCreated, w.Code)
	}

	if w.Header().Get("X-Custom-Header") != "test-value" {
		t.Error("custom header should be preserved")
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "created" {
		t.Error("response body should be preserved")
	}
}
