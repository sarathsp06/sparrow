package rest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	svcerrors "github.com/sarathsp06/sparrow/internal/errors"
	"github.com/sarathsp06/sparrow/internal/rest"
	"github.com/sarathsp06/sparrow/internal/webhooks"
)

// testEventService records test sends and answers with a fixed result.
type testEventService struct {
	webhooks.WebhookServiceInterface
	sent    []string
	payload map[string]any
	err     error
}

func (s *testEventService) SendTestEvent(_ context.Context, consumer, webhookID, subscriptionID string, payload map[string]any) (*webhooks.TestEventResult, error) {
	s.sent = append(s.sent, consumer+"/"+webhookID+"/"+subscriptionID)
	s.payload = payload
	if s.err != nil {
		return nil, s.err
	}
	return &webhooks.TestEventResult{StatusCode: 503, Latency: 42 * time.Millisecond, ErrorClass: "server_error"}, nil
}

func TestSendTestEvent(t *testing.T) {
	svc := &testEventService{}
	r := chi.NewRouter()
	rest.Mount(r, svc, rest.AccessDeps{}, rest.AIDeps{})
	path := "/v1/consumers/acme/webhooks/wh1:sendTestEvent"

	rec := do(r, http.MethodPost, path, `{"subscription_id":"sub1"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var out struct {
		StatusCode int    `json:"status_code"`
		LatencyMs  int64  `json:"latency_ms"`
		ErrorClass string `json:"error_class"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.StatusCode != 503 || out.LatencyMs != 42 || out.ErrorClass != "server_error" {
		t.Errorf("result = %+v", out)
	}
	if len(svc.sent) != 1 || svc.sent[0] != "acme/wh1/sub1" || svc.payload != nil {
		t.Errorf("sent = %v, payload %v", svc.sent, svc.payload)
	}

	do(r, http.MethodPost, path, `{"subscription_id":"sub1","payload":{"id":"ord_9"}}`)
	if svc.payload["id"] != "ord_9" {
		t.Errorf("payload = %v, want the request's", svc.payload)
	}

	svc.err = svcerrors.Error(svcerrors.ResourceExhausted, "slow down")
	if rec := do(r, http.MethodPost, path, `{"subscription_id":"sub1"}`); rec.Code != http.StatusTooManyRequests {
		t.Errorf("rate limited: status = %d, want 429", rec.Code)
	}
	if rec := do(r, http.MethodPost, path, `{}`); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("missing subscription_id: status = %d, want 422", rec.Code)
	}
}
