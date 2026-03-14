#!/bin/bash

# Quick Test Script - Manual Testing Commands
# Run these commands one by one after starting the application

echo "Quick Test Commands for Redis Storage Integration"
echo "=================================================="
echo ""
echo "Prerequisites:"
echo "1. Start Redis: redis-server"
echo "2. Start app: ./payment-middleware"
echo ""
echo "=================================================="
echo ""

echo "# 1. Check health"
echo "curl http://localhost:8080/health"
echo ""

echo "# 2. Migrate all mappings from .env to Redis"
echo "curl -X POST http://localhost:8080/api/v1/admin/migrate \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"force\": true}'"
echo ""

echo "# 3. Add the new mapping (MID: 1999115921, TID: 10747684)"
echo "curl -X POST http://localhost:8080/api/v1/admin/mapping \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"mid\": \"1999115921\", \"tid\": \"10747684\", \"serial_number\": \"PBM423AP31788\"}'"
echo ""

echo "# 4. Verify in Redis directly"
echo "redis-cli GET 'mapping:mid:1999115921:tid:10747684'"
echo ""

echo "# 5. List all mappings in Redis"
echo "redis-cli KEYS 'mapping:*'"
echo ""

echo "# 6. Test transaction with the new mapping"
echo "curl -X POST http://localhost:8080/api/v1/transaction \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"trx_id\": \"TRX001\", \"mid\": \"1999115921\", \"tid\": \"10747684\", \"token\": \"test_token\"}'"
echo ""

echo "# 7. Check transaction status"
echo "curl http://localhost:8080/api/v1/transaction/status/TRX001"
echo ""

echo "# 8. Check transaction TTL"
echo "curl http://localhost:8080/api/v1/admin/transaction/TRX001/ttl"
echo ""

echo "# 9. Extend transaction TTL (to 2 hours = 7200 seconds)"
echo "curl -X POST http://localhost:8080/api/v1/admin/transaction/TRX001/extend-ttl \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"duration_seconds\": 7200}'"
echo ""

echo "# 10. Delete a mapping"
echo "curl -X DELETE 'http://localhost:8080/api/v1/admin/mapping?mid=M001&tid=T001'"
echo ""

echo "=================================================="
echo "Redis CLI Commands:"
echo "=================================================="
echo ""
echo "# View all keys"
echo "redis-cli KEYS '*'"
echo ""
echo "# View specific mapping"
echo "redis-cli GET 'mapping:mid:1999115921:tid:10747684'"
echo ""
echo "# View transaction"
echo "redis-cli GET 'transaction:TRX001'"
echo ""
echo "# Check TTL"
echo "redis-cli TTL 'transaction:TRX001'"
echo ""
echo "# Flush all data (CAUTION!)"
echo "redis-cli FLUSHALL"
echo ""
