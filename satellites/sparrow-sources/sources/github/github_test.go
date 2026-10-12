package github

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
)

func githubSig(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyGitHubSignature(t *testing.T) {
	secret := "gh_test"
	body := []byte(`{"action":"opened"}`)

	if !verifySignature(githubSig(secret, body), body, secret) {
		t.Error("valid signature rejected")
	}
	if verifySignature(githubSig("other", body), body, secret) {
		t.Error("wrong-secret signature accepted")
	}
	if verifySignature(githubSig(secret, body), []byte(`{}`), secret) {
		t.Error("tampered body accepted")
	}
	for _, h := range []string{"", "sha256=", "sha1=abc", "deadbeef"} {
		if verifySignature(h, body, secret) {
			t.Errorf("garbage header %q accepted", h)
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
		name, labels, err := mapEvent("github", tt.ghEvent, []byte(tt.body))
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

	if _, _, err := mapEvent("github", "", []byte(`{}`)); err == nil {
		t.Error("missing X-GitHub-Event accepted")
	}
	if _, _, err := mapEvent("github", "push", []byte(`not json`)); err == nil {
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

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandler(t *testing.T) {
	if _, err := Handler(Config{}, &fakePusher{}, nil); err == nil || !strings.Contains(err.Error(), "secret is required") {
		t.Fatalf("missing secret: err = %v", err)
	}
	push := &fakePusher{}
	h, err := Handler(Config{Secret: "gh_test"}, push, discardLogger()) // default prefix
	if err != nil {
		t.Fatal(err)
	}
	body := `{"action":"opened","repository":{"full_name":"acme/site"}}`
	req := httptest.NewRequest(http.MethodPost, "/anywhere", strings.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", githubSig("gh_test", []byte(body)))
	req.Header.Set("X-GitHub-Event", "pull_request")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("valid: status %d", w.Code)
	}
	push.fail = true
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/anywhere", strings.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", githubSig("gh_test", []byte(body)))
	req.Header.Set("X-GitHub-Event", "pull_request")
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadGateway {
		t.Errorf("push failure: status %d", w.Code)
	}
	if got := fmt.Sprint(push.events); got != "[github.pull_request.opened]" {
		t.Errorf("pushed events = %s", got)
	}
}
