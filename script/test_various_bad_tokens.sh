#!/bin/bash

MIDDLEWARE_URL="http://localhost:8080/api/v1/transaction"
MID="1999115921"
TID="10747684"

echo "=== Testing Various Bad Token Scenarios ==="
echo ""

# Test 1: Empty token
echo "Test 1: Empty Token"
TRX_ID="TRX$(date +%s)001"
curl -s -X POST "$MIDDLEWARE_URL" \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"\",\"mid\":\"$MID\",\"tid\":\"$TID\",\"trx_id\":\"$TRX_ID\"}" | jq -r '.trx_id, .msg' | head -2
echo ""

sleep 2

# Test 2: Invalid Base64
echo "Test 2: Invalid Base64 (special characters)"
TRX_ID="TRX$(date +%s)002"
curl -s -X POST "$MIDDLEWARE_URL" \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"!!!invalid@@@\",\"mid\":\"$MID\",\"tid\":\"$TID\",\"trx_id\":\"$TRX_ID\"}" | jq -r '.trx_id, .msg' | head -2
echo ""

sleep 2

# Test 3: Valid Base64 but wrong encryption
echo "Test 3: Valid Base64 but wrong encryption"
TRX_ID="TRX$(date +%s)003"
curl -s -X POST "$MIDDLEWARE_URL" \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"SGVsbG9Xb3JsZA==\",\"mid\":\"$MID\",\"tid\":\"$TID\",\"trx_id\":\"$TRX_ID\"}" | jq -r '.trx_id, .msg' | head -2
echo ""

sleep 2

# Test 4: Very long invalid token
echo "Test 4: Very long invalid token"
TRX_ID="TRX$(date +%s)004"
LONG_TOKEN=$(printf 'A%.0s' {1..1000})
curl -s -X POST "$MIDDLEWARE_URL" \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"$LONG_TOKEN\",\"mid\":\"$MID\",\"tid\":\"$TID\",\"trx_id\":\"$TRX_ID\"}" | jq -r '.trx_id, .msg' | head -2
echo ""

echo "=== All Tests Complete ==="
