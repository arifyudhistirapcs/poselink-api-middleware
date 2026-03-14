# Payment Middleware API Contract

## Base URL
```
http://localhost:8080
```

## Authentication
No authentication required for localhost deployment.

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
  "amount": "string - Transaction amount in smallest currency unit (e.g., 100000 for Rp 1,000.00)",
  "action": "string - Transaction type (Sale, Void, Refund, Settlement, etc.)",
  "trx_id": "string - Unique transaction ID (must match outer trx_id)",
  "pos_address": "string - POS IP address or identifier",
  "time_stamp": "string - ISO 8601 timestamp (e.g., 2026-03-13T07:14:46.000Z)",
  "method": "string - Payment method (purchase, refund, etc.)"
}
```

#### Example Request

```bash
curl -X POST http://localhost:8080/api/v1/transaction \
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

**Response Body:**
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

**Response Body:**
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

## Error Handling

### Common Error Scenarios

1. **Invalid MID/TID Mapping**
   - Status: `400 Bad Request`
   - Response: `{"error": "MID/TID mapping not found"}`

2. **Decryption Failed**
   - Status: `400 Bad Request`
   - Response: `{"error": "failed to decrypt token"}`

3. **Transaction Timeout**
   - Status: `504 Gateway Timeout`
   - Response: `{"error": "transaction timeout"}`
   - Default timeout: 60 seconds

4. **EDC Device Offline**
   - Status: `500 Internal Server Error`
   - Response: `{"error": "EDC device not connected"}`

5. **User Cancelled**
   - Status: `200 OK`
   - Response: `{"status": "failed", "rc": "XX", "msg": "Transaction cancelled by user"}`

---

## Encryption Implementation

### Node.js Example

```javascript
const crypto = require('crypto');

const ENCRYPTION_SECRET = 'ECR2022secretKey';

function encryptTransaction(transaction) {
  // SHA-1 hash and take first 16 bytes
  const sha1Hash = crypto.createHash('sha1')
    .update(ENCRYPTION_SECRET, 'utf8')
    .digest('hex');
  const keyHex = sha1Hash.substring(0, 32);
  const keyBuffer = Buffer.from(keyHex, 'hex');
  
  // AES-128-ECB encryption
  const cipher = crypto.createCipheriv('aes-128-ecb', keyBuffer, null);
  let encrypted = cipher.update(JSON.stringify(transaction), 'utf8', 'base64');
  encrypted += cipher.final('base64');
  
  return encrypted;
}

// Usage
const transaction = {
  amount: "100000",
  action: "Sale",
  trx_id: "TRX1234567890",
  pos_address: "192.168.10.1",
  time_stamp: new Date().toISOString(),
  method: "purchase"
};

const token = encryptTransaction(transaction);
```

### Python Example

```python
import hashlib
import json
from Crypto.Cipher import AES
from Crypto.Util.Padding import pad
import base64

ENCRYPTION_SECRET = 'ECR2022secretKey'

def encrypt_transaction(transaction):
    # SHA-1 hash and take first 16 bytes
    sha1_hash = hashlib.sha1(ENCRYPTION_SECRET.encode('utf-8')).hexdigest()
    key_hex = sha1_hash[:32]
    key = bytes.fromhex(key_hex)
    
    # AES-128-ECB encryption
    cipher = AES.new(key, AES.MODE_ECB)
    plaintext = json.dumps(transaction).encode('utf-8')
    padded = pad(plaintext, AES.block_size)
    encrypted = cipher.encrypt(padded)
    
    return base64.b64encode(encrypted).decode('utf-8')

# Usage
transaction = {
    "amount": "100000",
    "action": "Sale",
    "trx_id": "TRX1234567890",
    "pos_address": "192.168.10.1",
    "time_stamp": "2026-03-13T07:14:46.000Z",
    "method": "purchase"
}

token = encrypt_transaction(transaction)
```

---

## Configuration

### MID/TID Mapping

The middleware uses MID/TID pairs to route transactions to the correct EDC device.

**Format:** `"MID:TID" -> "EDC_SERIAL_NUMBER"`

**Example:**
```json
{
  "1999115921:10747684": "PBM423AP31788",
  "M001:T001": "SN12345",
  "M002:T002": "SN67890"
}
```

Contact system administrator to register new MID/TID mappings.

---

## Testing

### Using cURL

```bash
# 1. Create transaction JSON
TRANSACTION='{"amount":"100000","action":"Sale","trx_id":"TRX123","pos_address":"192.168.10.1","time_stamp":"2026-03-13T07:14:46.000Z","method":"purchase"}'

# 2. Encrypt (requires Node.js)
TOKEN=$(node -e "
const crypto = require('crypto');
const secret = 'ECR2022secretKey';
const sha1 = crypto.createHash('sha1').update(secret, 'utf8').digest('hex');
const key = Buffer.from(sha1.substring(0, 32), 'hex');
const cipher = crypto.createCipheriv('aes-128-ecb', key, null);
let enc = cipher.update('$TRANSACTION', 'utf8', 'base64');
enc += cipher.final('base64');
console.log(enc);
")

# 3. Send request
curl -X POST http://localhost:8080/api/v1/transaction \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"$TOKEN\",\"mid\":\"1999115921\",\"tid\":\"10747684\",\"trx_id\":\"TRX123\"}"
```

### Using Provided Script

```bash
./send_to_middleware.sh [amount] [action]

# Examples:
./send_to_middleware.sh 100000 Sale
./send_to_middleware.sh 50000 Refund
```

---

## Support

For issues or questions, contact the development team.

**Version:** 1.0.0  
**Last Updated:** March 13, 2026
