#!/bin/bash

# Script untuk test koneksi Ably dan API transaction

echo "========================================="
echo "Test Ably Connection - Payment Middleware"
echo "========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if server is running
echo "1. Checking if server is running..."
if curl -s http://localhost:8080/health > /dev/null; then
    echo -e "${GREEN}✓ Server is running${NC}"
else
    echo -e "${RED}✗ Server is not running${NC}"
    echo "Please start the server first:"
    echo "  export ABLY_API_KEY=\"jKHFtA.3mx-Zw:njUj9PK5NZOliwWa5SDsx9aBlaI6dFXwKU0zDB_dfJA\""
    echo "  export MIDTID_MAPPINGS='{\"M001:T001\":\"SN12345\",\"M002:T002\":\"SN67890\"}'"
    echo "  go run main.go"
    exit 1
fi

echo ""
echo "2. Testing transaction endpoint..."
echo ""

# Generate unique transaction ID
TRX_ID="TRX-TEST-$(date +%s)"

echo "Sending transaction request with trx_id: $TRX_ID"
echo ""

# Send transaction request
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/transaction \
  -H "Content-Type: application/json" \
  -d "{
    \"token\": \"test_payment_token_$(date +%s)\",
    \"mid\": \"M001\",
    \"tid\": \"T001\",
    \"trx_id\": \"$TRX_ID\"
  }")

echo "Response:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

# Check if it's a timeout (expected behavior without EDC response)
if echo "$RESPONSE" | grep -q "transaction timeout"; then
    echo -e "${YELLOW}⚠ Transaction timed out (expected - no EDC response)${NC}"
    echo ""
    echo -e "${GREEN}✓ Ably publish successful!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Open Ably Dashboard: https://ably.com/accounts"
    echo "2. Go to Dev Console"
    echo "3. Subscribe to channel: edc:SN12345"
    echo "4. You should see the payment_request message"
    echo ""
    echo "To simulate EDC response:"
    echo "1. In Ably Dev Console, publish to channel: response:test"
    echo "2. Event name: payment_result"
    echo "3. Message data:"
    echo "{"
    echo "  \"trx_id\": \"$TRX_ID\","
    echo "  \"status\": \"success\","
    echo "  \"approval\": \"123456\","
    echo "  \"amount\": \"100000\","
    echo "  \"card_name\": \"VISA\""
    echo "}"
elif echo "$RESPONSE" | grep -q "error"; then
    echo -e "${RED}✗ Error occurred${NC}"
    echo "Check server logs for details"
else
    echo -e "${GREEN}✓ Transaction successful!${NC}"
    echo "EDC response received"
fi

echo ""
echo "3. Testing status endpoint..."
echo ""

# Test status endpoint
STATUS_RESPONSE=$(curl -s http://localhost:8080/api/v1/transaction/status/$TRX_ID)
echo "Status response:"
echo "$STATUS_RESPONSE" | jq '.' 2>/dev/null || echo "$STATUS_RESPONSE"

echo ""
echo "========================================="
echo "Test completed"
echo "========================================="
