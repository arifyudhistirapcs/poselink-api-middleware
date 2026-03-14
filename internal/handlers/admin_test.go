package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"payment-middleware/internal/mapper"
)

func setupAdminTest(t *testing.T) (*AdminHandler, *miniredis.Miniredis, *redis.Client) {
	// Create miniredis instance
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Create mapper and handler
	redisMapper := mapper.NewRedisMIDTIDMapper(client)
	handler := NewAdminHandler(redisMapper, client)

	return handler, mr, client
}

func TestCreateOrUpdateMapping_Success(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create request
	reqBody := MappingRequest{
		MID:          "M001",
		TID:          "T001",
		SerialNumber: "SN12345",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/mapping", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.CreateOrUpdateMapping(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)
	if response["message"] != "mapping created/updated successfully" {
		t.Errorf("unexpected response message: %s", response["message"])
	}

	// Verify mapping was stored
	serialNumber, err := handler.mapper.GetSerialNumber("M001", "T001")
	if err != nil {
		t.Errorf("failed to retrieve mapping: %v", err)
	}
	if serialNumber != "SN12345" {
		t.Errorf("expected serial number SN12345, got %s", serialNumber)
	}
}

func TestCreateOrUpdateMapping_UpdateExisting(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create initial mapping
	handler.mapper.SetMapping("M001", "T001", "SN_OLD")

	// Update with new serial number
	reqBody := MappingRequest{
		MID:          "M001",
		TID:          "T001",
		SerialNumber: "SN_NEW",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/mapping", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.CreateOrUpdateMapping(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify mapping was updated
	serialNumber, err := handler.mapper.GetSerialNumber("M001", "T001")
	if err != nil {
		t.Errorf("failed to retrieve mapping: %v", err)
	}
	if serialNumber != "SN_NEW" {
		t.Errorf("expected serial number SN_NEW, got %s", serialNumber)
	}
}

func TestCreateOrUpdateMapping_InvalidJSON(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create request with invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/mapping", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	// Execute
	handler.CreateOrUpdateMapping(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCreateOrUpdateMapping_MissingMID(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create request without MID
	reqBody := MappingRequest{
		TID:          "T001",
		SerialNumber: "SN12345",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/mapping", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.CreateOrUpdateMapping(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCreateOrUpdateMapping_MissingTID(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create request without TID
	reqBody := MappingRequest{
		MID:          "M001",
		SerialNumber: "SN12345",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/mapping", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.CreateOrUpdateMapping(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCreateOrUpdateMapping_MissingSerialNumber(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create request without SerialNumber
	reqBody := MappingRequest{
		MID: "M001",
		TID: "T001",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/mapping", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.CreateOrUpdateMapping(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestDeleteMapping_Success(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create initial mapping
	handler.mapper.SetMapping("M001", "T001", "SN12345")

	// Create delete request
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/mapping?mid=M001&tid=T001", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.DeleteMapping(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)
	if response["message"] != "mapping deleted successfully" {
		t.Errorf("unexpected response message: %s", response["message"])
	}

	// Verify mapping was deleted
	_, err := handler.mapper.GetSerialNumber("M001", "T001")
	if err == nil {
		t.Error("expected error when retrieving deleted mapping")
	}
}

func TestDeleteMapping_MissingMID(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create delete request without MID
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/mapping?tid=T001", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.DeleteMapping(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestDeleteMapping_MissingTID(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create delete request without TID
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/mapping?mid=M001", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.DeleteMapping(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestDeleteMapping_NonExistentMapping(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create delete request for non-existent mapping
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/mapping?mid=M999&tid=T999", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.DeleteMapping(w, req)

	// Assert - Redis DEL returns success even if key doesn't exist
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMigrateMapping_Success(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create mappings to migrate
	mappings := map[string]string{
		"M001:T001": "SN12345",
		"M002:T002": "SN67890",
	}

	// Create request
	reqBody := MigrateRequest{Force: false}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/migrate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.MigrateMapping(w, req, mappings)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	if response["success_count"].(float64) != 2 {
		t.Errorf("expected 2 successful migrations, got %v", response["success_count"])
	}

	// Verify mappings were stored
	sn1, err := handler.mapper.GetSerialNumber("M001", "T001")
	if err != nil || sn1 != "SN12345" {
		t.Errorf("expected SN12345, got %s (err: %v)", sn1, err)
	}

	sn2, err := handler.mapper.GetSerialNumber("M002", "T002")
	if err != nil || sn2 != "SN67890" {
		t.Errorf("expected SN67890, got %s (err: %v)", sn2, err)
	}
}

func TestMigrateMapping_WithForceFlag(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Pre-populate a mapping
	handler.mapper.SetMapping("M001", "T001", "OLD_SN")

	// Create mappings to migrate
	mappings := map[string]string{
		"M001:T001": "NEW_SN",
	}

	// Create request with force flag
	reqBody := MigrateRequest{Force: true}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/migrate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.MigrateMapping(w, req, mappings)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify mapping was overwritten
	sn, err := handler.mapper.GetSerialNumber("M001", "T001")
	if err != nil || sn != "NEW_SN" {
		t.Errorf("expected NEW_SN, got %s (err: %v)", sn, err)
	}
}

func TestMigrateMapping_SkipExisting(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Pre-populate a mapping
	handler.mapper.SetMapping("M001", "T001", "EXISTING_SN")

	// Create mappings to migrate
	mappings := map[string]string{
		"M001:T001": "NEW_SN",
		"M002:T002": "SN67890",
	}

	// Create request without force flag
	reqBody := MigrateRequest{Force: false}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/migrate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.MigrateMapping(w, req, mappings)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	if response["success_count"].(float64) != 1 {
		t.Errorf("expected 1 successful migration, got %v", response["success_count"])
	}

	// Verify existing mapping was not overwritten
	sn1, err := handler.mapper.GetSerialNumber("M001", "T001")
	if err != nil || sn1 != "EXISTING_SN" {
		t.Errorf("expected EXISTING_SN, got %s (err: %v)", sn1, err)
	}

	// Verify new mapping was added
	sn2, err := handler.mapper.GetSerialNumber("M002", "T002")
	if err != nil || sn2 != "SN67890" {
		t.Errorf("expected SN67890, got %s (err: %v)", sn2, err)
	}
}

func TestMigrateMapping_InvalidKeyFormat(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create mappings with invalid key format
	mappings := map[string]string{
		"INVALID_KEY": "SN12345",
		"M001:T001":   "SN67890",
	}

	// Create request
	reqBody := MigrateRequest{Force: false}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/migrate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.MigrateMapping(w, req, mappings)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	if response["success_count"].(float64) != 1 {
		t.Errorf("expected 1 successful migration, got %v", response["success_count"])
	}
	if response["error_count"].(float64) != 1 {
		t.Errorf("expected 1 error, got %v", response["error_count"])
	}
}

func TestGetTransactionTTL_Success(t *testing.T) {
	handler, mr, client := setupAdminTest(t)
	defer mr.Close()

	// Store a transaction with TTL
	ctx := context.Background()
	key := "transaction:TRX123"
	client.Set(ctx, key, `{"trx_id":"TRX123"}`, 3600*time.Second) // 1 hour

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/transaction/TRX123/ttl", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.GetTransactionTTL(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	if response["trx_id"] != "TRX123" {
		t.Errorf("expected trx_id TRX123, got %v", response["trx_id"])
	}
	if response["has_expiration"] != true {
		t.Errorf("expected has_expiration true, got %v", response["has_expiration"])
	}
}

func TestGetTransactionTTL_NotFound(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create request for non-existent transaction
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/transaction/NONEXISTENT/ttl", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.GetTransactionTTL(w, req)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestExtendTransactionTTL_Success(t *testing.T) {
	handler, mr, client := setupAdminTest(t)
	defer mr.Close()

	// Store a transaction with TTL
	ctx := context.Background()
	key := "transaction:TRX123"
	client.Set(ctx, key, `{"trx_id":"TRX123"}`, 3600*time.Second) // 1 hour

	// Create request to extend TTL
	reqBody := ExtendTTLRequest{DurationSeconds: 7200}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/transaction/TRX123/extend-ttl", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.ExtendTransactionTTL(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	if response["trx_id"] != "TRX123" {
		t.Errorf("expected trx_id TRX123, got %v", response["trx_id"])
	}
}

func TestExtendTransactionTTL_NotFound(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create request for non-existent transaction
	reqBody := ExtendTTLRequest{DurationSeconds: 7200}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/transaction/NONEXISTENT/extend-ttl", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.ExtendTransactionTTL(w, req)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestExtendTransactionTTL_InvalidDuration(t *testing.T) {
	handler, mr, client := setupAdminTest(t)
	defer mr.Close()

	// Store a transaction with TTL
	ctx := context.Background()
	key := "transaction:TRX123"
	client.Set(ctx, key, `{"trx_id":"TRX123"}`, 3600*time.Second)

	// Create request with invalid duration
	reqBody := ExtendTTLRequest{DurationSeconds: -100}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/transaction/TRX123/extend-ttl", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.ExtendTransactionTTL(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestExtendTransactionTTL_InvalidJSON(t *testing.T) {
	handler, mr, _ := setupAdminTest(t)
	defer mr.Close()

	// Create request with invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/transaction/TRX123/extend-ttl", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	// Execute
	handler.ExtendTransactionTTL(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
