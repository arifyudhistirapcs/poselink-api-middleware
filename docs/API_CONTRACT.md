# Payment Middleware API Contract

## Base URL

| Environment | URL |
|-------------|-----|
| Development | `https://development-ecrlink.pcsindonesia.com` |
| Local | `http://localhost:8080` |

## Authentication
No authentication required for current deployment.

---

## Endpoints

### 1. Process Transaction

Process a payment transaction through EDC device.

**Endpoint:** `POST /api/v1/transaction`

**Content-Type:** `application/json`

#### Request Body

```json
{
  "token": "string (required) - Encrypted transaction data in Base64",
  "mid": "string (required) - Merchant ID",
  "tid": "string (required) - Terminal ID",
  "trx_id": "string (required) - Unique transaction ID"
}
```

#### Token Encryption

The `token` field contains encrypted transaction data using **AES-128-ECB** with **SHA-1 key derivation**.

**Encryption Steps:**
1. Create transaction JSON object
2. SHA-1 hash the secret key: `ECR2022secretKey`
3. Take first 32 hex characters (16 bytes) as AES key
4. Encrypt JSON string using AES-128-ECB with PKCS5 padding
5. Encode result to Base64

**Transaction Data Structure (before encryption):**
```json
{
  "amount": "string - Transaction amount (e.g., 25000)",
  "action": "string - Transaction type (Sale, Void, Refund, Settlement, etc.)",
  "trx_id": "string - Unique transaction ID (must match outer trx_id)",
  "pos_address": "string - POS IP address or identifier",
  "time_stamp": "string - Format: yyyy-MM-dd HH:mm:ss (e.g., 2026-03-15 11:40:35)",
  "method": "string - Payment method (purchase, refund, etc.)"
}
```

#### Example Request

```bash
curl -X POST https://development-ecrlink.pcsindonesia.com/api/v1/transaction \
  -H "Content-Type: application/json" \
  -d '{
    "token": "+37sZApCdyaa0xcZbTIyLtNGTL0+Jd27OglY7BL7uqBv2ObSRN...",
    "mid": "1999115921",
    "tid": "10747684",
    "trx_id": "TRX1773386086000"
  }'
```

#### Response (Success)

**Status Code:** `200 OK`

```json
{
  "acq_mid": "string - Acquirer Merchant ID",
  "acq_tid": "string - Acquirer Terminal ID",
  "action": "string - Transaction type",
  "amount": "string - Transaction amount",
  "approval": "string - Approval code",
  "batch_number": "string - Batch number",
  "card_category": "string - Card category (DEBIT/CREDIT)",
  "card_name": "string - Card issuer name",
  "card_type": "string - Card type (VISA/MASTERCARD/etc)",
  "edc_address": "string - EDC device serial number",
  "is_credit": "string - Is credit card (true/false)",
  "is_off_us": "string - Is off-us transaction (true/false)",
  "method": "string - Payment method",
  "msg": "string - Response message",
  "pan": "string - Masked card number",
  "periode": "string - Card expiry (MM/YY)",
  "plan": "string - Installment plan",
  "pos_address": "string - POS address",
  "rc": "string - Response code (00 = success)",
  "reference_number": "string - Reference number",
  "status": "string - Transaction status (success/failed)",
  "trace_number": "string - Trace number",
  "transaction_date": "string - Transaction timestamp",
  "trx_id": "string - Transaction ID"
}
```

#### Response (Error)

**Status Code:** `400 Bad Request` | `500 Internal Server Error` | `504 Gateway Timeout`

```json
{
  "error": "string - Error message"
}
```

#### Response Codes

| Code | Description |
|------|-------------|
| `00` | Success |
| `05` | Do not honor |
| `12` | Invalid transaction |
| `13` | Invalid amount |
| `30` | Format error |
| `51` | Insufficient funds |
| `54` | Expired card |
| `55` | Incorrect PIN |
| `58` | Transaction not permitted |
| `91` | Issuer unavailable |
| `96` | System malfunction |

---

### 2. Check Transaction Status

Check the status of a previously submitted transaction.

**Endpoint:** `GET /api/v1/transaction/status/{trx_id}`

#### Example Request

```bash
curl https://development-ecrlink.pcsindonesia.com/api/v1/transaction/status/TRX1773549700000
```

---

### 3. Health Check

**Endpoint:** `GET /health`

#### Example Request

```bash
curl https://development-ecrlink.pcsindonesia.com/health
```

---

### 4. Admin - Create/Update MID/TID Mapping

**Endpoint:** `POST /api/v1/admin/mapping`

#### Request Body

```json
{
  "mid": "string - Merchant ID",
  "tid": "string - Terminal ID",
  "serial_number": "string - EDC device serial number"
}
```

---

### 5. Admin - Delete MID/TID Mapping

**Endpoint:** `DELETE /api/v1/admin/mapping`

#### Request Body

```json
{
  "mid": "string - Merchant ID",
  "tid": "string - Terminal ID"
}
```

---

### 6. Admin - Migrate Mappings

**Endpoint:** `POST /api/v1/admin/migrate`

Migrates MID/TID mappings from environment config to Redis.

---

### 7. Admin - Get Transaction TTL

**Endpoint:** `GET /api/v1/admin/transaction/{trx_id}/ttl`

---

### 8. Admin - Extend Transaction TTL

**Endpoint:** `POST /api/v1/admin/transaction/{trx_id}/extend-ttl`

---

## Error Handling

| Scenario | Status | Response |
|----------|--------|----------|
| Invalid MID/TID | `400` | `{"error": "MID/TID mapping not found"}` |
| Decryption Failed | `400` | `{"error": "failed to decrypt token"}` |
| Transaction Timeout | `504` | `{"error": "transaction timeout"}` |
| EDC Device Offline | `500` | `{"error": "EDC device not connected"}` |
| User Cancelled | `200` | `{"status": "failed", "msg": "Transaction cancelled by user"}` |

---

## Encryption Implementation

### Node.js

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
```

### Python

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

---

## Testing Scripts

### Local Testing

```bash
# From middleware directory
./script/send_to_middleware.sh [amount] [action]
./script/send_to_middleware.sh 25000 Sale
```

### Remote Testing (Development Server)

```bash
# From middleware directory
./script/send_to_dev.sh [amount] [action]
./script/send_to_dev.sh 25000 Sale
```

---

## Configuration

### MID/TID Mapping

Format: `"MID:TID" -> "EDC_SERIAL_NUMBER"`

```json
{
  "1999115921:10747684": "PBM423AP31788"
}
```

Contact system administrator to register new MID/TID mappings.

---

**Version:** 1.1.0
**Last Updated:** March 16, 2026
