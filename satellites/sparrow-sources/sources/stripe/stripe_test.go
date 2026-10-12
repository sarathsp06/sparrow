package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func stripeSig(secret string, ts int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%d.%s", ts, body)
	return fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func TestVerifyStripeSignature(t *testing.T) {
	secret := "whsec_test"
	body := []byte(`{"type":"payment_intent.succeeded"}`)
	now := time.Unix(1_700_000_000, 0)

	valid := stripeSig(secret, now.Unix(), body)
	if err := verifySignature(valid, body, secret, now); err != nil {
		t.Errorf("valid signature rejected: %v", err)
	}
	// extra unknown scheme alongside a valid v1 still verifies
	if err := verifySignature(valid+",v0=garbage", body, secret, now); err != nil {
		t.Errorf("valid signature with extra scheme rejected: %v", err)
	}
	// expired (t older than 5 min)
	expired := stripeSig(secret, now.Add(-6*time.Minute).Unix(), body)
	if err := verifySignature(expired, body, secret, now); err == nil {
		t.Error("expired timestamp accepted")
	}
	// future timestamp beyond tolerance
	future := stripeSig(secret, now.Add(6*time.Minute).Unix(), body)
	if err := verifySignature(future, body, secret, now); err == nil {
		t.Error("future timestamp accepted")
	}
	// wrong secret
	if err := verifySignature(stripeSig("other", now.Unix(), body), body, secret, now); err == nil {
		t.Error("wrong-secret signature accepted")
	}
	// tampered body
	if err := verifySignature(valid, []byte(`{"type":"evil"}`), secret, now); err == nil {
		t.Error("tampered body accepted")
	}
	// garbage headers
	for _, h := range []string{"", "garbage", "t=abc,v1=00", fmt.Sprintf("t=%d", now.Unix()), "v1=00"} {
		if err := verifySignature(h, body, secret, now); err == nil {
			t.Errorf("garbage header %q accepted", h)
		}
	}
}

func TestMapStripeEvent(t *testing.T) {
	name, labels, err := mapEvent("stripe", []byte(`{"type":"payment_intent.succeeded","livemode":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if name != "stripe.payment_intent.succeeded" {
		t.Errorf("name = %q", name)
	}
	if labels["source"] != "stripe" || labels["livemode"] != "true" {
		t.Errorf("labels = %v", labels)
	}

	_, labels, _ = mapEvent("stripe", []byte(`{"type":"charge.refunded"}`))
	if labels["livemode"] != "false" {
		t.Errorf("livemode label = %q, want false", labels["livemode"])
	}

	for _, body := range []string{`not json`, `{}`, `{"type":""}`} {
		if _, _, err := mapEvent("stripe", []byte(body)); err == nil {
			t.Errorf("body %q: expected error", body)
		}
	}
}

// fakePusher records pushes and optionally fails.
type fakePusher struct {
	events []string
	fail   bool
}

func (f *fakePusher) PushEvent(_ context.Context, event string, _ json.RawMessage, _ map[string]string) error {
	if f.fail {
		return fmt.Errorf("boom")
	}
	f.events = append(f.events, event)
	return nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandler(t *testing.T) {
	if _, err := Handler(Config{}, &fakePusher{}, nil); err == nil || !strings.Contains(err.Error(), "signing_secret is required") {
		t.Fatalf("missing secret: err = %v", err)
	}
	push := &fakePusher{}
	h, err := Handler(Config{SigningSecret: "whsec_test"}, push, discardLogger()) // default prefix
	if err != nil {
		t.Fatal(err)
	}
	do := func(sig string) int {
		body := `{"type":"charge.refunded"}`
		req := httptest.NewRequest(http.MethodPost, "/anywhere", strings.NewReader(body))
		req.Header.Set("Stripe-Signature", sig)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}
	if code := do(stripeSig("whsec_test", time.Now().Unix(), []byte(`{"type":"charge.refunded"}`))); code != http.StatusOK {
		t.Fatalf("valid: status %d", code)
	}
	if code := do("t=1,v1=00"); code != http.StatusUnauthorized {
		t.Errorf("bad sig: status %d", code)
	}
	if got := fmt.Sprint(push.events); got != "[stripe.charge.refunded]" {
		t.Errorf("pushed events = %s", got)
	}
}
