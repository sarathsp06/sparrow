package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Envelope encryption constants.
const (
	// envelopeVersion is the version byte prefix for envelope-encrypted data.
	envelopeVersion byte = 0x01

	// dekSize is the size of the Data Encryption Key in bytes (AES-256).
	dekSize = 32

	// envelopeHeaderSize is version(1) + edek_len(2).
	envelopeHeaderSize = 3

	// wrappedDEKSize is the expected size of a DEK encrypted with AES-256-GCM:
	// nonce(12) + ciphertext(32) + tag(16) = 60 bytes.
	wrappedDEKSize = 12 + dekSize + 16

	// envelopeMinSize is the minimum size of an envelope-encrypted blob:
	// header(3) + wrapped_dek(60) + data_nonce(12) + tag(16) = 91.
	envelopeMinSize = envelopeHeaderSize + wrappedDEKSize + 12 + 16
)

// EnvelopeEncrypt encrypts plaintext using envelope encryption:
//  1. Generates a random 256-bit DEK (Data Encryption Key)
//  2. Encrypts the plaintext with the DEK using AES-256-GCM
//  3. Encrypts (wraps) the DEK with the KEK (Key Encryption Key, held by Service)
//
// Wire format:
//
//	[0x01] [edek_len:2 LE] [encrypted_dek:edek_len] [data_nonce:12] [data_ciphertext+tag]
func (s *Service) EnvelopeEncrypt(plaintext []byte) ([]byte, error) {
	if !s.Enabled() {
		return nil, ErrNoEncryptionKey
	}

	// Generate random DEK
	dek := make([]byte, dekSize)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, fmt.Errorf("crypto: generate DEK: %w", err)
	}

	// Create data AEAD from DEK
	dataAEAD, err := newAEAD(dek)
	if err != nil {
		return nil, fmt.Errorf("crypto: create data cipher: %w", err)
	}

	// Encrypt data with DEK
	dataNonce := make([]byte, dataAEAD.NonceSize())
	if _, err := io.ReadFull(rand.Reader, dataNonce); err != nil {
		return nil, fmt.Errorf("crypto: generate data nonce: %w", err)
	}
	encryptedData := dataAEAD.Seal(nil, dataNonce, plaintext, nil)

	// Wrap DEK with KEK (using the Service's AEAD)
	dekNonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, dekNonce); err != nil {
		return nil, fmt.Errorf("crypto: generate DEK nonce: %w", err)
	}
	wrappedDEK := s.aead.Seal(dekNonce, dekNonce, dek, nil)

	// Build envelope: version(1) + edek_len(2 LE) + edek + data_nonce + encrypted_data
	edekLen := len(wrappedDEK)
	out := make([]byte, 0, envelopeHeaderSize+edekLen+len(dataNonce)+len(encryptedData))
	out = append(out, envelopeVersion)
	out = binary.LittleEndian.AppendUint16(out, uint16(edekLen))
	out = append(out, wrappedDEK...)
	out = append(out, dataNonce...)
	out = append(out, encryptedData...)

	// Zero the DEK from memory
	clear(dek)

	return out, nil
}

// EnvelopeDecrypt decrypts data produced by EnvelopeEncrypt.
// It unwraps the DEK using the KEK, then decrypts the data with the DEK.
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

	// Unwrap DEK with KEK
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

	// Decrypt data with DEK
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
// It checks the version byte and structural validity without attempting decryption.
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
