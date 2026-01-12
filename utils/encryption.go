package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

// EncryptionKey is the encryption key for API keys (derived from environment variable or default)
var encryptionKey []byte

func init() {
	// Get encryption key from environment, or generate a default one
	keyString := os.Getenv("MIMIR_ENCRYPTION_KEY")
	if keyString == "" {
		// Default key for development (MUST be changed in production)
		keyString = "mimir-aip-default-encryption-key-change-in-production"
		GetLogger().Warn("Using default encryption key. Set MIMIR_ENCRYPTION_KEY environment variable for production.")
	}

	// Derive a 32-byte key from the string using SHA-256
	hash := sha256.Sum256([]byte(keyString))
	encryptionKey = hash[:]
}

// EncryptAPIKey encrypts an API key using AES-256-GCM
func EncryptAPIKey(plaintext string) (string, error) {
	if plaintext == "" {
		return "", fmt.Errorf("plaintext cannot be empty")
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// GCM mode provides authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create a nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return base64-encoded ciphertext
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAPIKey decrypts an encrypted API key using AES-256-GCM
func DecryptAPIKey(encrypted string) (string, error) {
	if encrypted == "" {
		return "", fmt.Errorf("encrypted text cannot be empty")
	}

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// MaskAPIKey returns a masked version of an API key (shows first/last 4 chars)
func MaskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
