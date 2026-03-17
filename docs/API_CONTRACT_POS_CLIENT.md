# POSeLink API - Dokumentasi untuk POS Client

## Base URL

| Environment | URL |
|-------------|-----|
| Development | `https://development-ecrlink.pcsindonesia.com` |

---

## Endpoints

### 1. Kirim Transaksi

Mengirim transaksi pembayaran ke EDC device melalui middleware.

**Endpoint:** `POST /api/v1/transaction`

**Content-Type:** `application/json`

#### Request Body

```json
{
  "token": "string (wajib) - Data transaksi terenkripsi dalam format Base64",
  "mid": "string (wajib) - Merchant ID",
  "tid": "string (wajib) - Terminal ID",
  "trx_id": "string (wajib) - ID transaksi unik"
}
```

#### Struktur Data Transaksi (sebelum enkripsi)

```json
{
  "amount": "string - Nominal transaksi (contoh: 25000)",
  "action": "string - Jenis transaksi (Sale, Void, Refund, Settlement)",
  "trx_id": "string - ID transaksi unik (harus sama dengan trx_id di luar token)",
  "pos_address": "string - IP address atau identifier POS",
  "time_stamp": "string - Format: yyyy-MM-dd HH:mm:ss (contoh: 2026-03-16 14:30:00)",
  "method": "string - Metode pembayaran (purchase, qris, refund)"
}
```

#### Jenis Action

| Action | Keterangan |
|--------|------------|
| `Sale` | Transaksi pembelian |
| `Void` | Pembatalan transaksi |
| `Refund` | Pengembalian dana |
| `Settlement` | Settlement batch |

#### Jenis Method

| Method | Keterangan |
|--------|------------|
| `purchase` | Pembayaran kartu (debit/kredit) |
| `qris` | Pembayaran QRIS |
| `refund` | Pengembalian dana |

#### Contoh Request

```bash
curl -X POST https://development-ecrlink.pcsindonesia.com/api/v1/transaction \
  -H "Content-Type: application/json" \
  -d '{
    "token": "0Cf2RAWDHMYo9F+epuJktk/0yE/Ztc3MtzWJm0KrilI...",
    "mid": "1999115921",
    "tid": "10747684",
    "trx_id": "TRX1773646062000"
  }'
```

#### Response Sukses

**Status Code:** `200 OK`

```json
{
  "acq_mid": "000001999115921",
  "acq_tid": "10747684",
  "action": "Sale",
  "amount": "25000",
  "approval": "123456",
  "batch_number": "000001",
  "card_category": "DEBIT",
  "card_name": "BRI",
  "card_type": "MASTERCARD",
  "edc_address": "PBM423AP31788",
  "is_credit": "false",
  "is_off_us": "false",
  "method": "purchase",
  "msg": "Approved",
  "pan": "****1234",
  "periode": "N/A",
  "plan": "N/A",
  "pos_address": "192.168.10.1",
  "rc": "00",
  "reference_number": "000000012345",
  "status": "success",
  "trace_number": "000015",
  "transaction_date": "2026-03-16 14:30:00",
  "trx_id": "TRX1773646062000"
}
```

#### Response Gagal

```json
{
  "acq_mid": "000001999115921",
  "acq_tid": "10747684",
  "action": "Sale",
  "amount": "25000",
  "status": "failed",
  "msg": "Transaction Failed",
  "rc": "05",
  "trx_id": "TRX1773646062000",
  "...": "..."
}
```

#### Response Error

| Status Code | Body | Keterangan |
|-------------|------|------------|
| `400` | `{"error": "unknown mid/tid combination"}` | MID/TID belum terdaftar |
| `400` | `{"error": "failed to decrypt token"}` | Token tidak valid / enkripsi salah |
| `400` | `{"error": "missing required field: token"}` | Field wajib tidak diisi |
| `504` | `{"error": "transaction timeout"}` | EDC tidak merespon dalam 60 detik |

#### Response Code (rc)

| Code | Keterangan |
|------|------------|
| `00` | Sukses / Approved |
| `05` | Do not honor |
| `12` | Invalid transaction |
| `13` | Invalid amount |
| `51` | Saldo tidak cukup |
| `54` | Kartu expired |
| `55` | PIN salah |
| `58` | Transaksi tidak diizinkan |
| `91` | Issuer tidak tersedia |
| `96` | System malfunction |

---

### 2. Cek Status Transaksi

Mengecek status transaksi yang sudah dikirim sebelumnya.

**Endpoint:** `GET /api/v1/transaction/status/{trx_id}`

#### Contoh Request

```bash
curl https://development-ecrlink.pcsindonesia.com/api/v1/transaction/status/TRX1773646062000
```

---

### 3. Health Check

Mengecek apakah server middleware aktif.

**Endpoint:** `GET /health`

#### Contoh Request

```bash
curl https://development-ecrlink.pcsindonesia.com/health
```

#### Response

```json
{
  "status": "healthy"
}
```

---

## Enkripsi Token

Token dienkripsi menggunakan **AES-128-ECB** dengan key derivation dari **SHA-1**.

### Langkah Enkripsi

1. Hash secret key `ECR2022secretKey` menggunakan SHA-1
2. Ambil 32 karakter hex pertama (16 bytes) sebagai AES key
3. Enkripsi JSON string menggunakan AES-128-ECB dengan PKCS5 padding
4. Encode hasil ke Base64

### Implementasi Node.js

```javascript
const crypto = require('crypto');
const ENCRYPTION_SECRET = 'ECR2022secretKey';

function encrypt(data) {
  const sha1Hash = crypto.createHash('sha1').update(ENCRYPTION_SECRET, 'utf8').digest('hex');
  const keyBuffer = Buffer.from(sha1Hash.substring(0, 32), 'hex');
  const cipher = crypto.createCipheriv('aes-128-ecb', keyBuffer, null);
  let encrypted = cipher.update(JSON.stringify(data), 'utf8', 'base64');
  encrypted += cipher.final('base64');
  return encrypted;
}

// Contoh penggunaan
const transaction = {
  amount: "25000",
  action: "Sale",
  trx_id: "TRX123456",
  pos_address: "192.168.10.1",
  time_stamp: "2026-03-16 14:30:00",
  method: "purchase"
};

const token = encrypt(transaction);
```

### Implementasi Python

```python
import hashlib, json, base64
from Crypto.Cipher import AES
from Crypto.Util.Padding import pad

ENCRYPTION_SECRET = 'ECR2022secretKey'

def encrypt(data):
    sha1_hash = hashlib.sha1(ENCRYPTION_SECRET.encode('utf-8')).hexdigest()
    key = bytes.fromhex(sha1_hash[:32])
    cipher = AES.new(key, AES.MODE_ECB)
    padded = pad(json.dumps(data).encode('utf-8'), AES.block_size)
    return base64.b64encode(cipher.encrypt(padded)).decode('utf-8')
```

### Implementasi Java/Kotlin

```kotlin
import java.security.MessageDigest
import javax.crypto.Cipher
import javax.crypto.spec.SecretKeySpec
import android.util.Base64

fun encrypt(data: String, secret: String = "ECR2022secretKey"): String {
    val sha1 = MessageDigest.getInstance("SHA-1")
        .digest(secret.toByteArray(Charsets.UTF_8))
        .joinToString("") { "%02x".format(it) }
    val keyBytes = sha1.substring(0, 32).chunked(2)
        .map { it.toInt(16).toByte() }.toByteArray()
    val cipher = Cipher.getInstance("AES/ECB/PKCS5Padding")
    cipher.init(Cipher.ENCRYPT_MODE, SecretKeySpec(keyBytes, "AES"))
    return Base64.encodeToString(cipher.doFinal(data.toByteArray()), Base64.NO_WRAP)
}
```

---

## Flow Transaksi

```
POS Client                    Middleware                     EDC Device
    |                             |                              |
    |  POST /api/v1/transaction   |                              |
    |---------------------------->|                              |
    |                             |  Publish via Ably            |
    |                             |----------------------------->|
    |                             |                              |
    |                             |  (EDC proses pembayaran)     |
    |                             |                              |
    |                             |  Response via Ably           |
    |                             |<-----------------------------|
    |  200 OK (JSON response)     |                              |
    |<----------------------------|                              |
```

---

## Catatan Penting

- Timeout default: **60 detik**. Jika EDC tidak merespon dalam waktu tersebut, middleware akan mengembalikan error `504 Gateway Timeout`.
- `trx_id` harus unik untuk setiap transaksi. Disarankan menggunakan format `TRX{unix_timestamp}000`.
- Format `time_stamp` harus `yyyy-MM-dd HH:mm:ss` (contoh: `2026-03-16 14:30:00`). Format ISO 8601 dengan `T` dan `Z` tidak didukung.
- MID/TID harus sudah terdaftar di middleware sebelum bisa mengirim transaksi.

---

**Version:** 1.0.0
**Last Updated:** 16 Maret 2026
