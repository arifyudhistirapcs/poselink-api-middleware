#!/bin/bash

# Test script that simulates full transaction flow with Ably response

echo "==================================="
echo "Full Transaction Flow Test"
echo "==================================="
echo ""

# Step 1: Send transaction request (this will wait for response)
echo "1. Sending transaction request..."
echo "   MID: 1999115921"
echo "   TID: 10747684"
echo "   TRX_ID: AGAP5uMLLE3JS1003"
echo ""

# Start transaction in background
curl -X POST http://localhost:8080/api/v1/transaction \
  -H "Content-Type: application/json" \
  -d '{
    "token": "X02e/",
    "mid": "1999115921",
    "tid": "10747684",
    "trx_id": "AGAP5uMLLE3JS1003"
  }' &

CURL_PID=$!

echo "Transaction request sent (waiting for response)..."
echo ""
echo "2. Now publish SUCCESS response to Ably channel: response:PBM423AP31788"
echo ""
echo "   Use this JSON in Ably console:"
echo '   {
     "trx_id": "AGAP5uMLLE3JS1003",
     "status": "success",
     "data": {
       "approval_code": "APP123",
       "message": "Payment approved"
     }
   }'
echo ""
echo "3. Waiting for transaction to complete..."
echo ""

# Wait for curl to finish
wait $CURL_PID

echo ""
echo "==================================="
echo "Transaction completed!"
echo "==================================="
