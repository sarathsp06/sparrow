package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testKey() []byte {
	// 32 bytes of deterministic test key
	return []byte("01234567890123456789012345678901")
}

func TestNewService_NilKey(t *testing.T) {
	svc, err := NewService(nil)
	require.NoError(t, err)
	assert.False(t, svc.Enabled())
}

func TestNewService_ValidKey(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)
	assert.True(t, svc.Enabled())
}

func TestNewService_WrongKeyLength(t *testing.T) {
	_, err := NewService([]byte("too-short"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestEncryptDecrypt(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	plaintext := []byte("hello, secret headers!")
	ciphertext, err := svc.EnvelopeEncrypt(plaintext)
	require.NoError(t, err)

	// Ciphertext should be different from plaintext
	assert.NotEqual(t, plaintext, ciphertext)

	// Decrypt should recover original
	decrypted, err := svc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecrypt_DifferentNonces(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	plaintext := []byte("same input")
	ct1, err := svc.EnvelopeEncrypt(plaintext)
	require.NoError(t, err)
	ct2, err := svc.EnvelopeEncrypt(plaintext)
	require.NoError(t, err)

	// Two encryptions of the same plaintext should produce different ciphertext
	assert.NotEqual(t, ct1, ct2)

	// Both should decrypt correctly
	d1, err := svc.Decrypt(ct1)
	require.NoError(t, err)
	d2, err := svc.Decrypt(ct2)
	require.NoError(t, err)
	assert.Equal(t, d1, d2)
}

func TestDecrypt_TamperedData(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	ciphertext, err := svc.EnvelopeEncrypt([]byte("sensitive"))
	require.NoError(t, err)

	// Tamper with last byte
	ciphertext[len(ciphertext)-1] ^= 0xFF

	_, err = svc.Decrypt(ciphertext)
	assert.Error(t, err)
}

func TestDecrypt_TooShort(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	_, err = svc.Decrypt([]byte("short"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestNoEncryptionKey(t *testing.T) {
	svc, err := NewService(nil)
	require.NoError(t, err)

	_, err = svc.EnvelopeEncrypt([]byte("data"))
	assert.ErrorIs(t, err, ErrNoEncryptionKey)

	_, err = svc.Decrypt([]byte("data"))
	assert.ErrorIs(t, err, ErrNoEncryptionKey)
}

func TestEncryptDecryptJSON(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	original := map[string]string{
		"Authorization": "Bearer sk-secret-token",
		"X-Api-Key":     "key-12345",
	}

	ciphertext, err := svc.EncryptJSON(original)
	require.NoError(t, err)

	var decrypted map[string]string
	err = svc.DecryptJSON(ciphertext, &decrypted)
	require.NoError(t, err)

	assert.Equal(t, original, decrypted)
}

func TestParseKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantErr bool
	}{
		{"empty returns nil", "", true, false},
		{"valid 64 hex chars", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", false, false},
		{"invalid hex", "not-hex-at-all!", false, true},
		{"wrong length", "0123456789abcdef", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := ParseKey(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, key)
			} else {
				assert.Len(t, key, 32)
			}
		})
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	svc1, err := NewService(testKey())
	require.NoError(t, err)

	svc2, err := NewService([]byte("different-key-0123456789abcdefgh"))
	require.NoError(t, err)

	ciphertext, err := svc1.EnvelopeEncrypt([]byte("secret"))
	require.NoError(t, err)

	// Decrypting with a different key should fail
	_, err = svc2.Decrypt(ciphertext)
	assert.Error(t, err)
}

func TestDecrypt_BackwardCompatibility(t *testing.T) {
	// Simulate legacy direct-encrypted data by calling directDecrypt's inverse
	// (the old Encrypt format: nonce || ciphertext || tag, no envelope prefix).
	svc, err := NewService(testKey())
	require.NoError(t, err)

	plaintext := []byte("legacy secret headers data")

	// Manually create legacy direct-encrypted data using the KEK AEAD directly
	nonce := make([]byte, svc.aead.NonceSize())
	for i := range nonce {
		nonce[i] = byte(i) // deterministic nonce for test
	}
	legacyCiphertext := svc.aead.Seal(nonce, nonce, plaintext, nil)

	// The new Decrypt should fall back to direct decryption for legacy data
	assert.False(t, IsEnvelopeEncrypted(legacyCiphertext))
	decrypted, err := svc.Decrypt(legacyCiphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptString_DecryptString(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	// Normal string
	ct, err := svc.EncryptString("whsec_abc123")
	require.NoError(t, err)
	assert.NotNil(t, ct)

	decrypted, err := svc.DecryptString(ct)
	require.NoError(t, err)
	assert.Equal(t, "whsec_abc123", decrypted)

	// Empty string returns nil
	ct, err = svc.EncryptString("")
	require.NoError(t, err)
	assert.Nil(t, ct)

	decrypted, err = svc.DecryptString(nil)
	require.NoError(t, err)
	assert.Equal(t, "", decrypted)
}

func TestGenerateKey(t *testing.T) {
	hexKey, rawKey, err := GenerateKey()
	require.NoError(t, err)
	assert.Len(t, rawKey, 32)
	assert.Len(t, hexKey, 64)

	// Should be usable to create a service
	svc, err := NewService(rawKey)
	require.NoError(t, err)
	assert.True(t, svc.Enabled())

	// Hex should round-trip
	parsed, err := ParseKey(hexKey)
	require.NoError(t, err)
	assert.Equal(t, rawKey, parsed)
}
