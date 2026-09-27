package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccessLinkCreate(t *testing.T) {
	var method, path, query, apiKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, query, apiKey = r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("X-API-Key")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"token":"sal_v1.t","expires_at":"2030-01-01T00:00:00Z","path":"/#access=sal_v1.t"}`)) //nolint:errcheck
	}))
	defer srv.Close()
	pushEnv(t, srv)

	var out bytes.Buffer
	root := newRootCmd(&out)
	root.SetArgs([]string{"access-link", "create", "--ttl", "15m", "--ui-url", "https://ui.example.com/"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost || path != "/v1/access-links" || query != "ttl_seconds=900" || apiKey != "sekrit" {
		t.Fatalf("request = %s %s?%s key=%q", method, path, query, apiKey)
	}
	if first := strings.SplitN(out.String(), "\n", 2)[0]; first != "https://ui.example.com/#access=sal_v1.t" {
		t.Fatalf("link line = %q", first)
	}
}

func TestAccessLinkCreateDefaultsToServerURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"token":"t","expires_at":"2030-01-01T00:00:00Z","path":"/#access=t"}`)) //nolint:errcheck
	}))
	defer srv.Close()
	pushEnv(t, srv)

	var out bytes.Buffer
	root := newRootCmd(&out)
	root.SetArgs([]string{"access-link", "create", "-o", "json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"url": "`+srv.URL+`/#access=t"`) {
		t.Fatalf("output = %s", out.String())
	}
}

func TestAccessLinkCreateRejectsBadTTL(t *testing.T) {
	var out bytes.Buffer
	root := newRootCmd(&out)
	t.Setenv("SPARROW_URL", "http://127.0.0.1:1")
	root.SetArgs([]string{"access-link", "create", "--ttl", "48h"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "--ttl") {
		t.Fatalf("err = %v", err)
	}
}
