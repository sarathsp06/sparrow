package rest_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/sarathsp06/sparrow/internal/ai"
	"github.com/sarathsp06/sparrow/internal/rest"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// slowDrafter answers after delay, or gives up when ctx ends first.
type slowDrafter struct{ delay time.Duration }

func (s slowDrafter) DraftTemplate(ctx context.Context, _ ai.Request) (*ai.Result, error) {
	select {
	case <-time.After(s.delay):
		return &ai.Result{Template: `{"id":"{{ .payload.id }}"}`, Attempts: 1}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (slowDrafter) Model() string    { return "slow" }
func (slowDrafter) Provider() string { return "anthropic" }

// draftOverServer posts a draft through a real http.Server whose
// WriteTimeout is shorter than the draft, behind the same response-writer
// wrapping middleware as in production.
func draftOverServer(t *testing.T, d rest.TemplateDrafter, timeout time.Duration) (int, string) {
	t.Helper()
	r := chi.NewRouter()
	r.Use(otelhttp.NewMiddleware("test"))
	svc := eventOnlyService{events: map[string]*store.EventRegistration{
		"order.created": {Name: "order.created", Schema: map[string]any{"type": "object"}, SamplePayload: map[string]any{"id": "ord_1"}},
	}}
	rest.Mount(r, svc, rest.AccessDeps{}, rest.AIDeps{Drafter: d, Timeout: timeout})
	srv := httptest.NewUnstartedServer(r)
	srv.Config.WriteTimeout = 200 * time.Millisecond
	srv.Start()
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/v1/subscriptions:draftTemplate", "application/json",
		strings.NewReader(`{"event_name":"order.created","instructions":"just the id"}`))
	if err != nil {
		t.Fatalf("request failed (connection dropped by the write timeout?): %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestDraftTemplate_OutlivesServerWriteTimeout(t *testing.T) {
	code, body := draftOverServer(t, slowDrafter{delay: 600 * time.Millisecond}, 5*time.Second)
	if code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", code, body)
	}
}

func TestDraftTemplate_TimeoutIs504(t *testing.T) {
	code, body := draftOverServer(t, slowDrafter{delay: time.Minute}, 300*time.Millisecond)
	if code != http.StatusGatewayTimeout {
		t.Fatalf("status %d, want 504: %s", code, body)
	}
	if !strings.Contains(body, "SPARROW_AI_TIMEOUT") {
		t.Fatalf("504 does not say how to raise the limit: %s", body)
	}
}
