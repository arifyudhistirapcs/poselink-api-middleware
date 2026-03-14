#!/usr/bin/env python3

"""
Script untuk enkripsi transaction payload sesuai dengan standar ECR Link
Menggunakan AES-128-ECB dengan SHA-1 key derivation

Usage: python3 encrypt_transaction.py
"""

import json
import hashlib
import base64
from Crypto.Cipher import AES
from Crypto.Util.Padding import pad, unpad
from datetime import datetime

# Configuration
ENCRYPTION_SECRET = 'ECR2022secretKey'


def derive_key(secret):
    """
    Derive AES key dari secret menggunakan SHA-1
    Ambil 16 bytes pertama (32 hex chars)
    
    Args:
        secret (str): Secret key
        
    Returns:
        bytes: AES key (16 bytes)
    """
    # SHA-1 hash
    sha1_hash = hashlib.sha1(secret.encode('utf-8')).hexdigest()
    
    # Ambil 32 hex chars pertama (16 bytes)
    key_hex = sha1_hash[:32]
    
    # Convert hex to bytes
    key_bytes = bytes.fromhex(key_hex)
    
    return key_bytes, sha1_hash, key_hex


def encrypt(data, secret=ENCRYPTION_SECRET):
    """
    Enkripsi data menggunakan AES-128-ECB
    
    Args:
        data (str): Data yang akan dienkripsi
        secret (str): Secret key
        
    Returns:
        str: Base64 encoded encrypted data
    """
    try:
        # Derive key
        key_bytes, sha1_hash, key_hex = derive_key(secret)
        
        print('Key derivation:')
        print(f'  Secret: {secret}')
        print(f'  SHA-1 hash: {sha1_hash}')
        print(f'  AES Key (hex): {key_hex}')
        print(f'  Key length: {len(key_bytes)} bytes\n')
        
        # Create cipher (ECB mode)
        cipher = AES.new(key_bytes, AES.MODE_ECB)
        
        # Pad data to block size (16 bytes for AES)
        padded_data = pad(data.encode('utf-8'), AES.block_size)
        
        # Encrypt
        encrypted = cipher.encrypt(padded_data)
        
        # Base64 encode
        encrypted_b64 = base64.b64encode(encrypted).decode('utf-8')
        
        return encrypted_b64
        
    except Exception as e:
        print(f'Encryption error: {e}')
        raise


def decrypt(encrypted_data, secret=ENCRYPTION_SECRET):
    """
    Dekripsi data untuk verifikasi
    
    Args:
        encrypted_data (str): Base64 encoded encrypted data
        secret (str): Secret key
        
    Returns:
        str: Decrypted plaintext
    """
    try:
        # Derive key
        key_bytes, _, _ = derive_key(secret)
        
        # Create cipher
        cipher = AES.new(key_bytes, AES.MODE_ECB)
        
        # Base64 decode
        encrypted = base64.b64decode(encrypted_data)
        
        # Decrypt
        decrypted_padded = cipher.decrypt(encrypted)
        
        # Unpad
        decrypted = unpad(decrypted_padded, AES.block_size).decode('utf-8')
        
        return decrypted
        
    except Exception as e:
        print(f'Decryption error: {e}')
        raise


def main():
    print('=== ECR Link Transaction Encryption ===\n')
    
    # Example transaction payload
    transaction = {
        "amount": "100000",
        "action": "Sale",
        "trx_id": f"TRX{int(datetime.now().timestamp() * 1000)}",
        "pos_address": "192.168.10.1",
        "time_stamp": datetime.now().isoformat() + 'Z',
        "method": "purchase"
    }
    
    print('Original Transaction:')
    print(json.dumps(transaction, indent=2))
    print()
    
    # Convert to JSON string
    json_string = json.dumps(transaction, separators=(',', ':'))
    print(f'JSON String: {json_string}')
    print(f'JSON Length: {len(json_string)} chars\n')
    
    # Encrypt
    print('Encrypting...\n')
    encrypted = encrypt(json_string)
    
    print('Encrypted Token:')
    print(encrypted)
    print(f'\nToken Length: {len(encrypted)} chars\n')
    
    # Verify by decrypting
    print('Verifying decryption...')
    decrypted = decrypt(encrypted)
    print(f'Decrypted: {decrypted}')
    print(f'\nVerification: {"✓ SUCCESS" if decrypted == json_string else "✗ FAILED"}')
    
    print('\n=== Payload untuk Middleware API ===')
    payload = {
        "token": encrypted,
        "mid": "1999115921",
        "tid": "10747684",
        "trx_id": transaction["trx_id"]
    }
    print(json.dumps(payload, indent=2))
    
    print('\n=== cURL Command ===')
    print(f'curl -X POST http://localhost:8080/api/v1/transaction \\')
    print(f'  -H "Content-Type: application/json" \\')
    print(f'  -d \'{json.dumps(payload)}\'')


if __name__ == '__main__':
    main()
