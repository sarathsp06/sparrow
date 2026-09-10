package signature

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"
)

// signHMAC mirrors the server's signing: HMAC-SHA256 over
// "{msgID}.{timestamp}.{payload}" with the whsec_-decoded secret.
func signHMAC(t *testing.T, secret, msgID, timestamp string, payload []byte) string {
	t.Helper()
	key := []byte(secret)
	if len(secret) > 6 && secret[:6] == "whsec_" {
		decoded, err := base64.StdEncoding.DecodeString(secret[6:])
		if err != nil {
			t.Fatalf("decode secret: %v", err)
		}
		key = decoded
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msgID + "." + timestamp + "."))
	mac.Write(payload)
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifyHMAC(t *testing.T) {
	payload := []byte(`{"order_id":"ord_123"}`)
	const secret = "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw" // valid base64 after prefix
	msgID := "msg_test-delivery"
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	h := http.Header{}
	h.Set("webhook-id", msgID)
	h.Set("webhook-timestamp", ts)
	h.Set("webhook-signature", signHMAC(t, secret, msgID, ts, payload))

	if err := VerifyHMAC(payload, h, secret); err != nil {
		t.Fatalf("valid whsec_ signature rejected: %v", err)
	}

	// Tampered payload must fail.
	if err := VerifyHMAC([]byte(`{"order_id":"ord_999"}`), h, secret); !errors.Is(err, ErrNoMatch) {
		t.Fatalf("tampered payload: got %v, want ErrNoMatch", err)
	}

	// Wrong secret must fail.
	if err := VerifyHMAC(payload, h, "whsec_c29tZW90aGVyc2VjcmV0MDAwMDAwMDA="); !errors.Is(err, ErrNoMatch) {
		t.Fatalf("wrong secret: got %v, want ErrNoMatch", err)
	}
}

func TestVerifyHMACRawSecret(t *testing.T) {
	// Secrets without the whsec_ prefix are used as raw bytes.
	payload := []byte("hello")
	const secret = "plain-text-secret"
	msgID := "msg_raw"
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	h := http.Header{}
	h.Set("webhook-id", msgID)
	h.Set("webhook-timestamp", ts)
	h.Set("webhook-signature", signHMAC(t, secret, msgID, ts, payload))

	if err := VerifyHMAC(payload, h, secret); err != nil {
		t.Fatalf("raw secret signature rejected: %v", err)
	}
}

func TestVerifyHMACMultiSignatureHeader(t *testing.T) {
	// Header may hold several space-delimited signatures; v1 must be found
	// among them regardless of position.
	payload := []byte("multi")
	const secret = "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw"
	msgID := "msg_multi"
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	sig := signHMAC(t, secret, msgID, ts, payload)
	h := http.Header{}
	h.Set("webhook-id", msgID)
	h.Set("webhook-timestamp", ts)
	h.Set("webhook-signature", "v1a,AAAA "+sig)

	if err := VerifyHMAC(payload, h, secret); err != nil {
		t.Fatalf("v1 signature not found in multi-signature header: %v", err)
	}
}

func TestVerifyEd25519(t *testing.T) {
	payload := []byte(`{"order_id":"ord_123"}`)
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	msgID := "msg_ed"
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	message := []byte(msgID + "." + ts + "." + string(payload))
	sig := "v1a," + base64.StdEncoding.EncodeToString(ed25519.Sign(priv, message))

	h := http.Header{}
	h.Set("webhook-id", msgID)
	h.Set("webhook-timestamp", ts)
	// HMAC signature first, as the server emits for ed25519-type webhooks.
	h.Set("webhook-signature", "v1,AAAA "+sig)

	pubHex := hex.EncodeToString(pub)
	if err := VerifyEd25519(payload, h, pubHex); err != nil {
		t.Fatalf("valid Ed25519 signature rejected: %v", err)
	}

	// Tampered payload must fail.
	if err := VerifyEd25519([]byte("tampered"), h, pubHex); !errors.Is(err, ErrNoMatch) {
		t.Fatalf("tampered payload: got %v, want ErrNoMatch", err)
	}

	// Wrong key must fail.
	otherPub, _, _ := ed25519.GenerateKey(nil)
	if err := VerifyEd25519(payload, h, hex.EncodeToString(otherPub)); !errors.Is(err, ErrNoMatch) {
		t.Fatalf("wrong public key: got %v, want ErrNoMatch", err)
	}

	// Malformed key must error, not panic.
	if err := VerifyEd25519(payload, h, "zz"); err == nil {
		t.Fatal("malformed hex public key accepted")
	}
}

func TestTimestampTolerance(t *testing.T) {
	payload := []byte("x")
	const secret = "s"

	stale := strconv.FormatInt(time.Now().Add(-DefaultTolerance-time.Minute).Unix(), 10)
	h := http.Header{}
	h.Set("webhook-id", "msg_old")
	h.Set("webhook-timestamp", stale)
	h.Set("webhook-signature", signHMAC(t, secret, "msg_old", stale, payload))

	if err := VerifyHMAC(payload, h, secret); !errors.Is(err, ErrTimestamp) {
		t.Fatalf("stale timestamp: got %v, want ErrTimestamp", err)
	}

	h.Set("webhook-timestamp", "not-a-number")
	if err := VerifyHMAC(payload, h, secret); !errors.Is(err, ErrTimestamp) {
		t.Fatalf("malformed timestamp: got %v, want ErrTimestamp", err)
	}
}

func TestMissingHeaders(t *testing.T) {
	if err := VerifyHMAC([]byte("x"), http.Header{}, "s"); !errors.Is(err, ErrMissingHeaders) {
		t.Fatalf("empty headers: got %v, want ErrMissingHeaders", err)
	}
	if err := VerifyEd25519([]byte("x"), http.Header{}, "00"); !errors.Is(err, ErrMissingHeaders) {
		t.Fatalf("empty headers: got %v, want ErrMissingHeaders", err)
	}
}
