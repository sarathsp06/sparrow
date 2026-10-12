package main

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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sarathsp06/sparrow/satellites/sparrow-sources/sources/github"
	"github.com/sarathsp06/sparrow/satellites/sparrow-sources/sources/stripe"
)

func TestLoadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sources.yaml")
	yaml := `sparrow:
  url: http://file:8080
cron:
  - schedule: "*/5 * * * *"
    event: report.tick
webhook:
  providers:
    github:
      secret: gh
`
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SPARROW_URL", "http://env:8080")
	t.Setenv("SPARROW_API_KEY", "")
	t.Setenv("SPARROW_CONSUMER", "")
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Sparrow.URL != "http://env:8080" {
		t.Errorf("SPARROW_URL override not applied: %q", cfg.Sparrow.URL)
	}
	if cfg.Webhook.Listen != ":8787" {
		t.Errorf("listen default = %q", cfg.Webhook.Listen)
	}
	if g := cfg.Webhook.Providers.GitHub; g == nil || g.Secret != "gh" {
		t.Errorf("inline WebhookConfig not decoded: %+v", cfg.Webhook)
	}
	if len(cfg.Cron) != 1 || cfg.Cron[0].Event != "report.tick" {
		t.Errorf("cron = %+v", cfg.Cron)
	}
}

func TestLoadConfig_RequiresURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sources.yaml")
	if err := os.WriteFile(path, []byte("cron: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SPARROW_URL", "")
	if _, err := loadConfig(path); err == nil {
		t.Fatal("expected sparrow.url error")
	}
}

func stripeSig(secret string, ts int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%d.%s", ts, body)
	return fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func githubSig(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
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

func TestWebhookHandler(t *testing.T) {
	cfg := providers{
		Stripe: &stripe.Config{Path: "/webhooks/stripe", SigningSecret: "whsec_test", SparrowEventPrefix: "stripe"},
		GitHub: &github.Config{Path: "/webhooks/github", Secret: "gh_test", SparrowEventPrefix: "github"},
	}
	push := &fakePusher{}
	h, err := newWebhookHandler(cfg, push, discardLogger())
	if err != nil {
		t.Fatal(err)
	}

	do := func(path, body string, hdr map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	stripeBody := `{"type":"payment_intent.succeeded","livemode":false}`
	w := do("/webhooks/stripe", stripeBody,
		map[string]string{"Stripe-Signature": stripeSig("whsec_test", time.Now().Unix(), []byte(stripeBody))})
	if w.Code != http.StatusOK {
		t.Fatalf("valid stripe: status %d", w.Code)
	}

	// bad signature -> 401, empty body
	w = do("/webhooks/stripe", stripeBody, map[string]string{"Stripe-Signature": "t=1,v1=00"})
	if w.Code != http.StatusUnauthorized || w.Body.Len() != 0 {
		t.Errorf("bad stripe sig: status %d body %q", w.Code, w.Body.String())
	}

	ghBody := `{"action":"opened","repository":{"full_name":"acme/site"}}`
	w = do("/webhooks/github", ghBody, map[string]string{
		"X-Hub-Signature-256": githubSig("gh_test", []byte(ghBody)),
		"X-GitHub-Event":      "pull_request",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("valid github: status %d", w.Code)
	}

	// unknown provider path -> 404
	if w = do("/webhooks/nope", "{}", nil); w.Code != http.StatusNotFound {
		t.Errorf("unknown provider: status %d", w.Code)
	}

	if got := fmt.Sprint(push.events); got != "[stripe.payment_intent.succeeded github.pull_request.opened]" {
		t.Errorf("pushed events = %s", got)
	}

	// push failure -> 502
	push.fail = true
	w = do("/webhooks/github", ghBody, map[string]string{
		"X-Hub-Signature-256": githubSig("gh_test", []byte(ghBody)),
		"X-GitHub-Event":      "pull_request",
	})
	if w.Code != http.StatusBadGateway {
		t.Errorf("push failure: status %d", w.Code)
	}
}

func TestWebhookHandlerConfigErrors(t *testing.T) {
	cases := map[string]providers{
		"webhook.providers.stripe: signing_secret is required": {Stripe: &stripe.Config{}},
		"webhook.providers.github: secret is required":         {GitHub: &github.Config{}},
		"stripe and github both use path /hook": {
			Stripe: &stripe.Config{SigningSecret: "s", Path: "/hook"},
			GitHub: &github.Config{Secret: "g", Path: "/hook"},
		},
	}
	for want, cfg := range cases {
		if _, err := newWebhookHandler(cfg, &fakePusher{}, discardLogger()); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("err = %v, want containing %q", err, want)
		}
	}
}

func TestWebhookHandlerDefaultPath(t *testing.T) {
	push := &fakePusher{}
	h, err := newWebhookHandler(providers{Stripe: &stripe.Config{SigningSecret: "whsec_test"}}, push, discardLogger())
	if err != nil {
		t.Fatal(err)
	}
	body := `{"type":"charge.refunded"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(body))
	req.Header.Set("Stripe-Signature", stripeSig("whsec_test", time.Now().Unix(), []byte(body)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK || fmt.Sprint(push.events) != "[stripe.charge.refunded]" {
		t.Fatalf("default stripe path: status %d events %v", w.Code, push.events)
	}
}
