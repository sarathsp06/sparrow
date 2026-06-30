// Package crypto provides envelope encryption (AES-256-GCM, per-record DEK)
// for sensitive data such as secret headers and webhook secrets.
//
// The Key Encryption Key (KEK) is loaded from the SPARROW_ENCRYPTION_KEY
// environment variable (64 hex chars = 32 bytes). This is required -- the
// server will not start without it.
//
// # Envelope Encryption
//
// Each Encrypt call generates a random 256-bit Data Encryption Key (DEK),
// encrypts the plaintext with the DEK, then wraps (encrypts) the DEK with
// the KEK. This enables efficient key rotation: re-wrap every DEK with the
// new KEK without touching the (potentially large) data.
//
// # Backward Compatibility
//
// Decrypt auto-detects envelope-encrypted data (version prefix 0x01). If the
// prefix is absent, it falls back to legacy direct AES-256-GCM decryption so
// that data encrypted before the envelope migration is still readable.
//
// When no encryption key is configured, any attempt to encrypt or decrypt
// returns [ErrNoEncryptionKey].
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ErrNoEncryptionKey is returned when encryption/decryption is attempted
// without a configured encryption key.
var ErrNoEncryptionKey = errors.New("crypto: encryption key not configured (set SPARROW_ENCRYPTION_KEY)")

// Service provides encrypt/decrypt operations using envelope encryption
// (per-record DEK wrapped with a KEK). A nil *Service or one created
// without a key is valid — calls return ErrNoEncryptionKey.
type Service struct {
	aead interface {
		Seal(dst, nonce, plaintext, additionalData []byte) []byte
		Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error)
		NonceSize() int
		Overhead() int
	}
}

// NewService creates a new crypto service from a 32-byte AES-256 key (the KEK).
// Pass nil to create a no-op service that returns ErrNoEncryptionKey on use.
func NewService(key []byte) (*Service, error) {
	if key == nil {
		return &Service{}, nil
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: key must be exactly 32 bytes, got %d", len(key))
	}
	aead, err := newAEAD(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new KEK cipher: %w", err)
	}
	return &Service{aead: aead}, nil
}

// ParseKey decodes a 64-character hex string into a 32-byte key suitable for
// NewService. Returns nil if raw is empty.
func ParseKey(raw string) ([]byte, error) {
	if raw == "" {
		return nil, nil
	}
	key, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("crypto: invalid hex key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: key must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}
	return key, nil
}

// GenerateKey generates a cryptographically random 32-byte key and returns
// it as a 64-character hex string suitable for SPARROW_ENCRYPTION_KEY.
func GenerateKey() (string, []byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", nil, fmt.Errorf("crypto: generate key: %w", err)
	}
	return hex.EncodeToString(key), key, nil
}

// Envelope encryption constants.
const (
	envelopeVersion    byte = 0x01
	dekSize                 = 32
	envelopeHeaderSize      = 3
	wrappedDEKSize          = 12 + dekSize + 16
	envelopeMinSize         = envelopeHeaderSize + wrappedDEKSize + 12 + 16
)

// EnvelopeEncrypt encrypts plaintext using envelope encryption:
//  1. Generates a random 256-bit DEK
//  2. Encrypts the plaintext with the DEK using AES-256-GCM
//  3. Encrypts (wraps) the DEK with the KEK
func (s *Service) EnvelopeEncrypt(plaintext []byte) ([]byte, error) {
	if !s.Enabled() {
		return nil, ErrNoEncryptionKey
	}

	dek := make([]byte, dekSize)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, fmt.Errorf("crypto: generate DEK: %w", err)
	}

	dataAEAD, err := newAEAD(dek)
	if err != nil {
		return nil, fmt.Errorf("crypto: create data cipher: %w", err)
	}

	dataNonce := make([]byte, dataAEAD.NonceSize())
	if _, err := io.ReadFull(rand.Reader, dataNonce); err != nil {
		return nil, fmt.Errorf("crypto: generate data nonce: %w", err)
	}
	encryptedData := dataAEAD.Seal(nil, dataNonce, plaintext, nil)

	dekNonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, dekNonce); err != nil {
		return nil, fmt.Errorf("crypto: generate DEK nonce: %w", err)
	}
	wrappedDEK := s.aead.Seal(dekNonce, dekNonce, dek, nil)

	edekLen := len(wrappedDEK)
	out := make([]byte, 0, envelopeHeaderSize+edekLen+len(dataNonce)+len(encryptedData))
	out = append(out, envelopeVersion)
	out = binary.LittleEndian.AppendUint16(out, uint16(edekLen))
	out = append(out, wrappedDEK...)
	out = append(out, dataNonce...)
	out = append(out, encryptedData...)

	clear(dek)
	return out, nil
}

// EnvelopeDecrypt decrypts data produced by EnvelopeEncrypt.
func (s *Service) EnvelopeDecrypt(ciphertext []byte) ([]byte, error) {
	if !s.Enabled() {
		return nil, ErrNoEncryptionKey
	}
	if len(ciphertext) < envelopeMinSize {
		return nil, errors.New("crypto: envelope ciphertext too short")
	}
	if ciphertext[0] != envelopeVersion {
		return nil, fmt.Errorf("crypto: unknown envelope version: 0x%02x", ciphertext[0])
	}

	edekLen := int(binary.LittleEndian.Uint16(ciphertext[1:3]))
	if edekLen <= 0 || envelopeHeaderSize+edekLen > len(ciphertext) {
		return nil, errors.New("crypto: invalid envelope: bad edek_len")
	}

	wrappedDEK := ciphertext[envelopeHeaderSize : envelopeHeaderSize+edekLen]
	rest := ciphertext[envelopeHeaderSize+edekLen:]

	kekNonceSize := s.aead.NonceSize()
	if len(wrappedDEK) < kekNonceSize {
		return nil, errors.New("crypto: wrapped DEK too short")
	}
	dekNonce, dekCipher := wrappedDEK[:kekNonceSize], wrappedDEK[kekNonceSize:]
	dek, err := s.aead.Open(nil, dekNonce, dekCipher, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: unwrap DEK: %w", err)
	}
	defer clear(dek)

	dataAEAD, err := newAEAD(dek)
	if err != nil {
		return nil, fmt.Errorf("crypto: create data cipher from DEK: %w", err)
	}

	dataNonceSize := dataAEAD.NonceSize()
	if len(rest) < dataNonceSize+dataAEAD.Overhead() {
		return nil, errors.New("crypto: envelope data too short")
	}
	dataNonce, dataEncrypted := rest[:dataNonceSize], rest[dataNonceSize:]

	plaintext, err := dataAEAD.Open(nil, dataNonce, dataEncrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decrypt data: %w", err)
	}
	return plaintext, nil
}

// IsEnvelopeEncrypted reports whether ciphertext appears to be envelope-encrypted.
func IsEnvelopeEncrypted(ciphertext []byte) bool {
	if len(ciphertext) < envelopeMinSize {
		return false
	}
	if ciphertext[0] != envelopeVersion {
		return false
	}
	edekLen := int(binary.LittleEndian.Uint16(ciphertext[1:3]))
	return edekLen == wrappedDEKSize && len(ciphertext) >= envelopeHeaderSize+edekLen+12+16
}

// newAEAD creates an AES-256-GCM AEAD from a 32-byte key.
func newAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// Enabled reports whether the service has an encryption key configured.
func (s *Service) Enabled() bool {
	return s != nil && s.aead != nil
}

// Decrypt decrypts ciphertext, auto-detecting the format:
//   - Envelope format (version 0x01 prefix): uses envelope decryption
//   - Legacy format (no prefix): falls back to direct AES-256-GCM
func (s *Service) Decrypt(ciphertext []byte) ([]byte, error) {
	if !s.Enabled() {
		return nil, ErrNoEncryptionKey
	}
	if IsEnvelopeEncrypted(ciphertext) {
		return s.EnvelopeDecrypt(ciphertext)
	}
	return s.directDecrypt(ciphertext)
}

// directDecrypt decrypts data encrypted with the legacy direct AES-256-GCM
// format: nonce(12) || ciphertext || tag(16).
func (s *Service) directDecrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := s.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("crypto: ciphertext too short")
	}
	nonce, data := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := s.aead.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decrypt: %w", err)
	}
	return plaintext, nil
}

// EncryptJSON marshals v to JSON, then encrypts using envelope encryption.
func (s *Service) EncryptJSON(v any) ([]byte, error) {
	plain, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("crypto: marshal: %w", err)
	}
	return s.EnvelopeEncrypt(plain)
}

// DecryptJSON decrypts ciphertext (auto-detecting format) and unmarshals into v.
func (s *Service) DecryptJSON(ciphertext []byte, v any) error {
	plain, err := s.Decrypt(ciphertext)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(plain, v); err != nil {
		return fmt.Errorf("crypto: unmarshal: %w", err)
	}
	return nil
}

// EncryptString encrypts a plaintext string and returns the ciphertext bytes.
func (s *Service) EncryptString(plaintext string) ([]byte, error) {
	if plaintext == "" {
		return nil, nil
	}
	return s.EnvelopeEncrypt([]byte(plaintext))
}

// DecryptString decrypts ciphertext back to a plaintext string.
func (s *Service) DecryptString(ciphertext []byte) (string, error) {
	if len(ciphertext) == 0 {
		return "", nil
	}
	plain, err := s.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
