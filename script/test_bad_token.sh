#!/bin/bash

# Test script to send a bad token to middleware and verify error response

MIDDLEWARE_URL="http://localhost:8080/api/v1/transaction"
MID="1999115921"
TID="10747684"

echo "=== Testing Bad Token Handling ==="
echo ""

# Generate trx_id
TRX_ID="TRX$(date +%s)000"

# Use a bad token (not properly encrypted)
BAD_TOKEN="this_is_not_a_valid_encrypted_token"

# Create payload
PAYLOAD=$(cat <<EOF
{
  "token": "$BAD_TOKEN",
  "mid": "$MID",
  "tid": "$TID",
  "trx_id": "$TRX_ID"
}
EOF
)

echo "Sending bad token to middleware..."
echo "TRX_ID: $TRX_ID"
echo "Token: $BAD_TOKEN"
echo ""

# Send to middleware
RESPONSE=$(curl -s -X POST "$MIDDLEWARE_URL" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD")

echo "Response:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

# Check if response contains the correct trx_id
if echo "$RESPONSE" | grep -q "$TRX_ID"; then
    echo "✓ SUCCESS: Response contains correct trx_id: $TRX_ID"
else
    echo "✗ FAILED: Response does not contain trx_id: $TRX_ID"
fi

echo ""
echo "=== Test Complete ==="
