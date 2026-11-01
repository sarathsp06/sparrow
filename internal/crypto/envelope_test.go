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
	ciphertext, err := svc.EnvelopeEncrypt(plaintext, testBinding)
	require.NoError(t, err)

	// Ciphertext should differ from plaintext
	assert.NotEqual(t, plaintext, ciphertext)

	// First byte should be version marker
	assert.Equal(t, envelopeVersionBound, ciphertext[0])

	// Decrypt should recover original
	decrypted, err := svc.EnvelopeDecrypt(ciphertext, testBinding)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEnvelopeEncrypt_UsesPrimaryKeyID(t *testing.T) {
	keyring, err := NewKeyring([]Key{
		{ID: "old", Material: testKey()},
		{ID: "new", Material: otherTestKey()},
	}, "new")
	require.NoError(t, err)

	svc, err := NewServiceFromKeyring(keyring)
	require.NoError(t, err)

	ciphertext, err := svc.EnvelopeEncrypt([]byte("secret data"), testBinding)
	require.NoError(t, err)

	assert.Equal(t, envelopeVersionBound, ciphertext[0])
	assert.Equal(t, byte(len("new")), ciphertext[1])
	assert.Equal(t, "new", string(ciphertext[2:2+len("new")]))
}

func TestEnvelopeDecrypt_RejectsLegacyVersion(t *testing.T) {
	keyring, err := NewKeyring([]Key{{ID: "new", Material: otherTestKey()}}, "new")
	require.NoError(t, err)

	svc, err := NewServiceFromKeyring(keyring)
	require.NoError(t, err)

	legacyCiphertext := []byte{envelopeVersion, 0x00, 0x00}
	_, err = svc.EnvelopeDecrypt(legacyCiphertext, testBinding)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown envelope version")
}

func TestEnvelopeDecrypt_UnknownKeyIDFails(t *testing.T) {
	keyring, err := NewKeyring([]Key{
		{ID: "new", Material: otherTestKey()},
	}, "new")
	require.NoError(t, err)

	svc, err := NewServiceFromKeyring(keyring)
	require.NoError(t, err)

	ciphertext, err := svc.EnvelopeEncrypt([]byte("secret data"), testBinding)
	require.NoError(t, err)

	ciphertext[1] = byte(len("old"))
	copy(ciphertext[2:], []byte("old"))

	_, err = svc.EnvelopeDecrypt(ciphertext, testBinding)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown key id")
}

func TestEnvelopeEncryptDecrypt_Empty(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	// Encrypting empty plaintext should work
	ciphertext, err := svc.EnvelopeEncrypt([]byte{}, testBinding)
	require.NoError(t, err)

	decrypted, err := svc.EnvelopeDecrypt(ciphertext, testBinding)
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

	ciphertext, err := svc.EnvelopeEncrypt(plaintext, testBinding)
	require.NoError(t, err)

	decrypted, err := svc.EnvelopeDecrypt(ciphertext, testBinding)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEnvelopeEncrypt_DifferentCiphertexts(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	plaintext := []byte("same input")
	ct1, err := svc.EnvelopeEncrypt(plaintext, testBinding)
	require.NoError(t, err)
	ct2, err := svc.EnvelopeEncrypt(plaintext, testBinding)
	require.NoError(t, err)

	// Different DEKs + nonces -> different ciphertext
	assert.NotEqual(t, ct1, ct2)

	// Both should decrypt correctly
	d1, err := svc.EnvelopeDecrypt(ct1, testBinding)
	require.NoError(t, err)
	d2, err := svc.EnvelopeDecrypt(ct2, testBinding)
	require.NoError(t, err)
	assert.Equal(t, d1, d2)
}

func TestEnvelopeDecrypt_WrongKEK(t *testing.T) {
	svc1, err := NewService(testKey())
	require.NoError(t, err)

	svc2, err := NewService([]byte("different-key-0123456789abcdefgh"))
	require.NoError(t, err)

	ciphertext, err := svc1.EnvelopeEncrypt([]byte("secret data"), testBinding)
	require.NoError(t, err)

	// Decrypting with a different KEK should fail (DEK unwrap fails)
	_, err = svc2.EnvelopeDecrypt(ciphertext, testBinding)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unwrap DEK")
}

func TestEnvelopeDecrypt_Tampered(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	ciphertext, err := svc.EnvelopeEncrypt([]byte("sensitive"), testBinding)
	require.NoError(t, err)

	// Tamper with last byte (data portion)
	ciphertext[len(ciphertext)-1] ^= 0xFF

	_, err = svc.EnvelopeDecrypt(ciphertext, testBinding)
	assert.Error(t, err)
}

func TestEnvelopeDecrypt_TooShort(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	_, err = svc.EnvelopeDecrypt([]byte{envelopeVersionWithKeyID}, testBinding)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestEnvelopeDecrypt_BadVersion(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	ciphertext, err := svc.EnvelopeEncrypt([]byte("data"), testBinding)
	require.NoError(t, err)

	// Corrupt version byte
	ciphertext[0] = 0xFF

	_, err = svc.EnvelopeDecrypt(ciphertext, testBinding)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown envelope version")
}

func TestEnvelopeEncrypt_NoKey(t *testing.T) {
	svc, err := NewService(nil)
	require.NoError(t, err)

	_, err = svc.EnvelopeEncrypt([]byte("data"), testBinding)
	assert.ErrorIs(t, err, ErrNoEncryptionKey)
}

func TestEnvelopeDecrypt_NoKey(t *testing.T) {
	svc, err := NewService(nil)
	require.NoError(t, err)

	_, err = svc.EnvelopeDecrypt([]byte("data"), testBinding)
	assert.ErrorIs(t, err, ErrNoEncryptionKey)
}

func TestEnvelopeDecrypt_BadEdekLen(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	// Craft an invalid blob with version=0x01 but bad edek_len
	bad := make([]byte, envelopeMinSize)
	bad[0] = envelopeVersion
	bad[1] = 0xFF // edek_len = 0x00FF = 255, way larger than remaining data
	bad[2] = 0x00

	_, err = svc.EnvelopeDecrypt(bad, testBinding)
	assert.Error(t, err)
}

func TestEnvelope_BindingMustMatch(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	ct, err := svc.EnvelopeEncrypt([]byte("secret"), []byte("webhook_registrations.webhook_secret:A"))
	require.NoError(t, err)

	// Copied into another row or column, the value does not decrypt.
	for _, other := range []string{"webhook_registrations.webhook_secret:B", "webhook_registrations.secret_headers:A"} {
		_, err = svc.EnvelopeDecrypt(ct, []byte(other))
		require.Error(t, err, other)
	}
	_, err = svc.EnvelopeDecrypt(ct, nil)
	require.ErrorIs(t, err, errNoBinding)

	_, err = svc.EnvelopeEncrypt([]byte("secret"), nil)
	require.ErrorIs(t, err, errNoBinding)
}

func TestEnvelope_UnboundVersion2StillDecrypts(t *testing.T) {
	svc, err := NewService(testKey())
	require.NoError(t, err)

	legacy, err := svc.seal(envelopeVersionWithKeyID, []byte("old"), nil)
	require.NoError(t, err)
	plain, err := svc.EnvelopeDecrypt(legacy, []byte("any:binding"))
	require.NoError(t, err)
	assert.Equal(t, "old", string(plain))
	assert.True(t, svc.NeedsRewrap(legacy))
}

func TestNeedsRewrap(t *testing.T) {
	oldOnly, err := NewServiceFromKeyring(mustKeyring(t, "old", "old"))
	require.NoError(t, err)
	rotated, err := NewServiceFromKeyring(mustKeyring(t, "new", "old", "new"))
	require.NoError(t, err)

	underOld, err := oldOnly.EnvelopeEncrypt([]byte("x"), testBinding)
	require.NoError(t, err)
	underNew, err := rotated.EnvelopeEncrypt([]byte("x"), testBinding)
	require.NoError(t, err)

	assert.False(t, oldOnly.NeedsRewrap(underOld))
	assert.True(t, rotated.NeedsRewrap(underOld), "written under a key that is no longer primary")
	assert.False(t, rotated.NeedsRewrap(underNew))
	assert.False(t, rotated.NeedsRewrap(nil))
	assert.False(t, rotated.NeedsRewrap([]byte("not an envelope")))
}

func mustKeyring(t *testing.T, primary string, ids ...string) *Keyring {
	t.Helper()
	material := map[string][]byte{"old": testKey(), "new": otherTestKey()}
	var keys []Key
	for _, id := range ids {
		keys = append(keys, Key{ID: id, Material: material[id]})
	}
	kr, err := NewKeyring(keys, primary)
	require.NoError(t, err)
	return kr
}
