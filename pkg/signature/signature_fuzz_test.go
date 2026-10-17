package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// FuzzVerifyHMAC checks that arbitrary header values never panic or verify
// by accident, that a delivery signed with the secret verifies, and that
// changing its payload or secret makes it fail.
func FuzzVerifyHMAC(f *testing.F) {
	f.Add([]byte(`{"a":1}`), "msg_1", "secret", "v1,AAAA v1a,BBBB", "1700000000")
	f.Add([]byte{}, "", "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw", "", "")
	f.Add([]byte("x"), "a.1", "s", "v1,", "-1")
	now := time.Unix(1_700_000_000, 0)
	v := Verifier{Now: func() time.Time { return now }}

	f.Fuzz(func(t *testing.T, payload []byte, msgID, secret, rawSig, rawTS string) {
		h := http.Header{}
		h.Set("webhook-id", msgID)
		h.Set("webhook-timestamp", rawTS)
		h.Set("webhook-signature", rawSig)
		_ = v.VerifyEd25519(payload, h, secret)
		if v.VerifyHMAC(payload, h, secret) == nil &&
			!slices.Contains(strings.Fields(rawSig), "v1,"+sign(secret, msgID, rawTS, payload)) {
			t.Fatalf("header not signed with the secret verified: %q", rawSig)
		}

		// Round trip, with a raw (non-whsec_) secret.
		if secret == "" || strings.HasPrefix(secret, "whsec_") || msgID == "" {
			return
		}
		ts := strconv.FormatInt(now.Unix(), 10)
		signed := http.Header{}
		signed.Set("webhook-id", msgID)
		signed.Set("webhook-timestamp", ts)
		signed.Set("webhook-signature", "v1,"+sign(secret, msgID, ts, payload))
		if err := v.VerifyHMAC(payload, signed, secret); err != nil {
			t.Fatalf("correctly signed delivery rejected: %v", err)
		}
		if v.VerifyHMAC(append(slices.Clone(payload), 'x'), signed, secret) == nil {
			t.Fatal("tampered payload verified")
		}
		if v.VerifyHMAC(payload, signed, secret+"x") == nil {
			t.Fatal("wrong secret verified")
		}
	})
}

// sign is an independent HMAC-SHA256 signer for the Standard Webhooks
// message "{id}.{timestamp}.{payload}".
func sign(secret, msgID, ts string, payload []byte) string {
	key := []byte(secret)
	if enc, ok := strings.CutPrefix(secret, "whsec_"); ok {
		key, _ = base64.StdEncoding.DecodeString(enc)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msgID + "." + ts + "."))
	mac.Write(payload)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
