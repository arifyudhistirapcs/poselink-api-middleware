#!/usr/bin/env node

/**
 * Test script to send a test transaction to the EDC device via Ably
 * 
 * Usage: node test_ably_connection.js
 */

const Ably = require('ably');
const crypto = require('crypto');

// Configuration
const ABLY_API_KEY = 'jKHFtA.3mx-Zw:njUj9PK5NZOliwWa5SDsx9aBlaI6dFXwKU0zDB_dfJA';
const ENCRYPTION_SECRET = 'ECR2022secretKey';
const DEVICE_SERIAL = 'PBM423AP31788';
const CHANNEL_NAME = `edc:${DEVICE_SERIAL}`;

// AES-ECB Encryption function (matching ECR Link standard)
function encrypt(data) {
    try {
        const crypto = require('crypto');
        
        // SHA-1 hash the secret
        const sha1Hash = crypto.createHash('sha1').update(ENCRYPTION_SECRET, 'utf8').digest('hex');
        
        // Take first 32 hex chars (16 bytes)
        const keyHex = sha1Hash.substring(0, 32);
        
        // Convert hex to buffer
        const keyBuffer = Buffer.from(keyHex, 'hex');
        
        console.log('Key derivation - SHA-1 hex (first 32 chars):', keyHex);
        
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

// Create test transaction
function createTestTransaction() {
    const transaction = {
        amount: "50000",
        action: "Sale",
        trx_id: `TEST${Date.now()}`,
        pos_address: "192.168.10.1",
        time_stamp: new Date().toISOString(),
        method: "purchase"
    };
    
    return transaction;
}

// Main function
async function main() {
    console.log('=== Ably Connection Test ===\n');
    console.log(`API Key: ${ABLY_API_KEY.substring(0, 20)}...`);
    console.log(`Device Serial: ${DEVICE_SERIAL}`);
    console.log(`Channel: ${CHANNEL_NAME}\n`);
    
    // Initialize Ably
    console.log('Connecting to Ably...');
    const ably = new Ably.Realtime(ABLY_API_KEY);
    
    // Wait for connection
    await new Promise((resolve, reject) => {
        ably.connection.on('connected', () => {
            console.log('✓ Connected to Ably\n');
            resolve();
        });
        
        ably.connection.on('failed', (error) => {
            console.error('✗ Failed to connect to Ably:', error);
            reject(error);
        });
        
        // Timeout after 10 seconds
        setTimeout(() => {
            reject(new Error('Connection timeout'));
        }, 10000);
    });
    
    // Get channel
    const channel = ably.channels.get(CHANNEL_NAME);
    
    // Subscribe to response channel to see if device responds
    console.log('Subscribing to response channel...');
    const responseChannel = ably.channels.get('response:*');
    
    responseChannel.subscribe('payment_result', (message) => {
        console.log('\n✓ Received response from device!');
        console.log('Response data:', message.data);
        
        // Try to decrypt if it's encrypted
        try {
            // Note: Decryption would need to match the encryption logic
            console.log('(Response is encrypted, device processed the transaction)');
        } catch (e) {
            // Ignore decryption errors for now
        }
    });
    
    // Create and encrypt test transaction
    console.log('Creating test transaction...');
    const transaction = createTestTransaction();
    console.log('Transaction:', JSON.stringify(transaction, null, 2));
    
    console.log('\nEncrypting transaction...');
    const transactionJson = JSON.stringify(transaction);
    const encryptedToken = encrypt(transactionJson);
    console.log('Encrypted token (first 50 chars):', encryptedToken.substring(0, 50) + '...');
    
    // Publish to channel
    console.log(`\nPublishing to channel: ${CHANNEL_NAME}`);
    await channel.publish('payment_request', encryptedToken);
    console.log('✓ Test transaction sent!\n');
    
    console.log('Waiting for response from device (10 seconds)...');
    
    // Wait for response
    await new Promise(resolve => setTimeout(resolve, 10000));
    
    console.log('\nTest completed. Closing connection...');
    ably.close();
    
    console.log('\n=== Test Summary ===');
    console.log('1. Connected to Ably: ✓');
    console.log('2. Sent test transaction: ✓');
    console.log('3. Check your EDC device screen for the transaction');
    console.log('4. Expected channel: ' + CHANNEL_NAME);
    console.log('\nIf the device received the transaction, you should see:');
    console.log('- Amount: Rp 50,000');
    console.log('- Action: Sale');
    console.log('- Transaction ID: TEST...');
}

// Run the test
main().catch(error => {
    console.error('\n✗ Test failed:', error.message);
    process.exit(1);
});
