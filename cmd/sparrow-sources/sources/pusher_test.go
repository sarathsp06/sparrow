package sources

import (
	"context"
	"encoding/json"
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
		case "/v1/namespaces/default/events":
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

	c := NewClient(SparrowConfig{URL: srv.URL, Namespace: "default"})
	err := c.PushEvent(context.Background(), "report.tick", json.RawMessage(`{}`), map[string]string{"source": "cron"})
	if err != nil {
		t.Fatalf("PushEvent: %v", err)
	}
	if pushes != 2 || creates != 1 {
		t.Errorf("pushes=%d creates=%d, want 2 and 1", pushes, creates)
	}

	// second push of a now-known type: no extra create
	if err := c.PushEvent(context.Background(), "report.tick", json.RawMessage(`{}`), nil); err != nil {
		t.Fatalf("second PushEvent: %v", err)
	}
	if creates != 1 {
		t.Errorf("creates=%d after second push, want 1", creates)
	}
}
