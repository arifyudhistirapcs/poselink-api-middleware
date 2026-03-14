#!/usr/bin/env node

/**
 * Script untuk enkripsi transaction payload sesuai dengan standar ECR Link
 * Menggunakan AES-128-ECB dengan SHA-1 key derivation
 * 
 * Usage: node encrypt_transaction.js
 */

const crypto = require('crypto');

// Configuration
const ENCRYPTION_SECRET = 'ECR2022secretKey';

/**
 * Enkripsi data menggunakan AES-128-ECB
 * Key derivation: SHA-1 hash, ambil 16 bytes pertama (32 hex chars)
 * 
 * @param {string} data - Data yang akan dienkripsi (JSON string)
 * @returns {string} Base64 encoded encrypted data
 */
function encrypt(data) {
    try {
        // SHA-1 hash the secret
        const sha1Hash = crypto.createHash('sha1')
            .update(ENCRYPTION_SECRET, 'utf8')
            .digest('hex');
        
        // Take first 32 hex chars (16 bytes) for AES-128
        const keyHex = sha1Hash.substring(0, 32);
        
        // Convert hex to buffer
        const keyBuffer = Buffer.from(keyHex, 'hex');
        
        console.log('Key derivation:');
        console.log('  Secret:', ENCRYPTION_SECRET);
        console.log('  SHA-1 hash:', sha1Hash);
        console.log('  AES Key (hex):', keyHex);
        console.log('  Key length:', keyBuffer.length, 'bytes\n');
        
        // Create cipher (ECB mode - no IV needed)
        const cipher = crypto.createCipheriv('aes-128-ecb', keyBuffer, null);
        
        // Encrypt data
        let encrypted = cipher.update(data, 'utf8', 'base64');
        encrypted += cipher.final('base64');
        
        return encrypted;
    } catch (error) {
        console.error('Encryption error:', error);
        throw error;
    }
}

/**
 * Dekripsi data untuk verifikasi
 * 
 * @param {string} encryptedData - Base64 encoded encrypted data
 * @returns {string} Decrypted plaintext
 */
function decrypt(encryptedData) {
    try {
        // SHA-1 hash the secret
        const sha1Hash = crypto.createHash('sha1')
            .update(ENCRYPTION_SECRET, 'utf8')
            .digest('hex');
        
        // Take first 32 hex chars (16 bytes)
        const keyHex = sha1Hash.substring(0, 32);
        const keyBuffer = Buffer.from(keyHex, 'hex');
        
        // Create decipher
        const decipher = crypto.createDecipheriv('aes-128-ecb', keyBuffer, null);
        
        // Decrypt data
        let decrypted = decipher.update(encryptedData, 'base64', 'utf8');
        decrypted += decipher.final('utf8');
        
        return decrypted;
    } catch (error) {
        console.error('Decryption error:', error);
        throw error;
    }
}

// Example usage
function main() {
    console.log('=== ECR Link Transaction Encryption ===\n');
    
    // Example transaction payload
    const transaction = {
        amount: "100000",
        action: "Sale",
        trx_id: "TRX" + Date.now(),
        pos_address: "192.168.10.1",
        time_stamp: new Date().toISOString(),
        method: "purchase"
    };
    
    console.log('Original Transaction:');
    console.log(JSON.stringify(transaction, null, 2));
    console.log();
    
    // Convert to JSON string
    const jsonString = JSON.stringify(transaction);
    console.log('JSON String:', jsonString);
    console.log('JSON Length:', jsonString.length, 'chars\n');
    
    // Encrypt
    console.log('Encrypting...\n');
    const encrypted = encrypt(jsonString);
    
    console.log('Encrypted Token:');
    console.log(encrypted);
    console.log('\nToken Length:', encrypted.length, 'chars\n');
    
    // Verify by decrypting
    console.log('Verifying decryption...');
    const decrypted = decrypt(encrypted);
    console.log('Decrypted:', decrypted);
    console.log('\nVerification:', decrypted === jsonString ? '✓ SUCCESS' : '✗ FAILED');
    
    console.log('\n=== Payload untuk Middleware API ===');
    console.log(JSON.stringify({
        token: encrypted,
        mid: "1999115921",
        tid: "10747684",
        trx_id: transaction.trx_id
    }, null, 2));
}

// Run if called directly
if (require.main === module) {
    main();
}

// Export functions for use in other scripts
module.exports = { encrypt, decrypt };
