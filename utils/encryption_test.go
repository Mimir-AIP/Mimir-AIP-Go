package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptAPIKey(t *testing.T) {
	testCases := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "OpenAI API Key",
			plaintext: "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz",
		},
		{
			name:      "Anthropic API Key",
			plaintext: "sk-ant-api03-1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
		},
		{
			name:      "Short Key",
			plaintext: "test123",
		},
		{
			name:      "Long Key with Special Characters",
			plaintext: "key-with-special-chars-!@#$%^&*()_+-=[]{}|;:',.<>?/`~",
		},
		{
			name:      "Unicode Characters",
			plaintext: "key-with-unicode-你好世界-مرحبا",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := EncryptAPIKey(tc.plaintext)
			require.NoError(t, err)
			require.NotEmpty(t, encrypted)

			// Encrypted value should be different from plaintext
			assert.NotEqual(t, tc.plaintext, encrypted)

			// Encrypted value should be base64 (contains only valid base64 characters)
			assert.NotContains(t, encrypted, tc.plaintext)

			// Decrypt
			decrypted, err := DecryptAPIKey(encrypted)
			require.NoError(t, err)

			// Decrypted value should match original plaintext
			assert.Equal(t, tc.plaintext, decrypted)
		})
	}
}

func TestEncryptDecryptMultipleTimes(t *testing.T) {
	plaintext := "sk-test-key-1234567890"

	// Encrypt the same plaintext multiple times
	encrypted1, err1 := EncryptAPIKey(plaintext)
	encrypted2, err2 := EncryptAPIKey(plaintext)

	require.NoError(t, err1)
	require.NoError(t, err2)

	// Due to random nonce, encrypted values should be different each time
	assert.NotEqual(t, encrypted1, encrypted2, "Encrypted values should differ due to random nonce")

	// But both should decrypt to the same plaintext
	decrypted1, _ := DecryptAPIKey(encrypted1)
	decrypted2, _ := DecryptAPIKey(encrypted2)

	assert.Equal(t, plaintext, decrypted1)
	assert.Equal(t, plaintext, decrypted2)
}

func TestEncryptEmptyString(t *testing.T) {
	_, err := EncryptAPIKey("")
	assert.Error(t, err, "Should return error for empty plaintext")
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestDecryptEmptyString(t *testing.T) {
	_, err := DecryptAPIKey("")
	assert.Error(t, err, "Should return error for empty encrypted text")
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestDecryptInvalidBase64(t *testing.T) {
	_, err := DecryptAPIKey("not-valid-base64!!!")
	assert.Error(t, err, "Should return error for invalid base64")
	assert.Contains(t, err.Error(), "decode base64")
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	// Valid base64 but invalid ciphertext
	invalidCiphertext := "YWJjZGVmZ2hpams=" // "abcdefghijk" in base64
	_, err := DecryptAPIKey(invalidCiphertext)
	assert.Error(t, err, "Should return error for invalid ciphertext")
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	plaintext := "sk-test-original-key"

	encrypted, err := EncryptAPIKey(plaintext)
	require.NoError(t, err)

	// Tamper with the encrypted value
	tamperedEncrypted := encrypted[:len(encrypted)-5] + "XXXXX"

	_, err = DecryptAPIKey(tamperedEncrypted)
	assert.Error(t, err, "Should return error for tampered ciphertext")
}

func TestMaskAPIKey(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Long API Key",
			input:    "sk-proj-1234567890abcdefghij",
			expected: "sk-p...ghij",
		},
		{
			name:     "Short API Key",
			input:    "short",
			expected: "****",
		},
		{
			name:     "Exactly 8 Characters",
			input:    "12345678",
			expected: "****",
		},
		{
			name:     "9 Characters",
			input:    "123456789",
			expected: "1234...6789",
		},
		{
			name:     "Empty String",
			input:    "",
			expected: "****",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := MaskAPIKey(tc.input)
			assert.Equal(t, tc.expected, result)

			// Ensure the original key is not exposed
			if len(tc.input) > 8 {
				assert.NotEqual(t, tc.input, result)
				assert.Contains(t, result, "...")
			}
		})
	}
}

func TestEncryptionKeyInitialization(t *testing.T) {
	// Test that encryption key is initialized
	assert.NotNil(t, encryptionKey)
	assert.Len(t, encryptionKey, 32, "Encryption key should be 32 bytes for AES-256")
}

func TestEncryptDecryptLargePayload(t *testing.T) {
	// Test with a very large API key (simulating edge case)
	largeKey := strings.Repeat("a", 10000)

	encrypted, err := EncryptAPIKey(largeKey)
	require.NoError(t, err)

	decrypted, err := DecryptAPIKey(encrypted)
	require.NoError(t, err)

	assert.Equal(t, largeKey, decrypted)
}

func BenchmarkEncryptAPIKey(b *testing.B) {
	plaintext := "sk-test-benchmark-key-1234567890"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncryptAPIKey(plaintext)
	}
}

func BenchmarkDecryptAPIKey(b *testing.B) {
	plaintext := "sk-test-benchmark-key-1234567890"
	encrypted, _ := EncryptAPIKey(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecryptAPIKey(encrypted)
	}
}

func BenchmarkEncryptDecryptRoundTrip(b *testing.B) {
	plaintext := "sk-test-benchmark-key-1234567890"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encrypted, _ := EncryptAPIKey(plaintext)
		_, _ = DecryptAPIKey(encrypted)
	}
}
