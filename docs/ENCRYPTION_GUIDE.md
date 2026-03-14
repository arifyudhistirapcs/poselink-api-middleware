# ECR Link Encryption Guide

## Overview

EDC Sunmi menggunakan enkripsi **AES-128-ECB** dengan **SHA-1 key derivation** untuk mengamankan payload transaction.

## Algoritma Enkripsi

```
1. SHA-1 hash secret key
2. Ambil 16 bytes pertama (32 hex chars) sebagai AES-128 key
3. Encrypt payload dengan AES-128-ECB/PKCS5Padding
4. Encode hasil ke Base64
```

## Secret Key

- **Default**: `ECR2022secretKey`
- Dapat dikonfigurasi di `edc/app/src/main/assets/config.properties`

## Key Derivation

```javascript
// SHA-1 hash
const sha1Hash = crypto.createHash('sha1')
    .update('ECR2022secretKey', 'utf8')
    .digest('hex');

// Result: 1453f7df555d136f305d33519f66b473fdffe780

// Take first 32 hex chars (16 bytes)
const keyHex = sha1Hash.substring(0, 32);
// Result: 1453f7df555d136f305d33519f66b473
```

## Scripts Enkripsi

### 1. Node.js Script

**File**: `encrypt_transaction.js`

```bash
# Install dependencies (jika belum)
npm install

# Run script
node encrypt_transaction.js
```

**Output**:
```
=== ECR Link Transaction Encryption ===

Original Transaction:
{
  "amount": "100000",
  "action": "Sale",
  "trx_id": "TRX1773384828346",
  ...
}

Encrypted Token:
mDVk3G/jqHXtlMaJvrtNHSyOEUDJLO8e3d3xM0UZukH...

=== Payload untuk Middleware API ===
{
  "token": "mDVk3G/jqHXtlMaJvrtNH...",
  "mid": "1999115921",
  "tid": "10747684",
  "trx_id": "TRX1773384828346"
}
```

### 2. Python Script

**File**: `encrypt_transaction.py`

```bash
# Install dependencies
pip3 install pycryptodome

# Run script
python3 encrypt_transaction.py
```

## Implementasi di Berbagai Bahasa

### JavaScript (Node.js)

```javascript
const crypto = require('crypto');

function encrypt(data, secret = 'ECR2022secretKey') {
    // SHA-1 hash
    const sha1Hash = crypto.createHash('sha1')
        .update(secret, 'utf8')
        .digest('hex');
    
    // Take first 32 hex chars (16 bytes)
    const keyHex = sha1Hash.substring(0, 32);
    const keyBuffer = Buffer.from(keyHex, 'hex');
    
    // Create cipher (ECB mode)
    const cipher = crypto.createCipheriv('aes-128-ecb', keyBuffer, null);
    
    // Encrypt
    let encrypted = cipher.update(data, 'utf8', 'base64');
    encrypted += cipher.final('base64');
    
    return encrypted;
}
```

### Python

```python
import hashlib
import base64
from Crypto.Cipher import AES
from Crypto.Util.Padding import pad

def encrypt(data, secret='ECR2022secretKey'):
    # SHA-1 hash
    sha1_hash = hashlib.sha1(secret.encode('utf-8')).hexdigest()
    
    # Take first 32 hex chars (16 bytes)
    key_hex = sha1_hash[:32]
    key_bytes = bytes.fromhex(key_hex)
    
    # Create cipher (ECB mode)
    cipher = AES.new(key_bytes, AES.MODE_ECB)
    
    # Pad and encrypt
    padded_data = pad(data.encode('utf-8'), AES.block_size)
    encrypted = cipher.encrypt(padded_data)
    
    # Base64 encode
    return base64.b64encode(encrypted).decode('utf-8')
```

### Java (Android)

```java
import javax.crypto.Cipher;
import javax.crypto.spec.SecretKeySpec;
import java.security.MessageDigest;
import java.util.Base64;

public class ECREncryption {
    public static String encrypt(String data, String secret) throws Exception {
        // SHA-1 hash
        MessageDigest sha1 = MessageDigest.getInstance("SHA-1");
        byte[] hashedBytes = sha1.digest(secret.getBytes("UTF-8"));
        
        // Convert to hex
        StringBuilder hexString = new StringBuilder();
        for (byte b : hashedBytes) {
            String hex = String.format("%02x", b);
            hexString.append(hex);
        }
        
        // Take first 32 hex chars (16 bytes)
        String keyHex = hexString.substring(0, 32);
        byte[] keyBytes = hexToBytes(keyHex);
        
        // Create AES key
        SecretKeySpec secretKey = new SecretKeySpec(keyBytes, "AES");
        
        // Encrypt with AES/ECB/PKCS5Padding
        Cipher cipher = Cipher.getInstance("AES/ECB/PKCS5Padding");
        cipher.init(Cipher.ENCRYPT_MODE, secretKey);
        byte[] encryptedBytes = cipher.doFinal(data.getBytes("UTF-8"));
        
        // Base64 encode
        return Base64.getEncoder().encodeToString(encryptedBytes);
    }
    
    private static byte[] hexToBytes(String hex) {
        int len = hex.length();
        byte[] data = new byte[len / 2];
        for (int i = 0; i < len; i += 2) {
            data[i / 2] = (byte) ((Character.digit(hex.charAt(i), 16) << 4)
                                 + Character.digit(hex.charAt(i + 1), 16));
        }
        return data;
    }
}
```

### Kotlin (Android)

```kotlin
import android.util.Base64
import java.security.MessageDigest
import javax.crypto.Cipher
import javax.crypto.spec.SecretKeySpec

class CryptoManager(private val secretKey: String) {
    
    fun encrypt(data: String): String {
        // SHA-1 hash
        val sha1 = MessageDigest.getInstance("SHA-1")
        val hashedBytes = sha1.digest(secretKey.toByteArray(Charsets.UTF_8))
        
        // Convert to hex
        val hexString = StringBuilder()
        for (b in hashedBytes) {
            val hex = String.format("%02x", b)
            hexString.append(hex)
        }
        
        // Take first 32 hex chars (16 bytes)
        val keyHex = hexString.substring(0, 32)
        val keyBytes = hexToBytes(keyHex)
        
        // Create cipher
        val key = SecretKeySpec(keyBytes, "AES")
        val cipher = Cipher.getInstance("AES/ECB/PKCS5Padding")
        cipher.init(Cipher.ENCRYPT_MODE, key)
        
        // Encrypt
        val encrypted = cipher.doFinal(data.toByteArray(Charsets.UTF_8))
        
        // Base64 encode
        return Base64.encodeToString(encrypted, Base64.NO_WRAP)
    }
    
    private fun hexToBytes(hex: String): ByteArray {
        val len = hex.length
        val data = ByteArray(len / 2)
        var i = 0
        while (i < len) {
            data[i / 2] = ((Character.digit(hex[i], 16) shl 4) + 
                          Character.digit(hex[i + 1], 16)).toByte()
            i += 2
        }
        return data
    }
}
```

## Testing

### Test dengan Script

```bash
# Node.js
node encrypt_transaction.js

# Python
python3 encrypt_transaction.py
```

### Test dengan Middleware

```bash
# Generate encrypted token
TOKEN=$(node -e "
const { encrypt } = require('./encrypt_transaction.js');
const transaction = {
    amount: '100000',
    action: 'Sale',
    trx_id: 'TEST' + Date.now(),
    pos_address: '192.168.10.1',
    time_stamp: new Date().toISOString(),
    method: 'purchase'
};
console.log(encrypt(JSON.stringify(transaction)));
")

# Send to middleware
curl -X POST http://localhost:8080/api/v1/transaction \
  -H "Content-Type: application/json" \
  -d "{
    \"token\": \"$TOKEN\",
    \"mid\": \"1999115921\",
    \"tid\": \"10747684\",
    \"trx_id\": \"TEST123\"
  }"
```

### Test dengan Ably

```bash
# Run test script
node test_ably_connection.js
```

## Verifikasi

Untuk memverifikasi enkripsi bekerja dengan benar:

1. **Key Derivation harus sama**:
   ```
   Secret: ECR2022secretKey
   SHA-1: 1453f7df555d136f305d33519f66b473fdffe780
   AES Key: 1453f7df555d136f305d33519f66b473
   ```

2. **Encrypt dan Decrypt harus menghasilkan data yang sama**:
   ```javascript
   const original = '{"amount":"100000"}';
   const encrypted = encrypt(original);
   const decrypted = decrypt(encrypted);
   console.assert(original === decrypted);
   ```

3. **EDC harus bisa decrypt token yang dikirim**:
   - Check logcat untuk melihat decryption berhasil
   - Transaction harus muncul di layar EDC

## Troubleshooting

### Token tidak bisa di-decrypt

**Penyebab**:
- Secret key berbeda
- Algoritma enkripsi berbeda
- Key derivation berbeda

**Solusi**:
1. Pastikan secret key sama: `ECR2022secretKey`
2. Pastikan menggunakan AES-128-ECB
3. Pastikan key derivation menggunakan SHA-1 (16 bytes pertama)

### EDC tidak menerima transaction

**Penyebab**:
- Ably connection gagal
- Channel name salah
- Event name salah

**Solusi**:
1. Check Ably connection status di EDC
2. Pastikan channel: `edc:{SERIAL_NUMBER}`
3. Pastikan event name: `payment_request`

## Referensi

- **EDC Source**: `edc/app/src/main/java/com/sunmi/edc/demo/utils/CryptoManager.kt`
- **Test Script**: `test_ably_connection.js`
- **Encryption Scripts**: 
  - `encrypt_transaction.js` (Node.js)
  - `encrypt_transaction.py` (Python)
