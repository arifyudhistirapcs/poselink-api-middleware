# Payment Middleware - Deployment Guide

## Overview

Payment Middleware adalah HTTP server yang menjembatani komunikasi antara POS System dan EDC Device melalui Ably Realtime messaging. Middleware menerima request transaksi dari POS, meneruskan ke EDC via Ably, menunggu response dari EDC, lalu mengembalikan hasilnya ke POS.

## Architecture Topology

```
┌──────────┐     HTTP POST      ┌──────────────────┐     Ably Pub/Sub     ┌──────────────┐
│          │  ───────────────►  │                  │  ──────────────────► │              │
│   POS    │                    │    Middleware     │                      │  EDC Device  │
│  System  │  ◄───────────────  │   (Go HTTP)      │  ◄────────────────── │  (Android)   │
│          │     JSON Response  │                  │     Ably Pub/Sub     │              │
└──────────┘                    └──────────────────┘                      └──────────────┘
                                        │
                                        │ Redis
                                        ▼
                                ┌──────────────────┐
                                │                  │
                                │      Redis       │
                                │  (State Store)   │
                                │                  │
                                └──────────────────┘
```

### Flow Detail

```
1. POS  ──► POST /api/v1/transaction {token, mid, tid, trx_id}
2. Middleware lookup MID:TID → Serial Number (Redis)
3. Middleware publish ke Ably channel "edc:{serial}" (JSON: {token, trx_id})
4. EDC menerima, decrypt, proses transaksi
5. EDC encrypt response, publish ke Ably channel "response:*"
6. Middleware menerima, decrypt, match dengan pending transaction
7. Middleware return response ke POS
```

### Encryption Flow

```
POS → Middleware: Token sudah terenkripsi AES-128-ECB (POS yang encrypt)
Middleware → EDC: Token diteruskan apa adanya + trx_id sebagai metadata
EDC → Middleware: Response dienkripsi AES-128-ECB oleh EDC
Middleware: Decrypt response EDC, lalu return sebagai JSON ke POS
```

## Tech Stack

| Component | Technology | Version |
|-----------|-----------|---------|
| Language | Go | 1.25+ |
| HTTP Router | gorilla/mux | 1.8.1 |
| Realtime Messaging | Ably (ably-go) | 1.3.0 |
| State Store | Redis (go-redis) | 9.18.0 |
| Encryption | AES-128-ECB | stdlib crypto/aes |
| Testing | gopter (property-based) | 0.2.11 |

## External Dependencies

| Service | Purpose | Required |
|---------|---------|----------|
| **Ably** | Realtime pub/sub messaging antara Middleware dan EDC | ✅ Yes |
| **Redis** | State store untuk transaction data dan MID/TID mapping | ✅ Yes |

### Ably

- Digunakan untuk komunikasi realtime dengan EDC device
- Middleware publish ke channel `edc:{serial_number}` (event: `payment_request`)
- Middleware subscribe ke channel `response:*` (event: `payment_result`)
- Membutuhkan API key dari dashboard Ably (https://ably.com)

### Redis

- Menyimpan pending transaction state (TTL-based, auto-expire)
- Menyimpan MID/TID → Serial Number mapping (persistent, no TTL)
- Key format transaction: `transaction:{trx_id}`
- Key format mapping: `mapping:mid:{mid}:tid:{tid}`

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ABLY_API_KEY` | ✅ | - | Ably API key (format: `appId.keyId:keySecret`) |
| `ENCRYPTION_SECRET` | ❌ | `ECR2022secretKey` | Shared AES encryption secret (harus sama dengan EDC) |
| `SERVER_PORT` | ❌ | `8080` | HTTP server port |
| `TIMEOUT_DURATION` | ❌ | `60` | Transaction timeout dalam detik |
| `MIDTID_MAPPINGS` | ❌ | `{}` | Initial MID/TID mapping JSON (untuk migration) |
| `REDIS_HOST` | ❌ | `localhost` | Redis server host |
| `REDIS_PORT` | ❌ | `6379` | Redis server port |
| `REDIS_PASSWORD` | ❌ | `` | Redis password |
| `REDIS_DB` | ❌ | `0` | Redis database number |
| `REDIS_MIN_IDLE_CONNS` | ❌ | `5` | Minimum idle Redis connections |
| `REDIS_MAX_CONNS` | ❌ | `100` | Maximum Redis connections |

## API Contract

### Transaction Endpoints

#### POST /api/v1/transaction
Mengirim transaksi ke EDC dan menunggu response (synchronous, blocking).

**Request:**
```json
{
  "token": "encrypted_transaction_data_base64",
  "mid": "1999115921",
  "tid": "10747684",
  "trx_id": "TRX1234567890"
}
```

**Response (200 - Success):**
```json
{
  "acq_mid": "000000000000001",
  "acq_tid": "00000001",
  "action": "Sale",
  "amount": "100000",
  "approval": "123456",
  "batch_number": "000001",
  "card_category": "DEBIT",
  "card_name": "BCA",
  "card_type": "VISA",
  "edc_address": "PBM423AP31788",
  "is_credit": "false",
  "is_off_us": "false",
  "method": "purchase",
  "msg": "Transaction Success",
  "pan": "************1234",
  "periode": "01/25",
  "plan": "00",
  "pos_address": "192.168.10.1",
  "rc": "00",
  "reference_number": "000000000001",
  "status": "success",
  "trace_number": "000001",
  "transaction_date": "2026-03-13T16:00:00.000Z",
  "trx_id": "TRX1234567890"
}
```

**Response (408 - Timeout):**
```json
{
  "error": "transaction timeout"
}
```

**Response (400 - Bad Request):**
```json
{
  "error": "invalid request body"
}
```

**Response (404 - Mapping Not Found):**
```json
{
  "error": "unknown mid/tid combination: 1234:5678"
}
```

#### GET /api/v1/transaction/status/{trx_id}
Polling status transaksi.

**Response (200):**
```json
{
  "status": "PENDING",
  "data": null
}
```

### Admin Endpoints

#### POST /api/v1/admin/mapping
Tambah/update MID/TID mapping (hot-reload, tanpa restart).

**Request:**
```json
{
  "mid": "1999115921",
  "tid": "10747684",
  "serial_number": "PBM423AP31788"
}
```

**Response (200):**
```json
{
  "message": "mapping created/updated successfully"
}
```

#### DELETE /api/v1/admin/mapping?mid={mid}&tid={tid}
Hapus MID/TID mapping.

**Response (200):**
```json
{
  "message": "mapping deleted successfully"
}
```

#### POST /api/v1/admin/migrate
Migrate mapping dari env var `MIDTID_MAPPINGS` ke Redis.

**Request:**
```json
{
  "force": true
}
```

**Response (200):**
```json
{
  "message": "migration completed: 3 successful, 0 errors",
  "success_count": 3,
  "error_count": 0,
  "errors": []
}
```

#### GET /api/v1/admin/transaction/{trx_id}/ttl
Cek sisa TTL transaction di Redis.

#### POST /api/v1/admin/transaction/{trx_id}/extend-ttl
Extend TTL transaction.

### Health Check

#### GET /health
**Response (200):**
```json
{
  "status": "healthy"
}
```

**Response (503):**
```json
{
  "status": "unhealthy",
  "error": "Redis unavailable"
}
```

### CORS

Semua endpoint mendukung CORS:
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Authorization, X-Requested-With`

## Deployment

### Option 1: Docker Compose (Recommended)

```bash
# 1. Copy dan edit environment file
cp .env.example .env
# Edit .env, isi ABLY_API_KEY

# 2. Start services
docker compose up -d

# 3. Migrate initial mappings (optional)
curl -X POST http://localhost:8080/api/v1/admin/migrate \
  -H "Content-Type: application/json" \
  -d '{"force": true}'

# 4. Verify
curl http://localhost:8080/health
```

### Option 2: Binary + External Redis

```bash
# 1. Build binary
go build -o payment-middleware .

# 2. Set environment variables
export ABLY_API_KEY="your_key_here"
export ENCRYPTION_SECRET="ECR2022secretKey"
export REDIS_HOST="your-redis-host"
export REDIS_PORT=6379

# 3. Run
./payment-middleware

# 4. Migrate initial mappings (optional)
curl -X POST http://localhost:8080/api/v1/admin/migrate \
  -H "Content-Type: application/json" \
  -d '{"force": true}'
```

### Option 3: Systemd Service

```ini
# /etc/systemd/system/payment-middleware.service
[Unit]
Description=Payment Middleware
After=network.target redis.service

[Service]
Type=simple
User=middleware
WorkingDirectory=/opt/payment-middleware
EnvironmentFile=/opt/payment-middleware/.env
ExecStart=/opt/payment-middleware/payment-middleware
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable payment-middleware
sudo systemctl start payment-middleware
```

## Post-Deployment Checklist

1. ✅ Verify health: `curl http://<host>:8080/health`
2. ✅ Migrate mappings: `POST /api/v1/admin/migrate`
3. ✅ Verify mapping: `redis-cli KEYS "mapping:mid:*"`
4. ✅ Test transaction: kirim test transaction dari POS
5. ✅ Check logs: pastikan tidak ada error
6. ✅ Verify CORS: test preflight dari browser/frontend

## Monitoring

### Health Check
- Endpoint: `GET /health`
- Interval: 30 detik
- Checks: Redis connectivity

### Key Metrics to Monitor
- HTTP response times (terutama POST /api/v1/transaction)
- Transaction timeout rate
- Redis connection status
- Ably connection status (check logs)

### Logs
- Middleware menulis log ke stdout
- Format: `YYYY/MM/DD HH:MM:SS <message>`
- Error logs include: trx_id, mid, tid, error message

## Security Notes

- `ABLY_API_KEY` dan `ENCRYPTION_SECRET` adalah secrets, jangan commit ke git
- CORS saat ini `Allow-Origin: *`, untuk production pertimbangkan restrict ke domain tertentu
- Admin endpoints (`/api/v1/admin/*`) tidak memiliki authentication, pertimbangkan menambahkan auth atau restrict via network policy
- Encryption menggunakan AES-128-ECB (sesuai requirement EDC device)

## Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| `transaction timeout` | EDC tidak merespon dalam 60s | Cek koneksi internet EDC, cek Ably connection |
| `unknown mid/tid combination` | Mapping belum ada di Redis | Tambah via `POST /api/v1/admin/mapping` |
| `Redis unavailable` | Redis tidak bisa diakses | Cek Redis service, cek host/port/password |
| `failed to initialize Ably` | API key salah | Cek `ABLY_API_KEY` |
| EDC decryption error | Encryption secret berbeda | Pastikan `ENCRYPTION_SECRET` sama di middleware dan EDC |
