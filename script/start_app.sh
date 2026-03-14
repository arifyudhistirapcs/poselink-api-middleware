#!/bin/bash

# Load environment variables
export ABLY_API_KEY="jKHFtA.3mx-Zw:njUj9PK5NZOliwWa5SDsx9aBlaI6dFXwKU0zDB_dfJA"
export ENCRYPTION_SECRET="ECR2022secretKey"
export SERVER_PORT=8080
export TIMEOUT_DURATION=60
export MIDTID_MAPPINGS='{"1999115921:10747684":"PBM423AP31788","M001:T001":"SN12345","M002:T002":"SN67890"}'
export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_PASSWORD=""
export REDIS_DB=0
export REDIS_MIN_IDLE_CONNS=5
export REDIS_MAX_CONNS=100

# Start application
./payment-middleware
