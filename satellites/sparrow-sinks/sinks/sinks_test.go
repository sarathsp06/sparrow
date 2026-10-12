package sinks

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testSecret is a Standard Webhooks secret: "whsec_" + base64 key bytes.
var testSecret = "whsec_" + base64.StdEncoding.EncodeToString([]byte("test-signing-key"))

// signHeaders hand-computes the Standard Webhooks headers for body — the
// signing counterpart to pkg/signature's VerifyHMAC.
func signHeaders(t *testing.T, body []byte, secret string) http.Header {
	t.Helper()
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	if err != nil {
		t.Fatalf("decode secret: %v", err)
	}
	msgID := "msg_test"
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	mac := hmac.New(sha256.New, key)
	_, _ = fmt.Fprintf(mac, "%s.%s.", msgID, timestamp)
	mac.Write(body)
	h := http.Header{}
	h.Set("webhook-id", msgID)
	h.Set("webhook-timestamp", timestamp)
	h.Set("webhook-signature", "v1,"+base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	return h
}

const envelopeBody = `{"version":"1","event_id":"evt_1","event_name":"user.created","timestamp":"2026-09-12T10:00:00Z","attempt":1,"payload":{"user_id":"usr_abc"}}`

func postSink(t *testing.T, deliver func(context.Context, Envelope, []byte) error, body []byte, headers http.Header) *httptest.ResponseRecorder {
	t.Helper()
	verify, err := HMAC(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	h := Handle("test", verify, slog.New(slog.DiscardHandler), deliver)
	req := httptest.NewRequest(http.MethodPost, "/sinks/test", strings.NewReader(string(body)))
	for k, v := range headers {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSinkHandler_ValidSignature(t *testing.T) {
	body := []byte(envelopeBody)
	var got Envelope
	rec := postSink(t, func(_ context.Context, env Envelope, raw []byte) error {
		got = env
		if string(raw) != envelopeBody {
			t.Errorf("raw body altered: %s", raw)
		}
		return nil
	}, body, signHeaders(t, body, testSecret))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	if got.EventID != "evt_1" || got.EventName != "user.created" || got.Attempt != 1 {
		t.Errorf("parsed Envelope = %+v", got)
	}
}

func TestSinkHandler_TamperedBody(t *testing.T) {
	headers := signHeaders(t, []byte(envelopeBody), testSecret)
	tampered := []byte(strings.Replace(envelopeBody, "usr_abc", "usr_evil", 1))
	rec := postSink(t, func(context.Context, Envelope, []byte) error {
		t.Fatal("deliver must not run on tampered body")
		return nil
	}, tampered, headers)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestSinkHandler_MissingHeaders(t *testing.T) {
	rec := postSink(t, func(context.Context, Envelope, []byte) error { return nil }, []byte(envelopeBody), http.Header{})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestSinkHandler_WrongSecret(t *testing.T) {
	other := "whsec_" + base64.StdEncoding.EncodeToString([]byte("other-key"))
	body := []byte(envelopeBody)
	rec := postSink(t, func(context.Context, Envelope, []byte) error { return nil }, body, signHeaders(t, body, other))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestSinkHandler_TransformedBodyIs422(t *testing.T) {
	// A subscription with transform_enabled delivers arbitrary JSON, not the envelope.
	body := []byte(`{"text":"user.created fired"}`)
	rec := postSink(t, func(context.Context, Envelope, []byte) error {
		t.Fatal("deliver must not run without an envelope")
		return nil
	}, body, signHeaders(t, body, testSecret))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422: %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "envelope") {
		t.Errorf("422 body should explain the Envelope requirement: %s", rec.Body)
	}
}

func TestSinkHandler_DownstreamFailureIs502(t *testing.T) {
	body := []byte(envelopeBody)
	rec := postSink(t, func(context.Context, Envelope, []byte) error {
		return errors.New("smtp down")
	}, body, signHeaders(t, body, testSecret))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
}

// signEd25519Headers hand-computes Standard Webhooks headers with a "v1a,"
// Ed25519 signature, the way Sparrow signs ed25519 webhooks.
func signEd25519Headers(body []byte, priv ed25519.PrivateKey) http.Header {
	msgID := "msg_test"
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	sig := ed25519.Sign(priv, []byte(msgID+"."+timestamp+"."+string(body)))
	h := http.Header{}
	h.Set("webhook-id", msgID)
	h.Set("webhook-timestamp", timestamp)
	h.Set("webhook-signature", "v1a,"+base64.StdEncoding.EncodeToString(sig))
	return h
}

func TestHandle_Ed25519(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	verify, err := Ed25519(hex.EncodeToString(pub))
	if err != nil {
		t.Fatal(err)
	}
	h := Handle("test", verify, nil, func(context.Context, Envelope, []byte) error { return nil })
	post := func(headers http.Header, body string) int {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		for k, v := range headers {
			req.Header[k] = v
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	body := []byte(envelopeBody)
	if code := post(signEd25519Headers(body, priv), envelopeBody); code != http.StatusOK {
		t.Fatalf("v1a signed = %d, want 200", code)
	}
	// Dual-signed (what Sparrow sends for ed25519 webhooks): the v1a part verifies.
	dual := signEd25519Headers(body, priv)
	dual.Set("webhook-signature", "v1,bm90LWNoZWNrZWQ= "+dual.Get("webhook-signature"))
	if code := post(dual, envelopeBody); code != http.StatusOK {
		t.Fatalf("dual signed = %d, want 200", code)
	}
	if code := post(signHeaders(t, body, testSecret), envelopeBody); code != http.StatusUnauthorized {
		t.Fatalf("v1 only = %d, want 401", code)
	}
	tampered := strings.Replace(envelopeBody, "usr_abc", "usr_evil", 1)
	if code := post(signEd25519Headers(body, priv), tampered); code != http.StatusUnauthorized {
		t.Fatalf("tampered = %d, want 401", code)
	}
}

func TestVerifierFor(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(envelopeBody)
	// The public key wins when both are set: an HMAC-only delivery fails.
	v, err := VerifierFor(testSecret, hex.EncodeToString(pub))
	if err != nil {
		t.Fatal(err)
	}
	if v(body, signHeaders(t, body, testSecret)) == nil {
		t.Error("public key set: HMAC signature must not be enough")
	}
	if v, err = VerifierFor(testSecret, ""); err != nil || v(body, signHeaders(t, body, testSecret)) != nil {
		t.Errorf("secret only: err=%v", err)
	}
	for in, want := range map[[2]string]string{
		{"", ""}:     "webhook_secret or signing_public_key is required",
		{"", "zz"}:   "decode hex",
		{"", "abcd"}: "must be 32 bytes",
	} {
		if _, err := VerifierFor(in[0], in[1]); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("VerifierFor(%q, %q) err = %v, want %q", in[0], in[1], err, want)
		}
	}
}
