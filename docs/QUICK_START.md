# Quick Start - Payment Middleware

## 🚀 Jalankan Server (3 Langkah)

### 1. Set Environment Variables
```bash
export ABLY_API_KEY="jKHFtA.3mx-Zw:njUj9PK5NZOliwWa5SDsx9aBlaI6dFXwKU0zDB_dfJA"
export MIDTID_MAPPINGS='{"M001:T001":"SN12345","M002:T002":"SN67890"}'
```

### 2. Run Server
```bash
go run main.go
```

### 3. Test
```bash
./test_ably_connection.sh
```

## 📡 Channel Ably yang Digunakan

| Arah | Channel | Event | Data |
|------|---------|-------|------|
| Middleware → EDC | `edc:[serial_number]` | `payment_request` | Token (string) |
| EDC → Middleware | `response:*` | `payment_result` | EDCResponse (JSON) |

## 🧪 Test Manual

### Kirim Transaction
```bash
curl -X POST http://localhost:8080/api/v1/transaction \
  -H "Content-Type: application/json" \
  -d '{
    "token": "test_token_123",
    "mid": "M001",
    "tid": "T001",
    "trx_id": "TRX-001"
  }'
```

### Cek Status
```bash
curl http://localhost:8080/api/v1/transaction/status/TRX-001
```

## 👀 Monitor di Ably Dashboard

1. Buka: https://ably.com/accounts
2. Pilih app Anda
3. Klik tab "Dev Console"
4. Subscribe ke channel: `edc:SN12345`
5. Kirim transaction request
6. Lihat message muncul di dashboard ✅

## 🔄 Simulasi EDC Response

Di Ably Dev Console:
1. Publish to channel: `response:test`
2. Event: `payment_result`
3. Data:
```json
{
  "trx_id": "TRX-001",
  "status": "success",
  "approval": "123456",
  "amount": "100000",
  "card_name": "VISA"
}
```

## ✅ Checklist Koneksi Ably

- [ ] Server running tanpa error
- [ ] Log menunjukkan "Ably client initialized"
- [ ] Log menunjukkan "Subscribed to EDC response channels"
- [ ] Message muncul di Ably Dashboard saat hit API
- [ ] Response dari EDC diterima oleh middleware

## 🔧 Konfigurasi Ably yang Perlu Dicek

### Di Ably Dashboard → API Keys

Pastikan API key punya permission:
- ✅ Publish
- ✅ Subscribe
- ✅ Presence (optional)

### Channel yang Perlu Dikonfigurasi (Optional)

Untuk production, set channel rules:
- `edc:*` - Allow publish dari middleware
- `response:*` - Allow subscribe dari middleware

## 📝 Mapping MID/TID ke Serial Number

Format: `"MID:TID":"SerialNumber"`

Contoh:
```json
{
  "M001:T001": "SN12345",
  "M002:T002": "SN67890",
  "M003:T003": "SN11111"
}
```

Cara update:
```bash
export MIDTID_MAPPINGS='{"M001:T001":"SN12345","M002:T002":"SN67890"}'
```

## 🐛 Troubleshooting Cepat

### Error: "Failed to initialize Ably client"
→ Cek API key dan koneksi internet

### Message tidak muncul di Ably
→ Cek MID/TID mapping dan channel name

### Response tidak diterima
→ Pastikan channel format `response:*` dan event `payment_result`

## 📚 Dokumentasi Lengkap

- Setup detail: `ABLY_SETUP.md`
- Cara test lengkap: `CARA_TEST_ABLY.md`
- API documentation: `README.md`
