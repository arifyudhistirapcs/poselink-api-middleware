# Redis Polling Fix - Transaction Response Handling

## Problem

Ketika menggunakan Redis sebagai storage backend, transaction handler tidak bisa menerima notifikasi dari EDC response handler karena:

1. Transaction handler membuat `notifyChan` (Go channel) untuk menunggu response
2. Transaction disimpan ke Redis, tapi `notifyChan` tidak bisa di-serialize (tag `json:"-"`)
3. EDC response handler me-load transaction dari Redis, `NotifyChan` sudah nil
4. Notifikasi tidak sampai, request timeout setelah 60 detik

## Solution

Menggunakan **polling strategy** sebagai pengganti channel notification:

- Transaction handler melakukan polling setiap 500ms
- Mengecek apakah status transaction berubah dari `PENDING`
- Jika berubah, langsung return response
- Jika timeout (60 detik), update status ke `TIMEOUT`

## Changes Made

### File: `internal/handlers/transaction.go`

**Before** (Channel-based):
```go
select {
case <-notifyChan:
    // Response received
    updatedTx, ok := h.store.Load(req.TrxID)
    // ...
case <-ctx.Done():
    // Timeout
}
```

**After** (Polling-based):
```go
ticker := time.NewTicker(500 * time.Millisecond)
defer ticker.Stop()

for {
    select {
    case <-ticker.C:
        // Poll transaction store for updates
        updatedTx, ok := h.store.Load(req.TrxID)
        if updatedTx.Status != models.StatusPending {
            // Response received, return result
            return
        }
    case <-ctx.Done():
        // Timeout
        return
    }
}
```

## Testing

### 1. Start Redis
```bash
redis-server
```

### 2. Start Application
```bash
./start_app.sh
```

### 3. Send Transaction Request (Postman or curl)
```bash
POST http://localhost:8080/api/v1/transaction
Content-Type: application/json

{
  "token": "X02e/",
  "mid": "1999115921",
  "tid": "10747684",
  "trx_id": "AGAP5uMLLE3JS1003"
}
```

### 4. Publish Response to Ably

**Channel**: `response:PBM423AP31788`

**Message**:
```json
{
  "trx_id": "AGAP5uMLLE3JS1003",
  "status": "success",
  "data": {
    "approval_code": "APP123",
    "message": "Payment approved"
  }
}
```

### 5. Expected Result

Postman/curl akan menerima response dalam **< 1 detik** setelah Ably message dipublish:

```json
{
  "trx_id": "AGAP5uMLLE3JS1003",
  "status": "success",
  "data": {
    "approval_code": "APP123",
    "message": "Payment approved"
  }
}
```

## Performance Considerations

- **Polling interval**: 500ms (dapat disesuaikan)
- **Latency**: Maksimal 500ms setelah EDC response diterima
- **Redis load**: 2 queries per second per waiting transaction
- **Timeout**: 60 detik (120 polling attempts)

## Alternative Solutions (Future)

1. **Redis Pub/Sub**: Gunakan Redis PUBLISH/SUBSCRIBE untuk real-time notification
2. **Redis Streams**: Gunakan Redis Streams untuk event-driven architecture
3. **WebSocket**: Gunakan WebSocket untuk push notification ke client

## Verification

Check application logs untuk memastikan polling bekerja:

```bash
tail -f app.log
```

Expected log output:
```
2026/03/13 10:13:56 Starting Payment Middleware server on port 8080
2026/03/13 10:14:01 Successfully processed EDC response for transaction AGAP5uMLLE3JS1003 with status SUCCESS
```

Check Redis untuk verify transaction status:
```bash
redis-cli GET "transaction:AGAP5uMLLE3JS1003"
```

Expected output:
```json
{
  "trx_id": "AGAP5uMLLE3JS1003",
  "status": "SUCCESS",
  "created_at": "2026-03-13T10:14:00+07:00",
  "updated_at": "2026-03-13T10:14:01+07:00",
  "request_data": {...},
  "response_data": {...}
}
```
