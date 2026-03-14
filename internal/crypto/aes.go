package crypto

import (
	"crypto/aes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// DecryptAES128ECB decrypts data using AES-128-ECB with SHA-1 key derivation
// This matches the encryption used by the EDC device
func DecryptAES128ECB(encryptedData, secret string) (string, error) {
	// Derive key using SHA-1
	key, err := deriveKeyFromSecret(secret)
	if err != nil {
		return "", fmt.Errorf("failed to derive key: %w", err)
	}

	// Base64 decode the encrypted data
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Check if ciphertext is a multiple of block size
	if len(ciphertext)%aes.BlockSize != 0 {
		return "", fmt.Errorf("ciphertext is not a multiple of block size")
	}

	// Decrypt using ECB mode (decrypt each block independently)
	plaintext := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += aes.BlockSize {
		block.Decrypt(plaintext[i:i+aes.BlockSize], ciphertext[i:i+aes.BlockSize])
	}

	// Remove PKCS5 padding
	plaintext, err = pkcs5Unpad(plaintext)
	if err != nil {
		return "", fmt.Errorf("failed to unpad: %w", err)
	}

	return string(plaintext), nil
}

// EncryptAES128ECB encrypts data using AES-128-ECB with SHA-1 key derivation
// This matches the encryption used by the EDC device
func EncryptAES128ECB(plaintext, secret string) (string, error) {
	// Derive key using SHA-1
	key, err := deriveKeyFromSecret(secret)
	if err != nil {
		return "", fmt.Errorf("failed to derive key: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Add PKCS5 padding
	paddedPlaintext := pkcs5Pad([]byte(plaintext), aes.BlockSize)

	// Encrypt using ECB mode (encrypt each block independently)
	ciphertext := make([]byte, len(paddedPlaintext))
	for i := 0; i < len(paddedPlaintext); i += aes.BlockSize {
		block.Encrypt(ciphertext[i:i+aes.BlockSize], paddedPlaintext[i:i+aes.BlockSize])
	}

	// Base64 encode
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// deriveKeyFromSecret derives AES-128 key from secret using SHA-1
// Takes first 16 bytes (32 hex chars) of SHA-1 hash
func deriveKeyFromSecret(secret string) ([]byte, error) {
	// SHA-1 hash the secret
	h := sha1.New()
	h.Write([]byte(secret))
	hashBytes := h.Sum(nil)

	// Convert to hex string
	hashHex := hex.EncodeToString(hashBytes)

	// Take first 32 hex chars (16 bytes)
	keyHex := hashHex[:32]

	// Convert hex to bytes
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex key: %w", err)
	}

	return key, nil
}

// pkcs5Pad adds PKCS5 padding to data
func pkcs5Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	return append(data, padtext...)
}

// pkcs5Unpad removes PKCS5 padding from data
func pkcs5Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, fmt.Errorf("data is empty")
	}

	padding := int(data[length-1])
	if padding > length || padding > aes.BlockSize {
		return nil, fmt.Errorf("invalid padding")
	}

	// Verify padding
	for i := 0; i < padding; i++ {
		if data[length-1-i] != byte(padding) {
			return nil, fmt.Errorf("invalid padding bytes")
		}
	}

	return data[:length-padding], nil
}
