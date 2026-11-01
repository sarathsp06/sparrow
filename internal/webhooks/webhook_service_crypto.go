package webhooks

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/google/uuid"

	svcerrors "github.com/sarathsp06/sparrow/internal/errors"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// EncryptSecretHeaders encrypts a plaintext secret headers map to bytes for storage.
// Returns nil if the map is empty or nil, or if encryption is not configured.
func (s *WebhookService) EncryptSecretHeaders(webhookID uuid.UUID, headers map[string]string) ([]byte, error) {
	if len(headers) == 0 {
		return nil, nil
	}
	if s.crypto == nil || !s.crypto.Enabled() {
		return nil, svcerrors.Error(svcerrors.FailedPrecondition, "encryption is required for secret headers but SPARROW_ENCRYPTION_KEYS is not configured")
	}
	return s.crypto.EncryptJSON(headers, store.WebhookSecretBinding(webhookID, store.ColumnSecretHeaders))
}

// DecryptSecretHeaders decrypts encrypted secret headers bytes back to a plaintext map.
// Returns nil map if the encrypted data is nil/empty or if encryption is not configured.
func (s *WebhookService) DecryptSecretHeaders(webhookID uuid.UUID, encrypted []byte) (map[string]string, error) {
	if len(encrypted) == 0 {
		return nil, nil
	}
	if s.crypto == nil || !s.crypto.Enabled() {
		return nil, svcerrors.Error(svcerrors.FailedPrecondition, "encryption key not configured; cannot decrypt secret headers")
	}
	var headers map[string]string
	if err := s.crypto.DecryptJSON(encrypted, &headers, store.WebhookSecretBinding(webhookID, store.ColumnSecretHeaders)); err != nil {
		return nil, fmt.Errorf("failed to decrypt secret headers: %w", err)
	}
	return headers, nil
}

// EncryptWebhookSecret encrypts a plaintext webhook secret string to bytes for storage.
// Returns nil if the secret is empty, or if encryption is not configured.
func (s *WebhookService) EncryptWebhookSecret(webhookID uuid.UUID, secret string) ([]byte, error) {
	if secret == "" {
		return nil, nil
	}
	if s.crypto == nil || !s.crypto.Enabled() {
		return nil, svcerrors.Error(svcerrors.FailedPrecondition, "encryption is required for webhook secrets but SPARROW_ENCRYPTION_KEYS is not configured")
	}
	return s.crypto.EncryptString(secret, store.WebhookSecretBinding(webhookID, store.ColumnWebhookSecret))
}

// DecryptWebhookSecret decrypts encrypted webhook secret bytes back to a plaintext string.
// Returns "" if the encrypted data is nil/empty or if encryption is not configured.
func (s *WebhookService) DecryptWebhookSecret(webhookID uuid.UUID, encrypted []byte) (string, error) {
	if len(encrypted) == 0 {
		return "", nil
	}
	if s.crypto == nil || !s.crypto.Enabled() {
		return "", svcerrors.Error(svcerrors.FailedPrecondition, "encryption key not configured; cannot decrypt webhook secret")
	}
	return s.crypto.DecryptString(encrypted, store.WebhookSecretBinding(webhookID, store.ColumnWebhookSecret))
}

// WebhookSigningPublicKeyHex decrypts the envelope-encrypted Ed25519 private key
// and returns the hex-encoded public key. It returns an empty string on any
// error or if the key is absent, so transport modules never handle private-key
// material directly.
func (s *WebhookService) WebhookSigningPublicKeyHex(webhookID uuid.UUID, encryptedPrivKey []byte) string {
	if len(encryptedPrivKey) == 0 || s.crypto == nil {
		return ""
	}
	decrypted, err := s.crypto.DecryptString(encryptedPrivKey, store.WebhookSecretBinding(webhookID, store.ColumnEd25519PrivateKey))
	if err != nil {
		return ""
	}
	privKey := ed25519.PrivateKey([]byte(decrypted))
	if len(privKey) != ed25519.PrivateKeySize {
		return ""
	}
	pubKey := privKey.Public().(ed25519.PublicKey)
	return hex.EncodeToString(pubKey)
}

// generateEncryptedEd25519Key generates a new Ed25519 keypair and returns the
// envelope-encrypted private key for storage. The public key is derived from
// the private key at runtime (see WebhookSigningPublicKeyHex).
// Returns nil if encryption is not configured.
func (s *WebhookService) generateEncryptedEd25519Key(webhookID uuid.UUID) ([]byte, error) {
	if s.crypto == nil || !s.crypto.Enabled() {
		return nil, nil
	}
	_, privKey, err := ed25519.GenerateKey(nil) // crypto/rand by default
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 keypair: %w", err)
	}
	encPrivKey, err := s.crypto.EncryptString(string(privKey), store.WebhookSecretBinding(webhookID, store.ColumnEd25519PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt Ed25519 private key: %w", err)
	}
	return encPrivKey, nil
}

// generateWebhookSecret generates a new cryptographically random secret
// in Standard Webhooks format: "whsec_" prefix + base64 encoded entropy.
func generateWebhookSecret() (string, error) {
	// 24 bytes of entropy = 32 chars in base64
	key := make([]byte, 24)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return "whsec_" + base64.StdEncoding.EncodeToString(key), nil
}
