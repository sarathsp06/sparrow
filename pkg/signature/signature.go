// Package signature verifies Sparrow webhook delivery signatures in the
// Standard Webhooks format (https://www.standardwebhooks.com).
//
// Every delivery carries three headers:
//
//	webhook-id:        msg_<delivery-id>
//	webhook-timestamp: Unix seconds
//	webhook-signature: space-delimited signatures, e.g. "v1,<base64> v1a,<base64>"
//
// The signed message is "{webhook-id}.{webhook-timestamp}.{raw body}".
//
//   - "v1,"  is HMAC-SHA256 keyed with the webhook secret. Secrets in Standard
//     Webhooks format ("whsec_" + base64) are decoded before use; any other
//     secret is used as raw bytes.
//   - "v1a," is Ed25519, verified with the hex-encoded public key returned in
//     the webhook resource's signing_public_key field.
//
// Both verifiers reject deliveries whose webhook-timestamp is more than
// DefaultTolerance away from the current time, to prevent replay.
package signature

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultTolerance is the maximum accepted difference between the
// webhook-timestamp header and the verifier's clock.
const DefaultTolerance = 5 * time.Minute

var (
	// ErrMissingHeaders is returned when webhook-id, webhook-timestamp, or
	// webhook-signature is absent.
	ErrMissingHeaders = errors.New("signature: missing webhook-id, webhook-timestamp, or webhook-signature header")
	// ErrTimestamp is returned when webhook-timestamp is malformed or outside
	// DefaultTolerance (possible replay).
	ErrTimestamp = errors.New("signature: webhook-timestamp invalid or outside tolerance")
	// ErrNoMatch is returned when no signature of the requested scheme matches.
	ErrNoMatch = errors.New("signature: no matching signature")
)

// VerifyHMAC verifies the "v1," (HMAC-SHA256) signature of a delivery.
// payload must be the raw request body bytes, exactly as received.
// secret is the webhook secret, with or without the "whsec_" prefix.
func VerifyHMAC(payload []byte, headers http.Header, secret string) error {
	msgID, timestamp, sigs, err := parseHeaders(headers, time.Now())
	if err != nil {
		return err
	}

	key := []byte(secret)
	if encoded, ok := strings.CutPrefix(secret, "whsec_"); ok {
		key, err = base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return fmt.Errorf("signature: decode whsec_ secret: %w", err)
		}
	}

	mac := hmac.New(sha256.New, key)
	mac.Write(signedMessage(msgID, timestamp, payload))
	want := mac.Sum(nil)

	for _, sig := range sigs {
		encoded, ok := strings.CutPrefix(sig, "v1,")
		if !ok {
			continue
		}
		got, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			continue
		}
		if hmac.Equal(got, want) {
			return nil
		}
	}
	return ErrNoMatch
}

// VerifyEd25519 verifies the "v1a," (Ed25519) signature of a delivery.
// payload must be the raw request body bytes, exactly as received.
// publicKeyHex is the hex-encoded public key from the webhook resource's
// signing_public_key field.
func VerifyEd25519(payload []byte, headers http.Header, publicKeyHex string) error {
	msgID, timestamp, sigs, err := parseHeaders(headers, time.Now())
	if err != nil {
		return err
	}

	pub, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return fmt.Errorf("signature: decode hex public key: %w", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("signature: public key must be %d bytes, got %d", ed25519.PublicKeySize, len(pub))
	}

	message := signedMessage(msgID, timestamp, payload)
	for _, sig := range sigs {
		encoded, ok := strings.CutPrefix(sig, "v1a,")
		if !ok {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			continue
		}
		if ed25519.Verify(ed25519.PublicKey(pub), message, raw) {
			return nil
		}
	}
	return ErrNoMatch
}

// signedMessage builds "{msgID}.{timestamp}.{payload}" without extra copies.
func signedMessage(msgID, timestamp string, payload []byte) []byte {
	msg := make([]byte, 0, len(msgID)+len(timestamp)+len(payload)+2)
	msg = append(msg, msgID...)
	msg = append(msg, '.')
	msg = append(msg, timestamp...)
	msg = append(msg, '.')
	msg = append(msg, payload...)
	return msg
}

// parseHeaders extracts the Standard Webhooks headers and enforces the
// timestamp tolerance against now.
func parseHeaders(h http.Header, now time.Time) (msgID, timestamp string, sigs []string, err error) {
	msgID = h.Get("webhook-id")
	timestamp = h.Get("webhook-timestamp")
	sigHeader := h.Get("webhook-signature")
	if msgID == "" || timestamp == "" || sigHeader == "" {
		return "", "", nil, ErrMissingHeaders
	}

	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return "", "", nil, fmt.Errorf("%w: %v", ErrTimestamp, err)
	}
	if skew := now.Sub(time.Unix(seconds, 0)); skew > DefaultTolerance || skew < -DefaultTolerance {
		return "", "", nil, ErrTimestamp
	}

	return msgID, timestamp, strings.Fields(sigHeader), nil
}
