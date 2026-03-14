#!/bin/bash

# Script untuk manage MID/TID to Serial Number mapping via API
# Usage:
#   ./manage_mapping.sh add <mid> <tid> <serial_number>
#   ./manage_mapping.sh delete <mid> <tid>
#   ./manage_mapping.sh list
#   ./manage_mapping.sh migrate [--force]

MIDDLEWARE_URL="http://localhost:8080"

ACTION=${1:-help}

case "$ACTION" in
  add)
    MID=$2
    TID=$3
    SN=$4

    if [ -z "$MID" ] || [ -z "$TID" ] || [ -z "$SN" ]; then
      echo "Usage: $0 add <mid> <tid> <serial_number>"
      echo "Example: $0 add 1234567890 99887766 PBM423AP31788"
      exit 1
    fi

    echo "=== Adding Mapping ==="
    echo "MID: $MID"
    echo "TID: $TID"
    echo "Serial: $SN"
    echo ""

    RESPONSE=$(curl -s -X POST "$MIDDLEWARE_URL/api/v1/admin/mapping" \
      -H "Content-Type: application/json" \
      -d "{\"mid\":\"$MID\",\"tid\":\"$TID\",\"serial_number\":\"$SN\"}")

    echo "Response:"
    echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
    ;;

  delete)
    MID=$2
    TID=$3

    if [ -z "$MID" ] || [ -z "$TID" ]; then
      echo "Usage: $0 delete <mid> <tid>"
      echo "Example: $0 delete 1234567890 99887766"
      exit 1
    fi

    echo "=== Deleting Mapping ==="
    echo "MID: $MID"
    echo "TID: $TID"
    echo ""

    RESPONSE=$(curl -s -X DELETE "$MIDDLEWARE_URL/api/v1/admin/mapping?mid=$MID&tid=$TID")

    echo "Response:"
    echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
    ;;

  list)
    echo "=== All MID/TID Mappings in Redis ==="
    echo ""
    redis-cli KEYS "mapping:mid:*" | while read key; do
      VALUE=$(redis-cli GET "$key")
      # Extract MID and TID from key format "mapping:mid:MID:tid:TID"
      MID=$(echo "$key" | cut -d: -f3)
      TID=$(echo "$key" | cut -d: -f5)
      echo "MID: $MID | TID: $TID | Serial: $VALUE"
    done
    ;;

  migrate)
    FORCE=false
    if [ "$2" = "--force" ]; then
      FORCE=true
    fi

    echo "=== Migrating Mappings from Config to Redis ==="
    echo "Force: $FORCE"
    echo ""

    RESPONSE=$(curl -s -X POST "$MIDDLEWARE_URL/api/v1/admin/migrate" \
      -H "Content-Type: application/json" \
      -d "{\"force\":$FORCE}")

    echo "Response:"
    echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
    ;;

  *)
    echo "=== MID/TID Mapping Manager ==="
    echo ""
    echo "Usage:"
    echo "  $0 add <mid> <tid> <serial_number>  - Add/update a mapping"
    echo "  $0 delete <mid> <tid>               - Delete a mapping"
    echo "  $0 list                              - List all mappings"
    echo "  $0 migrate [--force]                 - Migrate from config to Redis"
    echo ""
    echo "Examples:"
    echo "  $0 add 1234567890 99887766 PBM423AP31788"
    echo "  $0 delete 1234567890 99887766"
    echo "  $0 list"
    echo "  $0 migrate --force"
    ;;
esac
