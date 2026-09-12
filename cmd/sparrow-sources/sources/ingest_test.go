package sources

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
	fmt.Fprintf(mac, "%d.%s", ts, body)
	return fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func githubSig(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyStripeSignature(t *testing.T) {
	secret := "whsec_test"
	body := []byte(`{"type":"payment_intent.succeeded"}`)
	now := time.Unix(1_700_000_000, 0)

	valid := stripeSig(secret, now.Unix(), body)
	if err := verifyStripeSignature(valid, body, secret, now); err != nil {
		t.Errorf("valid signature rejected: %v", err)
	}
	// extra unknown scheme alongside a valid v1 still verifies
	if err := verifyStripeSignature(valid+",v0=garbage", body, secret, now); err != nil {
		t.Errorf("valid signature with extra scheme rejected: %v", err)
	}
	// expired (t older than 5 min)
	expired := stripeSig(secret, now.Add(-6*time.Minute).Unix(), body)
	if err := verifyStripeSignature(expired, body, secret, now); err == nil {
		t.Error("expired timestamp accepted")
	}
	// future timestamp beyond tolerance
	future := stripeSig(secret, now.Add(6*time.Minute).Unix(), body)
	if err := verifyStripeSignature(future, body, secret, now); err == nil {
		t.Error("future timestamp accepted")
	}
	// wrong secret
	if err := verifyStripeSignature(stripeSig("other", now.Unix(), body), body, secret, now); err == nil {
		t.Error("wrong-secret signature accepted")
	}
	// tampered body
	if err := verifyStripeSignature(valid, []byte(`{"type":"evil"}`), secret, now); err == nil {
		t.Error("tampered body accepted")
	}
	// garbage headers
	for _, h := range []string{"", "garbage", "t=abc,v1=00", fmt.Sprintf("t=%d", now.Unix()), "v1=00"} {
		if err := verifyStripeSignature(h, body, secret, now); err == nil {
			t.Errorf("garbage header %q accepted", h)
		}
	}
}

func TestVerifyGitHubSignature(t *testing.T) {
	secret := "gh_test"
	body := []byte(`{"action":"opened"}`)

	if !verifyGitHubSignature(githubSig(secret, body), body, secret) {
		t.Error("valid signature rejected")
	}
	if verifyGitHubSignature(githubSig("other", body), body, secret) {
		t.Error("wrong-secret signature accepted")
	}
	if verifyGitHubSignature(githubSig(secret, body), []byte(`{}`), secret) {
		t.Error("tampered body accepted")
	}
	for _, h := range []string{"", "sha256=", "sha1=abc", "deadbeef"} {
		if verifyGitHubSignature(h, body, secret) {
			t.Errorf("garbage header %q accepted", h)
		}
	}
}

func TestMapStripeEvent(t *testing.T) {
	name, labels, err := mapStripeEvent("stripe", []byte(`{"type":"payment_intent.succeeded","livemode":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if name != "stripe.payment_intent.succeeded" {
		t.Errorf("name = %q", name)
	}
	if labels["source"] != "stripe" || labels["livemode"] != "true" {
		t.Errorf("labels = %v", labels)
	}

	_, labels, _ = mapStripeEvent("stripe", []byte(`{"type":"charge.refunded"}`))
	if labels["livemode"] != "false" {
		t.Errorf("livemode label = %q, want false", labels["livemode"])
	}

	for _, body := range []string{`not json`, `{}`, `{"type":""}`} {
		if _, _, err := mapStripeEvent("stripe", []byte(body)); err == nil {
			t.Errorf("body %q: expected error", body)
		}
	}
}

func TestMapGitHubEvent(t *testing.T) {
	tests := []struct {
		ghEvent  string
		body     string
		wantName string
		wantRepo string
	}{
		{"pull_request", `{"action":"opened","repository":{"full_name":"acme/site"}}`, "github.pull_request.opened", "acme/site"},
		{"push", `{"repository":{"full_name":"acme/site"}}`, "github.push", "acme/site"},
		{"ping", `{}`, "github.ping", ""},
	}
	for _, tt := range tests {
		name, labels, err := mapGitHubEvent("github", tt.ghEvent, []byte(tt.body))
		if err != nil {
			t.Errorf("%s: %v", tt.ghEvent, err)
			continue
		}
		if name != tt.wantName {
			t.Errorf("%s: name = %q, want %q", tt.ghEvent, name, tt.wantName)
		}
		if labels["source"] != "github" || labels["repo"] != tt.wantRepo {
			t.Errorf("%s: labels = %v", tt.ghEvent, labels)
		}
	}

	if _, _, err := mapGitHubEvent("github", "", []byte(`{}`)); err == nil {
		t.Error("missing X-GitHub-Event accepted")
	}
	if _, _, err := mapGitHubEvent("github", "push", []byte(`not json`)); err == nil {
		t.Error("invalid body accepted")
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

func TestIngestHandler(t *testing.T) {
	cfg := IngestConfig{Providers: ProvidersConfig{
		Stripe: &StripeConfig{SigningSecret: "whsec_test", EventPrefix: "stripe"},
		GitHub: &GitHubConfig{Secret: "gh_test", EventPrefix: "github"},
	}}
	push := &fakePusher{}
	h := NewIngestHandler(cfg, push, discardLogger())

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
	w := do("/ingest/stripe", stripeBody,
		map[string]string{"Stripe-Signature": stripeSig("whsec_test", time.Now().Unix(), []byte(stripeBody))})
	if w.Code != http.StatusOK {
		t.Fatalf("valid stripe: status %d", w.Code)
	}

	// bad signature -> 401, empty body
	w = do("/ingest/stripe", stripeBody, map[string]string{"Stripe-Signature": "t=1,v1=00"})
	if w.Code != http.StatusUnauthorized || w.Body.Len() != 0 {
		t.Errorf("bad stripe sig: status %d body %q", w.Code, w.Body.String())
	}

	ghBody := `{"action":"opened","repository":{"full_name":"acme/site"}}`
	w = do("/ingest/github", ghBody, map[string]string{
		"X-Hub-Signature-256": githubSig("gh_test", []byte(ghBody)),
		"X-GitHub-Event":      "pull_request",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("valid github: status %d", w.Code)
	}

	// unknown provider path -> 404
	if w = do("/ingest/nope", "{}", nil); w.Code != http.StatusNotFound {
		t.Errorf("unknown provider: status %d", w.Code)
	}

	if got := fmt.Sprint(push.events); got != "[stripe.payment_intent.succeeded github.pull_request.opened]" {
		t.Errorf("pushed events = %s", got)
	}

	// push failure -> 502
	push.fail = true
	w = do("/ingest/github", ghBody, map[string]string{
		"X-Hub-Signature-256": githubSig("gh_test", []byte(ghBody)),
		"X-GitHub-Event":      "pull_request",
	})
	if w.Code != http.StatusBadGateway {
		t.Errorf("push failure: status %d", w.Code)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCronTickPushesOnMatch(t *testing.T) {
	jobs := []CronJob{
		{Event: "report.tick", spec: mustParse(t, "* * * * *"), payloadJSON: json.RawMessage(`{"kind":"hourly"}`), Labels: map[string]string{"source": "cron"}},
		{Event: "never.fires", spec: mustParse(t, "1 2 3 4 5"), payloadJSON: json.RawMessage(`{}`)},
	}
	push := &fakePusher{}
	cronTick(context.Background(), jobs, time.Date(2026, 9, 11, 10, 30, 0, 0, time.UTC), push, discardLogger())
	if got := fmt.Sprint(push.events); got != "[report.tick]" {
		t.Errorf("pushed events = %s", got)
	}
}
