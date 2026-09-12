package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// pushEnv points the CLI at srv with an API key and no config file.
func pushEnv(t *testing.T, srv *httptest.Server) {
	t.Helper()
	t.Setenv("SPARROW_CONFIG", filepath.Join(t.TempDir(), "missing.yaml"))
	t.Setenv("SPARROW_URL", srv.URL)
	t.Setenv("SPARROW_API_KEY", "sekrit")
	t.Setenv("SPARROW_NAMESPACE", "acme")
}

func TestPushRequestShape(t *testing.T) {
	type captured struct {
		method, path, query, apiKey string
		body                        []byte
	}
	var got captured
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = captured{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("X-API-Key"), body}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"event_id":"evt-123"}`)) //nolint:errcheck
	}))
	defer srv.Close()
	pushEnv(t, srv)

	var out bytes.Buffer
	err := runPush(context.Background(), []string{
		"-d", `{"amount":42}`, "-l", "env=prod", "--idempotency-key", "idem-1", "order.created",
	}, &out)
	if err != nil {
		t.Fatal(err)
	}

	if got.method != http.MethodPost {
		t.Errorf("method = %s", got.method)
	}
	if got.path != "/v1/namespaces/acme/events" {
		t.Errorf("path = %s", got.path)
	}
	if got.query != "event=order.created" {
		t.Errorf("query = %s", got.query)
	}
	if got.apiKey != "sekrit" {
		t.Errorf("X-API-Key = %q", got.apiKey)
	}
	var body pushBody
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}
	if body.Payload["amount"] != float64(42) || body.Labels["env"] != "prod" || body.IdempotencyKey != "idem-1" {
		t.Errorf("body = %+v", body)
	}
	if strings.TrimSpace(out.String()) != "evt-123" {
		t.Errorf("stdout = %q, want event id", out.String())
	}
}

func TestPushAutoCreatesEventType(t *testing.T) {
	var eventTypeCreated bool
	pushes := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/event-types" && r.Method == http.MethodPost:
			var body struct {
				Name string `json:"name"`
			}
			json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
			if body.Name != "order.created" {
				t.Errorf("event type name = %q", body.Name)
			}
			eventTypeCreated = true
			w.WriteHeader(http.StatusCreated)
		case strings.HasSuffix(r.URL.Path, "/events"):
			pushes++
			if !eventTypeCreated {
				http.Error(w, `{"detail":"event type not found"}`, http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"event_id":"evt-after-create"}`)) //nolint:errcheck
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	pushEnv(t, srv)

	var out bytes.Buffer
	if err := runPush(context.Background(), []string{"order.created"}, &out); err != nil {
		t.Fatal(err)
	}
	if !eventTypeCreated || pushes != 2 {
		t.Fatalf("created=%v pushes=%d, want auto-create then retry", eventTypeCreated, pushes)
	}
	if !strings.Contains(out.String(), "evt-after-create") {
		t.Errorf("stdout = %q", out.String())
	}
}
