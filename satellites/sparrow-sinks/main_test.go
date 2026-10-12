package main

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sarathsp06/sparrow/satellites/sparrow-sinks/sinks/email"
	"github.com/sarathsp06/sparrow/satellites/sparrow-sinks/sinks/otlp"
	"github.com/sarathsp06/sparrow/satellites/sparrow-sinks/sinks/s3"
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

func TestLoadConfig_InlineLibraryConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sinks.yaml")
	yaml := `webhook_secret: whsec_top
s3:
  bucket: events
  webhook_secret: whsec_s3
`
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":8788" {
		t.Errorf("listen default = %q", cfg.Listen)
	}
	if cfg.WebhookSecret != "whsec_top" || cfg.S3 == nil || cfg.S3.Bucket != "events" || cfg.S3.WebhookSecret != "whsec_s3" {
		t.Errorf("sinks.yaml not decoded: %+v", cfg)
	}
}

func TestNewHandler_MountsConfiguredSinksWithSharedSecret(t *testing.T) {
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	t.Cleanup(collector.Close)
	h, err := newHandler(context.Background(), fileConfig{
		WebhookSecret: testSecret,
		OTLP:          &otlp.Config{Endpoint: collector.URL},
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(envelopeBody)
	req := httptest.NewRequest(http.MethodPost, "/sinks/otlp", strings.NewReader(envelopeBody))
	for k, v := range signHeaders(t, body, testSecret) {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /sinks/otlp = %d: %s", rec.Code, rec.Body)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/sinks/email", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unconfigured sink = %d, want 404", rec.Code)
	}
}

func TestNewHandler_ConfigErrors(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	cases := map[string]struct {
		cfg  fileConfig
		want string
	}{
		"no sinks":         {fileConfig{WebhookSecret: testSecret}, "no sinks configured"},
		"no secret":        {fileConfig{OTLP: &otlp.Config{Endpoint: "http://c:4318"}}, "otlp: webhook_secret or signing_public_key is required"},
		"bad public key":   {fileConfig{SigningPublicKey: "zz", OTLP: &otlp.Config{Endpoint: "http://c:4318"}}, "otlp: signing_public_key: decode hex"},
		"short public key": {fileConfig{SigningPublicKey: "abcd", OTLP: &otlp.Config{Endpoint: "http://c:4318"}}, "must be 32 bytes"},
		"invalid sink":     {fileConfig{WebhookSecret: testSecret, S3: &s3.Config{}}, "s3: bucket is required"},
		"bad otlp url":     {fileConfig{WebhookSecret: testSecret, OTLP: &otlp.Config{Endpoint: "c:4318"}}, "otlp:"},
		"email incomplete": {fileConfig{WebhookSecret: testSecret, Email: &email.Config{}}, "email: smtp host and port are required"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := newHandler(context.Background(), tc.cfg, logger)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestNewHandler_EmailPerSinkSecretOverrides(t *testing.T) {
	other := "whsec_" + base64.StdEncoding.EncodeToString([]byte("email-only"))
	h, err := newHandler(context.Background(), fileConfig{
		WebhookSecret: testSecret,
		Email:         &email.Config{WebhookSecret: other, SMTP: email.SMTPConfig{Host: "127.0.0.1", Port: 1}, From: "a@b", To: []string{"c@d"}},
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(envelopeBody)
	req := httptest.NewRequest(http.MethodPost, "/sinks/email", strings.NewReader(envelopeBody))
	for k, v := range signHeaders(t, body, testSecret) { // signed with the shared secret
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("shared secret on overridden sink = %d, want 401", rec.Code)
	}
}

func TestNewHandler_Ed25519PublicKey(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	t.Cleanup(collector.Close)
	h, err := newHandler(context.Background(), fileConfig{
		SigningPublicKey: hex.EncodeToString(pub),
		OTLP:             &otlp.Config{Endpoint: collector.URL},
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}

	post := func(headers http.Header, body string) int {
		req := httptest.NewRequest(http.MethodPost, "/sinks/otlp", strings.NewReader(body))
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
	dual.Set("webhook-signature", "v1,"+"bm90LWNoZWNrZWQ= "+dual.Get("webhook-signature"))
	if code := post(dual, envelopeBody); code != http.StatusOK {
		t.Fatalf("dual signed = %d, want 200", code)
	}
	// HMAC only is not enough once a public key is configured.
	if code := post(signHeaders(t, body, testSecret), envelopeBody); code != http.StatusUnauthorized {
		t.Fatalf("v1 only = %d, want 401", code)
	}
	tampered := strings.Replace(envelopeBody, "usr_abc", "usr_evil", 1)
	if code := post(signEd25519Headers(body, priv), tampered); code != http.StatusUnauthorized {
		t.Fatalf("tampered = %d, want 401", code)
	}
}

// A sink's own webhook_secret wins over a top-level signing_public_key: the
// sink keeps its own verification settings as a whole.
func TestNewHandler_PerSinkSecretBeatsTopLevelPublicKey(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	t.Cleanup(collector.Close)
	h, err := newHandler(context.Background(), fileConfig{
		SigningPublicKey: hex.EncodeToString(pub),
		OTLP:             &otlp.Config{Endpoint: collector.URL, WebhookSecret: testSecret},
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(envelopeBody)
	req := httptest.NewRequest(http.MethodPost, "/sinks/otlp", strings.NewReader(envelopeBody))
	for k, v := range signHeaders(t, body, testSecret) {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("per-sink HMAC = %d, want 200", rec.Code)
	}
}
