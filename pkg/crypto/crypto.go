// Package crypto provides AES-256-GCM encryption and decryption for sensitive
// data such as secret headers and webhook secrets. The encryption key is loaded
// from the SPARROW_ENCRYPTION_KEY environment variable (64 hex chars = 32 bytes).
//
// When no encryption key is configured, Sparrow continues to work but any
// attempt to encrypt or decrypt will return an [ErrNoEncryptionKey] error.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ErrNoEncryptionKey is returned when encryption/decryption is attempted
// without a configured encryption key.
var ErrNoEncryptionKey = errors.New("crypto: encryption key not configured (set SPARROW_ENCRYPTION_KEY)")

// Service provides encrypt/decrypt operations using AES-256-GCM.
// A nil *Service or one created without a key is valid — calls to Encrypt
// and Decrypt will return ErrNoEncryptionKey.
type Service struct {
	aead cipher.AEAD
}

// NewService creates a new crypto service from a 32-byte AES-256 key.
// Pass nil to create a no-op service that returns ErrNoEncryptionKey on use.
func NewService(key []byte) (*Service, error) {
	if key == nil {
		return &Service{}, nil
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: key must be exactly 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new GCM: %w", err)
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

// Enabled reports whether the service has an encryption key configured.
func (s *Service) Enabled() bool {
	return s != nil && s.aead != nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns
// nonce || ciphertext (nonce is prepended).
func (s *Service) Encrypt(plaintext []byte) ([]byte, error) {
	if !s.Enabled() {
		return nil, ErrNoEncryptionKey
	}

	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: generate nonce: %w", err)
	}

	// Seal appends ciphertext to nonce slice: result = nonce || ciphertext || tag
	return s.aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts data produced by Encrypt (nonce || ciphertext).
func (s *Service) Decrypt(ciphertext []byte) ([]byte, error) {
	if !s.Enabled() {
		return nil, ErrNoEncryptionKey
	}

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

// EncryptJSON marshals v to JSON, then encrypts the result.
func (s *Service) EncryptJSON(v any) ([]byte, error) {
	plain, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("crypto: marshal: %w", err)
	}
	return s.Encrypt(plain)
}

// DecryptJSON decrypts ciphertext and unmarshals the result into v.
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
