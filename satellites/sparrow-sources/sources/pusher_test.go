package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestClientAutoCreatesEventType verifies the 404 -> create event type ->
// retry-once flow against a fake Sparrow server.
func TestClientAutoCreatesEventType(t *testing.T) {
	known := map[string]bool{}
	var pushes, creates int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/event-types":
			creates++
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			known[body.Name] = true
			w.WriteHeader(http.StatusCreated)
		case "/v1/consumers/default/events":
			pushes++
			if !known[r.URL.Query().Get("event")] {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewClient(SparrowConfig{URL: srv.URL, Consumer: "default"})
	err := c.PushEvent(context.Background(), "report.tick", json.RawMessage(`{}`), map[string]string{"source": "cron"}, "")
	if err != nil {
		t.Fatalf("PushEvent: %v", err)
	}
	if pushes != 2 || creates != 1 {
		t.Errorf("pushes=%d creates=%d, want 2 and 1", pushes, creates)
	}

	// second push of a now-known type: no extra create
	if err := c.PushEvent(context.Background(), "report.tick", json.RawMessage(`{}`), nil, ""); err != nil {
		t.Fatalf("second PushEvent: %v", err)
	}
	if creates != 1 {
		t.Errorf("creates=%d after second push, want 1", creates)
	}
}

// TestClientSendsIdempotencyKey checks the key reaches Sparrow on the push
// and on the retry after auto-creating the event type, and that an empty key
// is left out of the body.
func TestClientSendsIdempotencyKey(t *testing.T) {
	var keys []any
	created := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/event-types" {
			created = true
			w.WriteHeader(http.StatusCreated)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		keys = append(keys, body["idempotency_key"])
		if !created {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := NewClient(SparrowConfig{URL: srv.URL})
	if err := c.PushEvent(context.Background(), "github.push", json.RawMessage(`{}`), nil, "github:sha256:ab"); err != nil {
		t.Fatal(err)
	}
	if err := c.PushEvent(context.Background(), "github.push", json.RawMessage(`{}`), nil, ""); err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(keys); got != "[github:sha256:ab github:sha256:ab <nil>]" {
		t.Errorf("idempotency keys sent = %s", got)
	}
}
