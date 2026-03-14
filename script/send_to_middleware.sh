#!/bin/bash

# Script untuk mengirim transaction ke middleware
# Usage: ./send_to_middleware.sh [amount] [action]

# Configuration
MIDDLEWARE_URL="http://localhost:8080/api/v1/transaction"
MID="1999115921"
TID="10747684"

# Default values
AMOUNT=${1:-100000}
ACTION=${2:-Sale}

echo "=== Sending Transaction to Middleware ==="
echo "Amount: Rp $AMOUNT"
echo "Action: $ACTION"
echo ""

# Generate trx_id first
TRX_ID="TRX$(date +%s)000"

# Generate transaction JSON
TRANSACTION=$(cat <<EOF
{
  "amount": "$AMOUNT",
  "action": "$ACTION",
  "trx_id": "$TRX_ID",
  "pos_address": "192.168.10.1",
  "time_stamp": "$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")",
  "method": "purchase"
}
EOF
)

echo "Transaction:"
echo "$TRANSACTION"
echo ""

# Encrypt transaction
echo "Encrypting..."
TOKEN=$(node -e "
const crypto = require('crypto');
const ENCRYPTION_SECRET = 'ECR2022secretKey';

function encrypt(data) {
    const sha1Hash = crypto.createHash('sha1').update(ENCRYPTION_SECRET, 'utf8').digest('hex');
    const keyHex = sha1Hash.substring(0, 32);
    const keyBuffer = Buffer.from(keyHex, 'hex');
    const cipher = crypto.createCipheriv('aes-128-ecb', keyBuffer, null);
    let encrypted = cipher.update(data, 'utf8', 'base64');
    encrypted += cipher.final('base64');
    return encrypted;
}

const transaction = $TRANSACTION;
console.log(encrypt(JSON.stringify(transaction)));
")

echo "Encrypted token (first 50 chars): ${TOKEN:0:50}..."
echo ""

# Create payload
PAYLOAD=$(cat <<EOF
{
  "token": "$TOKEN",
  "mid": "$MID",
  "tid": "$TID",
  "trx_id": "$TRX_ID"
}
EOF
)

echo "Sending to middleware..."
echo "URL: $MIDDLEWARE_URL"
echo ""

# Send to middleware
RESPONSE=$(curl -s -X POST "$MIDDLEWARE_URL" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD")

echo "Response:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

echo "=== Done ==="
echo "Check your EDC device for the transaction"

