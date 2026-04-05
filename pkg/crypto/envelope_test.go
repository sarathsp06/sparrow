package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvelopeEncryptDecrypt(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	plaintext := []byte("hello, envelope encryption!")
	ciphertext, err := svc.EnvelopeEncrypt(plaintext)
	require.NoError(t, err)

	// Ciphertext should differ from plaintext
	assert.NotEqual(t, plaintext, ciphertext)

	// First byte should be version marker
	assert.Equal(t, envelopeVersion, ciphertext[0])

	// Should be detected as envelope-encrypted
	assert.True(t, IsEnvelopeEncrypted(ciphertext))

	// Decrypt should recover original
	decrypted, err := svc.EnvelopeDecrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEnvelopeEncryptDecrypt_Empty(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	// Encrypting empty plaintext should work
	ciphertext, err := svc.EnvelopeEncrypt([]byte{})
	require.NoError(t, err)
	assert.True(t, IsEnvelopeEncrypted(ciphertext))

	decrypted, err := svc.EnvelopeDecrypt(ciphertext)
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}

func TestEnvelopeEncryptDecrypt_LargePayload(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	// 1 MB payload
	plaintext := make([]byte, 1<<20)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	ciphertext, err := svc.EnvelopeEncrypt(plaintext)
	require.NoError(t, err)

	decrypted, err := svc.EnvelopeDecrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEnvelopeEncrypt_DifferentCiphertexts(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	plaintext := []byte("same input")
	ct1, err := svc.EnvelopeEncrypt(plaintext)
	require.NoError(t, err)
	ct2, err := svc.EnvelopeEncrypt(plaintext)
	require.NoError(t, err)

	// Different DEKs + nonces -> different ciphertext
	assert.NotEqual(t, ct1, ct2)

	// Both should decrypt correctly
	d1, err := svc.EnvelopeDecrypt(ct1)
	require.NoError(t, err)
	d2, err := svc.EnvelopeDecrypt(ct2)
	require.NoError(t, err)
	assert.Equal(t, d1, d2)
}

func TestEnvelopeDecrypt_WrongKEK(t *testing.T) {
	svc1, err := NewService(testKey())
	require.NoError(t, err)

	svc2, err := NewService([]byte("different-key-0123456789abcdefgh"))
	require.NoError(t, err)

	ciphertext, err := svc1.EnvelopeEncrypt([]byte("secret data"))
	require.NoError(t, err)

	// Decrypting with a different KEK should fail (DEK unwrap fails)
	_, err = svc2.EnvelopeDecrypt(ciphertext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unwrap DEK")
}

func TestEnvelopeDecrypt_Tampered(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	ciphertext, err := svc.EnvelopeEncrypt([]byte("sensitive"))
	require.NoError(t, err)

	// Tamper with last byte (data portion)
	ciphertext[len(ciphertext)-1] ^= 0xFF

	_, err = svc.EnvelopeDecrypt(ciphertext)
	assert.Error(t, err)
}

func TestEnvelopeDecrypt_TooShort(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	_, err = svc.EnvelopeDecrypt([]byte("short"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestEnvelopeDecrypt_BadVersion(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	ciphertext, err := svc.EnvelopeEncrypt([]byte("data"))
	require.NoError(t, err)

	// Corrupt version byte
	ciphertext[0] = 0xFF

	_, err = svc.EnvelopeDecrypt(ciphertext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown envelope version")
}

func TestEnvelopeEncrypt_NoKey(t *testing.T) {
	svc, err := NewService(nil)
	require.NoError(t, err)

	_, err = svc.EnvelopeEncrypt([]byte("data"))
	assert.ErrorIs(t, err, ErrNoEncryptionKey)
}

func TestEnvelopeDecrypt_NoKey(t *testing.T) {
	svc, err := NewService(nil)
	require.NoError(t, err)

	_, err = svc.EnvelopeDecrypt([]byte("data"))
	assert.ErrorIs(t, err, ErrNoEncryptionKey)
}

func TestIsEnvelopeEncrypted(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	// Envelope-encrypted data should be detected
	ct, err := svc.EnvelopeEncrypt([]byte("test"))
	require.NoError(t, err)
	assert.True(t, IsEnvelopeEncrypted(ct))

	// Legacy direct-encrypted data should NOT be detected as envelope.
	// Craft it manually using the internal directDecrypt-compatible format.
	legacyDirect := make([]byte, 12+16+4) // nonce(12) + ciphertext+tag
	legacyDirect[0] = 0x00                // first byte != envelopeVersion
	assert.False(t, IsEnvelopeEncrypted(legacyDirect))

	// Even if first byte happens to be 0x01, edek_len won't match wrappedDEKSize
	legacyDirect[0] = envelopeVersion
	legacyDirect[1] = 0x00 // edek_len = 0, not wrappedDEKSize
	legacyDirect[2] = 0x00
	assert.False(t, IsEnvelopeEncrypted(legacyDirect))

	// Random bytes should not be detected
	assert.False(t, IsEnvelopeEncrypted([]byte("random garbage")))
	assert.False(t, IsEnvelopeEncrypted(nil))
	assert.False(t, IsEnvelopeEncrypted([]byte{}))
}

func TestEnvelopeDecrypt_BadEdekLen(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	// Craft an invalid blob with version=0x01 but bad edek_len
	bad := make([]byte, envelopeMinSize)
	bad[0] = envelopeVersion
	bad[1] = 0xFF // edek_len = 0x00FF = 255, way larger than remaining data
	bad[2] = 0x00

	_, err = svc.EnvelopeDecrypt(bad)
	assert.Error(t, err)
}
