package webhooks

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"google.golang.org/grpc/codes"

	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

// EncryptSecretHeaders encrypts a plaintext secret headers map to bytes for storage.
// Returns nil if the map is empty or nil, or if encryption is not configured.
func (s *WebhookService) EncryptSecretHeaders(headers map[string]string) ([]byte, error) {
	if len(headers) == 0 {
		return nil, nil
	}
	if s.crypto == nil || !s.crypto.Enabled() {
		return nil, svcerrors.Error(codes.FailedPrecondition, "encryption is required for secret headers but SPARROW_ENCRYPTION_KEY is not configured")
	}
	return s.crypto.EncryptJSON(headers)
}

// DecryptSecretHeaders decrypts encrypted secret headers bytes back to a plaintext map.
// Returns nil map if the encrypted data is nil/empty or if encryption is not configured.
func (s *WebhookService) DecryptSecretHeaders(encrypted []byte) (map[string]string, error) {
	if len(encrypted) == 0 {
		return nil, nil
	}
	if s.crypto == nil || !s.crypto.Enabled() {
		return nil, svcerrors.Error(codes.FailedPrecondition, "encryption key not configured; cannot decrypt secret headers")
	}
	var headers map[string]string
	if err := s.crypto.DecryptJSON(encrypted, &headers); err != nil {
		return nil, fmt.Errorf("failed to decrypt secret headers: %w", err)
	}
	return headers, nil
}

// EncryptWebhookSecret encrypts a plaintext webhook secret string to bytes for storage.
// Returns nil if the secret is empty, or if encryption is not configured.
func (s *WebhookService) EncryptWebhookSecret(secret string) ([]byte, error) {
	if secret == "" {
		return nil, nil
	}
	if s.crypto == nil || !s.crypto.Enabled() {
		return nil, svcerrors.Error(codes.FailedPrecondition, "encryption is required for webhook secrets but SPARROW_ENCRYPTION_KEY is not configured")
	}
	return s.crypto.EncryptString(secret)
}

// DecryptWebhookSecret decrypts encrypted webhook secret bytes back to a plaintext string.
// Returns "" if the encrypted data is nil/empty or if encryption is not configured.
func (s *WebhookService) DecryptWebhookSecret(encrypted []byte) (string, error) {
	if len(encrypted) == 0 {
		return "", nil
	}
	if s.crypto == nil || !s.crypto.Enabled() {
		return "", svcerrors.Error(codes.FailedPrecondition, "encryption key not configured; cannot decrypt webhook secret")
	}
	return s.crypto.DecryptString(encrypted)
}

// WebhookSigningPublicKeyHex decrypts the envelope-encrypted Ed25519 private key
// and returns the hex-encoded public key. It returns an empty string on any
// error or if the key is absent, so transport modules never handle private-key
// material directly.
func (s *WebhookService) WebhookSigningPublicKeyHex(encryptedPrivKey []byte) string {
	if len(encryptedPrivKey) == 0 || s.crypto == nil {
		return ""
	}
	decrypted, err := s.crypto.DecryptString(encryptedPrivKey)
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
