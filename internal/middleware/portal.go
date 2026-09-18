package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sarathsp06/sparrow/pkg/crypto"
)

// PortalTokens mints and verifies stateless, expiring bearer tokens that grant
// an end consumer scoped access to their own slice of the API
// (/v1/consumers/{consumer}/...). Tokens are HMAC-SHA256 signed with the
// server's encryption key: nothing is stored, and revocation is by expiry.
//
// Token format: "spt_v2.<key-id>.<b64url(consumer)>.<unix-expiry>.<b64url(signature)>".
// The consumer is plaintext (base64url) so the portal UI can display it
// without an extra endpoint.
type PortalTokens struct {
	primaryKeyID string
	keys         map[string][]byte
}

const (
	portalTokenDefaultKeyID = "default"
	portalTokenPrefixV2     = "spt_v2"
)

// Portal token TTL bounds.
const (
	PortalDefaultTTL = 7 * 24 * time.Hour
	PortalMaxTTL     = 30 * 24 * time.Hour
)

// ErrPortalTokenInvalid is returned by Verify for any malformed, tampered,
// or expired token. Deliberately unspecific to avoid oracle behavior.
var ErrPortalTokenInvalid = errors.New("invalid or expired portal token")

// NewPortalTokens creates a signer/verifier from a single secret key.
// Returns nil for an empty key; a nil *PortalTokens rejects every token.
func NewPortalTokens(key []byte) *PortalTokens {
	if len(key) == 0 {
		return nil
	}
	keyring, err := crypto.NewKeyring([]crypto.Key{{ID: portalTokenDefaultKeyID, Material: key}}, portalTokenDefaultKeyID)
	if err != nil {
		return nil
	}
	return NewPortalTokensFromKeyring(keyring)
}

// NewPortalTokensFromKeyring creates a signer/verifier from the configured
// encryption keyring. New tokens are signed with the primary key ID and only
// v2 tokens are accepted.
func NewPortalTokensFromKeyring(keyring *crypto.Keyring) *PortalTokens {
	if keyring == nil {
		return nil
	}
	primary := keyring.Primary()
	if primary.ID == "" || len(primary.Material) == 0 {
		return nil
	}
	keys := make(map[string][]byte)
	for _, key := range keyring.Keys() {
		keys[key.ID] = append([]byte(nil), key.Material...)
	}
	return &PortalTokens{
		primaryKeyID: primary.ID,
		keys:         keys,
	}
}

// Mint creates a token scoped to consumer, valid for ttl (clamped to
// [1s, PortalMaxTTL]; ttl <= 0 uses PortalDefaultTTL).
func (p *PortalTokens) Mint(consumer string, ttl time.Duration) (string, time.Time, error) {
	if p == nil {
		return "", time.Time{}, errors.New("portal tokens not configured")
	}
	if consumer == "" {
		return "", time.Time{}, errors.New("consumer must not be empty")
	}
	if ttl <= 0 {
		ttl = PortalDefaultTTL
	}
	if ttl > PortalMaxTTL {
		ttl = PortalMaxTTL
	}
	exp := time.Now().Add(ttl).Truncate(time.Second)
	expStr := strconv.FormatInt(exp.Unix(), 10)

	sig := p.signV2(p.primaryKeyID, consumer, expStr)
	token := fmt.Sprintf("%s.%s.%s.%s.%s",
		portalTokenPrefixV2,
		p.primaryKeyID,
		base64.RawURLEncoding.EncodeToString([]byte(consumer)),
		expStr,
		base64.RawURLEncoding.EncodeToString(sig),
	)
	return token, exp, nil
}

// Verify checks a token's signature and expiry and returns the consumer it
// is scoped to. Any failure returns ErrPortalTokenInvalid.
func (p *PortalTokens) Verify(token string) (string, error) {
	if p == nil {
		return "", ErrPortalTokenInvalid
	}
	parts := strings.Split(token, ".")
	if len(parts) != 5 || parts[0] != portalTokenPrefixV2 {
		return "", ErrPortalTokenInvalid
	}
	consumer, expStr, sig, err := decodePortalToken(parts[2], parts[3], parts[4])
	if err != nil || !p.verifyExpiry(expStr) {
		return "", ErrPortalTokenInvalid
	}
	key, ok := p.keys[parts[1]]
	if !ok {
		return "", ErrPortalTokenInvalid
	}
	if !hmac.Equal(sig, signPortalTokenV2(key, parts[1], consumer, expStr)) {
		return "", ErrPortalTokenInvalid
	}
	return consumer, nil
}

func decodePortalToken(consumerPart, expStr, sigPart string) (string, string, []byte, error) {
	consumerBytes, err := base64.RawURLEncoding.DecodeString(consumerPart)
	if err != nil {
		return "", "", nil, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(sigPart)
	if err != nil {
		return "", "", nil, err
	}
	return string(consumerBytes), expStr, sig, nil
}

func (p *PortalTokens) verifyExpiry(expStr string) bool {
	expUnix, err := strconv.ParseInt(expStr, 10, 64)
	return err == nil && time.Now().Unix() <= expUnix
}

func (p *PortalTokens) signV2(keyID, consumer, exp string) []byte {
	key, ok := p.keys[keyID]
	if !ok {
		return nil
	}
	return signPortalTokenV2(key, keyID, consumer, exp)
}

func signPortalTokenV2(key []byte, keyID, consumer, exp string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(portalTokenPrefixV2))
	mac.Write([]byte{0})
	mac.Write([]byte(keyID))
	mac.Write([]byte{0})
	mac.Write([]byte(consumer))
	mac.Write([]byte{0})
	mac.Write([]byte(exp))
	return mac.Sum(nil)
}
