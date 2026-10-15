package rest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/sarathsp06/sparrow/internal/rest"
	"github.com/sarathsp06/sparrow/internal/webhooks"
)

// templateService records what testTemplate passes to the service.
type templateService struct {
	webhooks.WebhookServiceInterface
	payload map[string]any
	strict  bool
	called  bool
}

func (s *templateService) TestSubscriptionTemplate(_ context.Context, _, _, _ string, strict bool, payload map[string]any) (string, error) {
	s.called, s.strict, s.payload = true, strict, payload
	return `{"ok":true}`, nil
}

// testTemplate renders the sample when payload is omitted and the given
// payload otherwise, an empty one included.
func TestTestTemplate_Payload(t *testing.T) {
	cases := []struct {
		name, body string
		wantNil    bool
		wantKeys   int
	}{
		{"omitted uses the sample", `{"event_name":"order.created","template":"{}"}`, true, 0},
		{"given is passed through", `{"event_name":"order.created","template":"{}","payload":{"id":"ord_7","amount":4950}}`, false, 2},
		{"empty is used as given", `{"event_name":"order.created","template":"{}","payload":{}}`, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &templateService{}
			r := chi.NewRouter()
			rest.Mount(r, svc, rest.AccessDeps{}, rest.AIDeps{})

			rec := do(r, http.MethodPost, "/v1/subscriptions:testTemplate", tc.body)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
			}
			if !svc.called || !svc.strict {
				t.Fatalf("service called=%v strict=%v; want a strict render", svc.called, svc.strict)
			}
			if (svc.payload == nil) != tc.wantNil || len(svc.payload) != tc.wantKeys {
				t.Fatalf("payload = %#v, want nil=%v with %d keys", svc.payload, tc.wantNil, tc.wantKeys)
			}
			var out struct{ Rendered string }
			if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.Rendered != `{"ok":true}` {
				t.Fatalf("response %s (%v)", rec.Body, err)
			}
		})
	}
}

func TestTestTemplate_PayloadMustBeObject(t *testing.T) {
	svc := &templateService{}
	r := chi.NewRouter()
	rest.Mount(r, svc, rest.AccessDeps{}, rest.AIDeps{})

	rec := do(r, http.MethodPost, "/v1/subscriptions:testTemplate", `{"event_name":"order.created","template":"{}","payload":[1,2]}`)
	if rec.Code != http.StatusUnprocessableEntity && rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d for an array payload, body %s", rec.Code, rec.Body)
	}
	if svc.called {
		t.Fatal("service must not be called with an invalid payload")
	}
}
