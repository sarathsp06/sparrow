package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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

func postSink(t *testing.T, deliver func(context.Context, envelope, []byte) error, body []byte, headers http.Header) *httptest.ResponseRecorder {
	t.Helper()
	h := sinkHandler("test", testSecret, slog.New(slog.DiscardHandler), deliver)
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
	var got envelope
	rec := postSink(t, func(_ context.Context, env envelope, raw []byte) error {
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
		t.Errorf("parsed envelope = %+v", got)
	}
}

func TestSinkHandler_TamperedBody(t *testing.T) {
	headers := signHeaders(t, []byte(envelopeBody), testSecret)
	tampered := []byte(strings.Replace(envelopeBody, "usr_abc", "usr_evil", 1))
	rec := postSink(t, func(context.Context, envelope, []byte) error {
		t.Fatal("deliver must not run on tampered body")
		return nil
	}, tampered, headers)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestSinkHandler_MissingHeaders(t *testing.T) {
	rec := postSink(t, func(context.Context, envelope, []byte) error { return nil }, []byte(envelopeBody), http.Header{})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestSinkHandler_WrongSecret(t *testing.T) {
	other := "whsec_" + base64.StdEncoding.EncodeToString([]byte("other-key"))
	body := []byte(envelopeBody)
	rec := postSink(t, func(context.Context, envelope, []byte) error { return nil }, body, signHeaders(t, body, other))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestSinkHandler_TransformedBodyIs422(t *testing.T) {
	// A subscription with transform_enabled delivers arbitrary JSON, not the envelope.
	body := []byte(`{"text":"user.created fired"}`)
	rec := postSink(t, func(context.Context, envelope, []byte) error {
		t.Fatal("deliver must not run without an envelope")
		return nil
	}, body, signHeaders(t, body, testSecret))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422: %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "envelope") {
		t.Errorf("422 body should explain the envelope requirement: %s", rec.Body)
	}
}

func TestSinkHandler_DownstreamFailureIs502(t *testing.T) {
	body := []byte(envelopeBody)
	rec := postSink(t, func(context.Context, envelope, []byte) error {
		return errors.New("smtp down")
	}, body, signHeaders(t, body, testSecret))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
}
