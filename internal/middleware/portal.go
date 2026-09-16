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
)

// PortalTokens mints and verifies stateless, expiring bearer tokens that grant
// an end consumer scoped access to their own slice of the API
// (/v1/consumers/{consumer}/...). Tokens are HMAC-SHA256 signed with the
// server's encryption key: nothing is stored, and revocation is by expiry.
//
// Token format: "spt_v1.<b64url(consumer)>.<unix-expiry>.<b64url(signature)>"
// The consumer is plaintext (base64url) so the portal UI can display it
// without an extra endpoint; the signature covers consumer and expiry.
type PortalTokens struct {
	key []byte
}

const portalTokenPrefix = "spt_v1"

// Portal token TTL bounds.
const (
	PortalDefaultTTL = 7 * 24 * time.Hour
	PortalMaxTTL     = 30 * 24 * time.Hour
)

// ErrPortalTokenInvalid is returned by Verify for any malformed, tampered,
// or expired token. Deliberately unspecific to avoid oracle behavior.
var ErrPortalTokenInvalid = errors.New("invalid or expired portal token")

// NewPortalTokens creates a signer/verifier from a secret key.
// Returns nil for an empty key; a nil *PortalTokens rejects every token.
func NewPortalTokens(key []byte) *PortalTokens {
	if len(key) == 0 {
		return nil
	}
	return &PortalTokens{key: key}
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
	token := fmt.Sprintf("%s.%s.%s.%s",
		portalTokenPrefix,
		base64.RawURLEncoding.EncodeToString([]byte(consumer)),
		expStr,
		base64.RawURLEncoding.EncodeToString(p.sign(consumer, expStr)),
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
	if len(parts) != 4 || parts[0] != portalTokenPrefix {
		return "", ErrPortalTokenInvalid
	}
	consumerBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrPortalTokenInvalid
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return "", ErrPortalTokenInvalid
	}
	consumer := string(consumerBytes)
	if !hmac.Equal(sig, p.sign(consumer, parts[2])) {
		return "", ErrPortalTokenInvalid
	}
	expUnix, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || time.Now().Unix() > expUnix {
		return "", ErrPortalTokenInvalid
	}
	return consumer, nil
}

func (p *PortalTokens) sign(consumer, exp string) []byte {
	mac := hmac.New(sha256.New, p.key)
	mac.Write([]byte(portalTokenPrefix))
	mac.Write([]byte{0})
	mac.Write([]byte(consumer))
	mac.Write([]byte{0})
	mac.Write([]byte(exp))
	return mac.Sum(nil)
}
