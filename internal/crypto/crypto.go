// Package crypto provides envelope encryption (AES-256-GCM, per-record DEK)
// for sensitive data such as secret headers and webhook secrets.
//
// The Key Encryption Key (KEK) is loaded from the SPARROW_ENCRYPTION_KEYS
// environment variable as a small keyring. One key is primary for new writes
// and all configured keys are valid for decryption.
//
// # Envelope Encryption
//
// Each Encrypt call generates a random 256-bit Data Encryption Key (DEK),
// encrypts the plaintext with the DEK, then wraps (encrypts) the DEK with
// the KEK.
//
// # Binding
//
// Every value is bound to where it is stored: callers pass a binding (for
// example the owning row's id and column) that is authenticated as GCM
// additional data on both the data and the wrapped DEK, and must pass the
// same binding to decrypt. A ciphertext copied into another row, column or
// tenant therefore fails to decrypt instead of being used there.
//
// Envelopes written before bindings existed (version 2) still decrypt, with
// the binding ignored; [Service.NeedsRewrap] reports them, and those under a
// key that is no longer primary, so they can be encrypted again.
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
var ErrNoEncryptionKey = errors.New("crypto: encryption key not configured (set SPARROW_ENCRYPTION_KEYS)")

const defaultKeyID = "default"

// Key is one KEK in a keyring.
type Key struct {
	ID       string
	Material []byte
}

// Keyring holds a small set of KEKs. One key is primary for new encryption;
// all configured keys remain available for decryption during rotation.
type Keyring struct {
	primaryID string
	keys      map[string]Key
	order     []string
}

// NewKeyring validates a set of KEKs and selects the primary key by ID.
func NewKeyring(keys []Key, primaryID string) (*Keyring, error) {
	if len(keys) == 0 {
		return nil, errors.New("crypto: keyring requires at least one key")
	}

	kr := &Keyring{keys: make(map[string]Key, len(keys)), order: make([]string, 0, len(keys))}
	for _, key := range keys {
		if key.ID == "" {
			return nil, errors.New("crypto: key id must not be empty")
		}
		if len(key.ID) > 255 {
			return nil, fmt.Errorf("crypto: key id %q is too long", key.ID)
		}
		if !isSafeKeyID(key.ID) {
			return nil, fmt.Errorf("crypto: key id %q must use only [A-Za-z0-9_-]", key.ID)
		}
		if len(key.Material) != 32 {
			return nil, fmt.Errorf("crypto: key %q must be exactly 32 bytes, got %d", key.ID, len(key.Material))
		}
		if _, exists := kr.keys[key.ID]; exists {
			return nil, fmt.Errorf("crypto: duplicate key id %q", key.ID)
		}
		material := append([]byte(nil), key.Material...)
		kr.keys[key.ID] = Key{ID: key.ID, Material: material}
		kr.order = append(kr.order, key.ID)
	}

	if primaryID == "" {
		if len(keys) != 1 {
			return nil, errors.New("crypto: primary key id is required when keyring has multiple keys")
		}
		primaryID = keys[0].ID
	}
	if _, ok := kr.keys[primaryID]; !ok {
		return nil, fmt.Errorf("crypto: primary key id %q not found in keyring", primaryID)
	}
	kr.primaryID = primaryID
	return kr, nil
}

type kek struct {
	id   string
	aead cipher.AEAD
}

// Service provides encrypt/decrypt operations using envelope encryption
// (per-record DEK wrapped with a KEK). A nil *Service or one created
// without a key is valid — calls return ErrNoEncryptionKey.
type Service struct {
	primary *kek
	byID    map[string]*kek
}

// NewService creates a new crypto service from a single 32-byte key. It is a
// convenience wrapper that emits the same keyed envelope format as the full
// keyring path, using the default key id.
func NewService(key []byte) (*Service, error) {
	if key == nil {
		return &Service{}, nil
	}
	keyring, err := NewKeyring([]Key{{ID: defaultKeyID, Material: key}}, defaultKeyID)
	if err != nil {
		return nil, err
	}
	return NewServiceFromKeyring(keyring)
}

// NewServiceFromKeyring creates a crypto service from a validated keyring.
// The primary key encrypts new values and all configured keys can decrypt.
func NewServiceFromKeyring(keyring *Keyring) (*Service, error) {
	if keyring == nil {
		return &Service{}, nil
	}

	byID := make(map[string]*kek, len(keyring.order))
	for _, id := range keyring.order {
		k, err := buildKEK(id, keyring.keys[id].Material)
		if err != nil {
			return nil, fmt.Errorf("crypto: new KEK cipher for %q: %w", id, err)
		}
		byID[id] = k
	}

	return &Service{
		primary: byID[keyring.primaryID],
		byID:    byID,
	}, nil
}

func buildKEK(id string, material []byte) (*kek, error) {
	aead, err := newAEAD(material)
	if err != nil {
		return nil, err
	}
	return &kek{id: id, aead: aead}, nil
}

func isSafeKeyID(id string) bool {
	for i := 0; i < len(id); i++ {
		b := id[i]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '-' || b == '_' {
			continue
		}
		return false
	}
	return true
}

// ParseKey decodes a 64-character hex string into a 32-byte key suitable for
// a keyring entry. Returns nil if raw is empty.
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
// it as a 64-character hex string suitable for a SPARROW_ENCRYPTION_KEYS entry.
func GenerateKey() (string, []byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", nil, fmt.Errorf("crypto: generate key: %w", err)
	}
	return hex.EncodeToString(key), key, nil
}

// Envelope encryption constants.
const (
	envelopeVersion          byte = 0x01
	envelopeVersionWithKeyID byte = 0x02 // unbound; decrypt only
	envelopeVersionBound     byte = 0x03
	dekSize                       = 32
	wrappedDEKSize                = 12 + dekSize + 16
	envelopeMinSize               = 1 + 1 + 1 + 2 + wrappedDEKSize + 12 + 16
)

// errNoBinding is returned when a value is encrypted without a binding.
var errNoBinding = errors.New("crypto: a binding is required")

// EnvelopeEncrypt encrypts plaintext using envelope encryption:
//  1. Generates a random 256-bit DEK
//  2. Encrypts the plaintext with the DEK using AES-256-GCM
//  3. Encrypts (wraps) the DEK with the KEK
//
// Both steps authenticate binding, which EnvelopeDecrypt must be given again.
func (s *Service) EnvelopeEncrypt(plaintext, binding []byte) ([]byte, error) {
	if !s.Enabled() {
		return nil, ErrNoEncryptionKey
	}
	if len(binding) == 0 {
		return nil, errNoBinding
	}
	return s.seal(envelopeVersionBound, plaintext, binding)
}

// seal writes an envelope of the given version under the primary key, with
// binding as additional data (nil for the unbound version 2).
func (s *Service) seal(version byte, plaintext, binding []byte) ([]byte, error) {
	dek := make([]byte, dekSize)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, fmt.Errorf("crypto: generate DEK: %w", err)
	}
	defer clear(dek)

	dataAEAD, err := newAEAD(dek)
	if err != nil {
		return nil, fmt.Errorf("crypto: create data cipher: %w", err)
	}

	dataNonce := make([]byte, dataAEAD.NonceSize())
	if _, err := io.ReadFull(rand.Reader, dataNonce); err != nil {
		return nil, fmt.Errorf("crypto: generate data nonce: %w", err)
	}
	encryptedData := dataAEAD.Seal(nil, dataNonce, plaintext, binding)

	dekNonce := make([]byte, s.primary.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, dekNonce); err != nil {
		return nil, fmt.Errorf("crypto: generate DEK nonce: %w", err)
	}
	wrappedDEK := s.primary.aead.Seal(dekNonce, dekNonce, dek, binding)

	kid := []byte(s.primary.id)
	out := make([]byte, 0, 1+1+len(kid)+2+len(wrappedDEK)+len(dataNonce)+len(encryptedData))
	out = append(out, version, byte(len(kid))) //nolint:gosec // key IDs are at most 255 bytes, checked when the keyring is loaded
	out = append(out, kid...)
	out = binary.LittleEndian.AppendUint16(out, uint16(len(wrappedDEK))) //nolint:gosec // a sealed 32-byte DEK plus nonce and tag, far below 64 KiB
	out = append(out, wrappedDEK...)
	out = append(out, dataNonce...)
	out = append(out, encryptedData...)
	return out, nil
}

// EnvelopeDecrypt decrypts data produced by EnvelopeEncrypt with the same
// binding. A version 2 envelope (written before bindings) decrypts whatever
// the binding.
func (s *Service) EnvelopeDecrypt(ciphertext, binding []byte) ([]byte, error) {
	if !s.Enabled() {
		return nil, ErrNoEncryptionKey
	}
	if len(ciphertext) == 0 {
		return nil, errors.New("crypto: envelope ciphertext too short")
	}
	switch ciphertext[0] {
	case envelopeVersionBound:
		if len(binding) == 0 {
			return nil, errNoBinding
		}
	case envelopeVersionWithKeyID:
		binding = nil
	default:
		return nil, fmt.Errorf("crypto: unknown envelope version: 0x%02x", ciphertext[0])
	}
	if len(ciphertext) < envelopeMinSize {
		return nil, errors.New("crypto: envelope ciphertext too short")
	}
	return s.decryptKeyedEnvelope(ciphertext, binding)
}

// NeedsRewrap reports whether ciphertext should be encrypted again: it is
// an unbound version 2 envelope, or its key is not the primary one. Data
// that is not an envelope this service understands reports false.
func (s *Service) NeedsRewrap(ciphertext []byte) bool {
	if !s.Enabled() || len(ciphertext) < 2 {
		return false
	}
	switch ciphertext[0] {
	case envelopeVersionWithKeyID:
		return true
	case envelopeVersionBound:
		kidLen := int(ciphertext[1])
		return len(ciphertext) >= 2+kidLen && string(ciphertext[2:2+kidLen]) != s.primary.id
	}
	return false
}

func (s *Service) decryptKeyedEnvelope(ciphertext, binding []byte) ([]byte, error) {
	kidLen := int(ciphertext[1])
	if kidLen <= 0 {
		return nil, errors.New("crypto: invalid envelope: missing key id")
	}
	if len(ciphertext) < 1+1+kidLen+2 {
		return nil, errors.New("crypto: invalid envelope: truncated key id")
	}

	kidStart := 2
	kidEnd := kidStart + kidLen
	keyID := string(ciphertext[kidStart:kidEnd])
	key, ok := s.byID[keyID]
	if !ok {
		return nil, fmt.Errorf("crypto: unknown key id %q", keyID)
	}

	edekLen := int(binary.LittleEndian.Uint16(ciphertext[kidEnd : kidEnd+2]))
	headerSize := kidEnd + 2
	if edekLen <= 0 || headerSize+edekLen > len(ciphertext) {
		return nil, errors.New("crypto: invalid envelope: bad edek_len")
	}

	wrappedDEK := ciphertext[headerSize : headerSize+edekLen]
	rest := ciphertext[headerSize+edekLen:]
	dek, err := unwrapDEK(key.aead, wrappedDEK, binding)
	if err != nil {
		return nil, err
	}
	defer clear(dek)
	return decryptDataWithDEK(dek, rest, binding)
}

func unwrapDEK(aead cipher.AEAD, wrappedDEK, binding []byte) ([]byte, error) {
	nonceSize := aead.NonceSize()
	if len(wrappedDEK) < nonceSize {
		return nil, errors.New("crypto: wrapped DEK too short")
	}
	dekNonce, dekCipher := wrappedDEK[:nonceSize], wrappedDEK[nonceSize:]
	dek, err := aead.Open(nil, dekNonce, dekCipher, binding)
	if err != nil {
		return nil, fmt.Errorf("crypto: unwrap DEK: %w", err)
	}
	return dek, nil
}

func decryptDataWithDEK(dek, rest, binding []byte) ([]byte, error) {
	dataAEAD, err := newAEAD(dek)
	if err != nil {
		return nil, fmt.Errorf("crypto: create data cipher from DEK: %w", err)
	}

	dataNonceSize := dataAEAD.NonceSize()
	if len(rest) < dataNonceSize+dataAEAD.Overhead() {
		return nil, errors.New("crypto: envelope data too short")
	}
	dataNonce, dataEncrypted := rest[:dataNonceSize], rest[dataNonceSize:]

	plaintext, err := dataAEAD.Open(nil, dataNonce, dataEncrypted, binding)
	if err != nil {
		return nil, fmt.Errorf("crypto: decrypt data: %w", err)
	}
	return plaintext, nil
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
	return s != nil && s.primary != nil
}

// EncryptJSON marshals v to JSON, then encrypts using envelope encryption.
func (s *Service) EncryptJSON(v any, binding []byte) ([]byte, error) {
	plaintext, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("crypto: marshal JSON: %w", err)
	}
	return s.EnvelopeEncrypt(plaintext, binding)
}

// DecryptJSON decrypts ciphertext and unmarshals into v.
func (s *Service) DecryptJSON(ciphertext []byte, v any, binding []byte) error {
	plain, err := s.EnvelopeDecrypt(ciphertext, binding)
	if err != nil {
		return err
	}
	return json.Unmarshal(plain, v)
}

// EncryptString encrypts a plaintext string and returns the ciphertext bytes.
// An empty string returns nil rather than an envelope, so callers can store
// NULL for "no secret configured" instead of an encrypted empty payload.
func (s *Service) EncryptString(plaintext string, binding []byte) ([]byte, error) {
	if plaintext == "" {
		return nil, nil
	}
	return s.EnvelopeEncrypt([]byte(plaintext), binding)
}

// DecryptString decrypts ciphertext back to a plaintext string. Nil/empty
// ciphertext (the EncryptString("") case) decrypts to "" with no error.
func (s *Service) DecryptString(ciphertext []byte, binding []byte) (string, error) {
	if len(ciphertext) == 0 {
		return "", nil
	}
	plain, err := s.EnvelopeDecrypt(ciphertext, binding)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
