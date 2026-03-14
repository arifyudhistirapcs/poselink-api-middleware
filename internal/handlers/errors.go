package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"

	"payment-middleware/internal/models"
)

// WriteErrorResponse writes a JSON error response with logging
// This function ensures all error responses use the ErrorResponse struct with "error" field
func WriteErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(models.ErrorResponse{Error: message})
}

// LogError logs an error with context information
func LogError(trxID, mid, tid, errorMsg string) {
	log.Printf("Error - trx_id: %s, mid: %s, tid: %s, error: %s", trxID, mid, tid, errorMsg)
}

// LogErrorWithDetails logs an error with additional details
func LogErrorWithDetails(trxID, mid, tid, errorMsg string, details interface{}) {
	log.Printf("Error - trx_id: %s, mid: %s, tid: %s, error: %s, details: %+v", trxID, mid, tid, errorMsg, details)
}

// PanicRecoveryMiddleware recovers from panics and returns a 500 error response
func PanicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				log.Printf("Panic recovered: %v\nStack trace:\n%s", err, debug.Stack())
				
				// Return 500 error response
				WriteErrorResponse(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		
		next.ServeHTTP(w, r)
	})
}
