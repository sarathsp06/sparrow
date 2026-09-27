package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type captured struct {
	method, path, query, apiKey string
	body                        map[string]any
}

// fakeServer answers every request with status/body and records the last one.
func fakeServer(t *testing.T, status int, body string) (*httptest.Server, *captured) {
	t.Helper()
	got := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		got.method, got.path, got.query, got.apiKey = r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("X-API-Key")
		got.body = nil
		_ = json.Unmarshal(raw, &got.body)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	pushEnv(t, srv)
	return srv, got
}

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	root := newRootCmd(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestParseDuration(t *testing.T) {
	for in, want := range map[string]time.Duration{"90d": 90 * 24 * time.Hour, "12h": 12 * time.Hour, "15m": 15 * time.Minute, "0d": 0} {
		if got, err := parseDuration(in); err != nil || got != want {
			t.Errorf("parseDuration(%q) = %v, %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "x", "-1d", "1.5d", "-2h"} {
		if _, err := parseDuration(bad); err == nil {
			t.Errorf("parseDuration(%q) accepted", bad)
		}
	}
}

func TestTokensCreate(t *testing.T) {
	_, got := fakeServer(t, http.StatusCreated, `{"secret":"sparrow_tk_abc","token":{"id":"tok_1","name":"ci","consumer":"acme","expires_at":"2030-01-01T00:00:00Z"}}`)
	out, err := runCLI(t, "tokens", "create", "--name", "ci", "--consumer", "acme", "--ttl", "30d")
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodPost || got.path != "/v1/tokens" || got.apiKey != "sekrit" {
		t.Fatalf("request = %+v", got)
	}
	if got.body["name"] != "ci" || got.body["consumer"] != "acme" || got.body["ttl_seconds"] != float64(30*24*3600) {
		t.Fatalf("body = %v", got.body)
	}
	if first := strings.SplitN(out, "\n", 2)[0]; first != "sparrow_tk_abc" {
		t.Fatalf("first line = %q (the secret should be alone on it for piping)", first)
	}
	if !strings.Contains(out, "portal:acme token tok_1") {
		t.Fatalf("output = %s", out)
	}
}

func TestTokensCreateTenantWideOmitsOptionalFields(t *testing.T) {
	_, got := fakeServer(t, http.StatusCreated, `{"secret":"sparrow_tk_abc","token":{"id":"tok_1","name":"ci","consumer":null,"expires_at":null}}`)
	out, err := runCLI(t, "tokens", "create", "--name", "ci")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.body["consumer"]; ok {
		t.Fatalf("consumer sent for tenant-wide token: %v", got.body)
	}
	if _, ok := got.body["ttl_seconds"]; ok {
		t.Fatalf("ttl sent without --ttl: %v", got.body)
	}
	if !strings.Contains(out, "full token tok_1 (ci), expires: never") {
		t.Fatalf("output = %s", out)
	}
}

func TestTokensCreateNeverExpires(t *testing.T) {
	_, got := fakeServer(t, http.StatusCreated, `{"secret":"sparrow_tk_abc","token":{"id":"tok_1","name":"bot","consumer":null,"expires_at":null}}`)
	if _, err := runCLI(t, "tokens", "create", "--name", "bot", "--ttl", "never"); err != nil {
		t.Fatal(err)
	}
	if got.body["never_expires"] != true {
		t.Fatalf("--ttl never did not send never_expires: %v", got.body)
	}
	if _, ok := got.body["ttl_seconds"]; ok {
		t.Fatalf("ttl_seconds sent with --ttl never: %v", got.body)
	}
}

func TestInviteTokenNeverExpires(t *testing.T) {
	_, got := fakeServer(t, http.StatusCreated, `{"secret":"sparrow_inv_x","path":"/#invite=sparrow_inv_x","invite":{"id":"inv_1","name":"ops","expires_at":"2030-01-01T00:00:00Z"}}`)
	if _, err := runCLI(t, "invite", "ops", "--token-ttl", "never", "-o", "json"); err != nil {
		t.Fatal(err)
	}
	if got.body["token_never_expires"] != true {
		t.Fatalf("--token-ttl never did not send token_never_expires: %v", got.body)
	}
}

func TestTokensCreateRequiresName(t *testing.T) {
	fakeServer(t, http.StatusCreated, `{}`)
	if _, err := runCLI(t, "tokens", "create"); err == nil {
		t.Fatal("tokens create without --name succeeded")
	}
}

func TestTokensListAndRevoke(t *testing.T) {
	_, got := fakeServer(t, http.StatusOK, `{"items":[{"id":"tok_1","name":"alice","consumer":null,"status":"active","created_by":"master key"}]}`)
	out, err := runCLI(t, "tokens", "list", "--all")
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v1/tokens" || got.query != "include_inactive=true" || !strings.Contains(out, "alice") || !strings.Contains(out, "master key") {
		t.Fatalf("list: %+v\n%s", got, out)
	}

	_, got = fakeServer(t, http.StatusNoContent, ``)
	if out, err := runCLI(t, "tokens", "revoke", "tok_1"); err != nil || !strings.Contains(out, "revoked tok_1") {
		t.Fatalf("revoke: %v %s", err, out)
	}
	if got.method != http.MethodDelete || got.path != "/v1/tokens/tok_1" {
		t.Fatalf("revoke request = %+v", got)
	}
}

func TestInviteBuildsLinkOnUIURL(t *testing.T) {
	_, got := fakeServer(t, http.StatusCreated, `{"secret":"sparrow_inv_x","path":"/#invite=sparrow_inv_x","invite":{"id":"inv_1","name":"bob","consumer":null,"expires_at":"2030-01-01T00:00:00Z"}}`)
	out, err := runCLI(t, "invite", "bob", "--ttl", "15m", "--ui-url", "https://ui.example.com/")
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v1/invites" || got.body["name"] != "bob" || got.body["ttl_seconds"] != float64(900) {
		t.Fatalf("request = %+v", got)
	}
	if first := strings.SplitN(out, "\n", 2)[0]; first != "https://ui.example.com/#invite=sparrow_inv_x" {
		t.Fatalf("link = %q", first)
	}
	if !strings.Contains(out, "single use, for bob (full access)") {
		t.Fatalf("output = %s", out)
	}
}

func TestInviteDefaultsToServerURLAndJSON(t *testing.T) {
	srv, got := fakeServer(t, http.StatusCreated, `{"secret":"s","path":"/portal#invite=s","invite":{"id":"inv_1","name":"acme","consumer":"acme","expires_at":"2030-01-01T00:00:00Z"}}`)
	out, err := runCLI(t, "invite", "acme", "--consumer", "acme", "--token-ttl", "7d", "-o", "json")
	if err != nil {
		t.Fatal(err)
	}
	if got.body["consumer"] != "acme" || got.body["token_ttl_seconds"] != float64(7*24*3600) {
		t.Fatalf("body = %v", got.body)
	}
	if !strings.Contains(out, `"url": "`+srv.URL+`/portal#invite=s"`) {
		t.Fatalf("output = %s", out)
	}
}

func TestInvitesListAndCancel(t *testing.T) {
	_, got := fakeServer(t, http.StatusOK, `{"items":[{"id":"inv_1","name":"bob","consumer":null,"status":"pending","created_by":"alice","expires_at":"2030-01-01T00:00:00Z"}]}`)
	out, err := runCLI(t, "invites", "list")
	if err != nil || got.query != "include_inactive=false" || !strings.Contains(out, "bob") {
		t.Fatalf("list: %v %+v %s", err, got, out)
	}
	_, got = fakeServer(t, http.StatusNoContent, ``)
	if out, err := runCLI(t, "invites", "cancel", "inv_1"); err != nil || !strings.Contains(out, "cancelled inv_1") || got.path != "/v1/invites/inv_1" || got.method != http.MethodDelete {
		t.Fatalf("cancel: %v %+v %s", err, got, out)
	}
}

func TestAccessCommandsSurfaceServerErrors(t *testing.T) {
	fakeServer(t, http.StatusBadRequest, `{"detail":"consumer tokens can live at most 720h0m0s"}`)
	_, err := runCLI(t, "tokens", "create", "--name", "x", "--consumer", "acme", "--ttl", "90d")
	if err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("err = %v", err)
	}
}
