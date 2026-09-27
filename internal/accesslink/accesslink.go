// Package accesslink hands the admin API key to a browser through a
// one-time, short-lived link instead of pasting the key into chat.
//
// An admin mints a link (POST /v1/access-links, or `sparrow access-link
// create --ttl 15m`). The link carries a signed token in the URL fragment:
//
//	https://sparrow.example.com/#access=sal_v1.<key-id>.<unix-expiry>.<b64url(nonce)>.<b64url(signature)>
//
// The UI posts the token to POST /access-link/redeem, gets the API key back,
// stores it like a key typed into the sign-in prompt, and drops the fragment.
//
// Properties:
//   - Signed with the encryption keyring (like portal tokens, but a distinct
//     token type, so neither can stand in for the other).
//   - Expires after the TTL (default 15 minutes, max 24 hours).
//   - Single use: each nonce can be redeemed once (tracked in Postgres).
//   - Bound to the current SPARROW_API_KEY: rotating the key invalidates all
//     unredeemed links.
//
// Whoever redeems a link ends up holding the real API key, so this makes
// sharing the key safer; it is not per-user access control.
package accesslink

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sarathsp06/sparrow/pkg/crypto"
)

const tokenPrefix = "sal_v1"

// TTL bounds for minted links.
const (
	DefaultTTL = 15 * time.Minute
	MaxTTL     = 24 * time.Hour
)

var (
	// ErrNoAPIKey means the server runs without SPARROW_API_KEY, so there is
	// no key to share (every endpoint is already open).
	ErrNoAPIKey = errors.New("SPARROW_API_KEY is not set; there is no API key to share")
	// ErrInvalid covers malformed, tampered, expired, and already-used links.
	// Deliberately unspecific to avoid oracle behavior.
	ErrInvalid = errors.New("invalid, expired, or already used access link")
)

// Claimer records that a link's nonce was redeemed. Claim returns false if
// the nonce was already claimed. expiresAt lets the store forget the nonce
// once the link could no longer verify anyway.
type Claimer interface {
	Claim(ctx context.Context, nonce string, expiresAt time.Time) (bool, error)
}

// Links mints and redeems access links. A nil *Links refuses everything.
type Links struct {
	primaryKeyID string
	keys         map[string][]byte
	apiKey       string
	claims       Claimer
	now          func() time.Time
}

// New returns a Links signing with keyring's primary key and sharing apiKey.
// Returns nil if the keyring is unusable.
func New(keyring *crypto.Keyring, apiKey string, claims Claimer) *Links {
	if keyring == nil || claims == nil {
		return nil
	}
	primary := keyring.Primary()
	if primary.ID == "" || len(primary.Material) == 0 {
		return nil
	}
	keys := make(map[string][]byte)
	for _, k := range keyring.Keys() {
		keys[k.ID] = append([]byte(nil), k.Material...)
	}
	return &Links{primaryKeyID: primary.ID, keys: keys, apiKey: apiKey, claims: claims, now: time.Now}
}

// Mint creates a link token valid for ttl (<= 0 uses DefaultTTL; capped at MaxTTL).
func (l *Links) Mint(ttl time.Duration) (string, time.Time, error) {
	if l == nil {
		return "", time.Time{}, errors.New("access links not configured")
	}
	if l.apiKey == "" {
		return "", time.Time{}, ErrNoAPIKey
	}
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	ttl = min(ttl, MaxTTL)

	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", time.Time{}, fmt.Errorf("generate nonce: %w", err)
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)
	exp := l.now().Add(ttl).Truncate(time.Second)
	expStr := strconv.FormatInt(exp.Unix(), 10)
	sig := sign(l.keys[l.primaryKeyID], l.primaryKeyID, expStr, nonce, l.apiKey)

	token := strings.Join([]string{tokenPrefix, l.primaryKeyID, expStr, nonce, base64.RawURLEncoding.EncodeToString(sig)}, ".")
	return token, exp, nil
}

// Redeem verifies token, marks it used, and returns the API key.
// Any verification failure returns ErrInvalid.
func (l *Links) Redeem(ctx context.Context, token string) (string, error) {
	if l == nil || l.apiKey == "" {
		return "", ErrInvalid
	}
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 5 || parts[0] != tokenPrefix {
		return "", ErrInvalid
	}
	keyID, expStr, nonce := parts[1], parts[2], parts[3]
	key, ok := l.keys[keyID]
	if !ok {
		return "", ErrInvalid
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[4])
	if err != nil || !hmac.Equal(sig, sign(key, keyID, expStr, nonce, l.apiKey)) {
		return "", ErrInvalid
	}
	expUnix, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || l.now().Unix() > expUnix {
		return "", ErrInvalid
	}

	claimed, err := l.claims.Claim(ctx, nonce, time.Unix(expUnix, 0))
	if err != nil {
		return "", fmt.Errorf("claim access link: %w", err)
	}
	if !claimed {
		return "", ErrInvalid
	}
	return l.apiKey, nil
}

// sign binds the token to its fields and to a fingerprint of the API key, so
// a key rotation invalidates every outstanding link.
func sign(key []byte, keyID, exp, nonce, apiKey string) []byte {
	fp := sha256.Sum256([]byte(apiKey))
	mac := hmac.New(sha256.New, key)
	for _, part := range [][]byte{[]byte(tokenPrefix), []byte(keyID), []byte(exp), []byte(nonce), fp[:]} {
		mac.Write(part)
		mac.Write([]byte{0})
	}
	return mac.Sum(nil)
}
