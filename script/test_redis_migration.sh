#!/bin/bash

# Test Redis Migration Script
# This script tests the Redis storage integration

echo "==================================="
echo "Redis Storage Integration Test"
echo "==================================="
echo ""

# Check if Redis is running
echo "1. Checking Redis connection..."
redis-cli ping > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ Redis is running"
else
    echo "✗ Redis is not running. Please start Redis first:"
    echo "  brew services start redis  (macOS)"
    echo "  redis-server              (manual)"
    exit 1
fi

echo ""

# Start the application in background
echo "2. Starting Payment Middleware..."
./payment-middleware &
APP_PID=$!
echo "✓ Application started (PID: $APP_PID)"

# Wait for application to be ready
echo "3. Waiting for application to be ready..."
sleep 3

# Check health endpoint
echo "4. Checking health endpoint..."
HEALTH_RESPONSE=$(curl -s http://localhost:8080/health)
echo "Response: $HEALTH_RESPONSE"

if echo "$HEALTH_RESPONSE" | grep -q "healthy"; then
    echo "✓ Health check passed"
else
    echo "✗ Health check failed"
    kill $APP_PID
    exit 1
fi

echo ""
echo "==================================="
echo "Testing Admin Endpoints"
echo "==================================="
echo ""

# Test 1: Migrate mappings from .env to Redis
echo "5. Migrating mappings from .env to Redis..."
MIGRATE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/admin/migrate \
  -H "Content-Type: application/json" \
  -d '{"force": true}')
echo "Response: $MIGRATE_RESPONSE"
echo ""

# Test 2: Add new mapping via API
echo "6. Adding new mapping via API..."
ADD_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/admin/mapping \
  -H "Content-Type: application/json" \
  -d '{
    "mid": "1999115921",
    "tid": "10747684",
    "serial_number": "PBM423AP31788"
  }')
echo "Response: $ADD_RESPONSE"
echo ""

# Test 3: Verify mapping in Redis directly
echo "7. Verifying mapping in Redis..."
REDIS_VALUE=$(redis-cli GET "mapping:mid:1999115921:tid:10747684")
echo "Redis value for mapping:mid:1999115921:tid:10747684 = $REDIS_VALUE"

if [ "$REDIS_VALUE" = "PBM423AP31788" ]; then
    echo "✓ Mapping verified in Redis"
else
    echo "✗ Mapping not found or incorrect in Redis"
fi

echo ""

# Test 4: Check all mappings in Redis
echo "8. Listing all mappings in Redis..."
echo "All mapping keys:"
redis-cli KEYS "mapping:*"
echo ""

# Test 5: Test transaction storage
echo "9. Testing transaction creation..."
TRX_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/transaction \
  -H "Content-Type: application/json" \
  -d '{
    "trx_id": "TEST123",
    "mid": "1999115921",
    "tid": "10747684",
    "token": "test_token_123"
  }')
echo "Response: $TRX_RESPONSE"
echo ""

# Test 6: Check transaction in Redis
echo "10. Checking transaction in Redis..."
REDIS_TRX=$(redis-cli GET "transaction:TEST123")
echo "Transaction data: $REDIS_TRX"
echo ""

# Test 7: Check transaction TTL
echo "11. Checking transaction TTL..."
TTL_RESPONSE=$(curl -s http://localhost:8080/api/v1/admin/transaction/TEST123/ttl)
echo "Response: $TTL_RESPONSE"
echo ""

# Test 8: Delete mapping
echo "12. Testing delete mapping..."
DELETE_RESPONSE=$(curl -s -X DELETE "http://localhost:8080/api/v1/admin/mapping?mid=M001&tid=T001")
echo "Response: $DELETE_RESPONSE"
echo ""

echo "==================================="
echo "Test Summary"
echo "==================================="
echo ""
echo "All tests completed!"
echo ""
echo "To view all Redis keys:"
echo "  redis-cli KEYS '*'"
echo ""
echo "To view specific mapping:"
echo "  redis-cli GET 'mapping:mid:1999115921:tid:10747684'"
echo ""
echo "To view transaction:"
echo "  redis-cli GET 'transaction:TEST123'"
echo ""

# Cleanup
echo "Stopping application..."
kill $APP_PID
echo "✓ Application stopped"
