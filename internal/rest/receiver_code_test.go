package rest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	svcerrors "github.com/sarathsp06/sparrow/internal/errors"
	"github.com/sarathsp06/sparrow/internal/rest"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

const receiverPublicKey = "3b6a27bcceb6a42d62a3a8d02a6f0d73653215771de243a63ac048a18b59da29"

// receiverService serves one webhook with two subscriptions.
type receiverService struct {
	webhooks.WebhookServiceInterface
	reg     *store.WebhookRegistration
	subs    []*store.EventSubscription
	sendErr error
	sent    []string
}

func newReceiverService(scheme store.SignatureType) *receiverService {
	reg := &store.WebhookRegistration{ID: uuid.New(), Consumer: "acme", URL: "https://example.com/hook",
		SignatureType: scheme, Headers: map[string]string{"Content-Type": "application/json"}}
	return &receiverService{
		reg: reg,
		subs: []*store.EventSubscription{
			{ID: uuid.New(), WebhookID: reg.ID, EventName: "order.paid", Consumer: "acme"},
			{ID: uuid.New(), WebhookID: reg.ID, EventName: "order.note", Consumer: "acme",
				TransformEnabled: true, TransformTemplate: `Order {{.payload.id}}`,
				Headers: map[string]string{"Content-Type": "text/plain"}},
		},
	}
}

func (s *receiverService) ListWebhooks(_ context.Context, consumer, webhookID, _ string, _ bool, _ string, _, _ int32) ([]*store.WebhookRegistration, int32, error) {
	if consumer != s.reg.Consumer || webhookID != s.reg.ID.String() {
		return nil, 0, nil
	}
	return []*store.WebhookRegistration{s.reg}, 1, nil
}

func (s *receiverService) ListSubscriptions(_ context.Context, _, _, _ string, _, _ int32) ([]*store.EventSubscription, int32, error) {
	return s.subs, 0, nil
}

func (s *receiverService) GetEvent(_ context.Context, name string) (*store.EventRegistration, error) {
	if name == "order.note" {
		return nil, svcerrors.Error(svcerrors.NotFound, "event type not found")
	}
	return &store.EventRegistration{Name: name, Version: 2,
		Schema:        store.JSONMap{"type": "object", "required": []any{"id"}, "properties": map[string]any{"id": map[string]any{"type": "string"}}},
		SamplePayload: store.JSONMap{"id": "ord_1"}}, nil
}

func (s *receiverService) WebhookSigningPublicKeyHex([]byte) string { return receiverPublicKey }

func (s *receiverService) AlertDeliveryConfigured(context.Context) (bool, error) { return false, nil }

func (s *receiverService) SendTestEvent(_ context.Context, consumer, webhookID, subscriptionID string) (*webhooks.TestEventResult, error) {
	s.sent = append(s.sent, consumer+"/"+webhookID+"/"+subscriptionID)
	if s.sendErr != nil {
		return nil, s.sendErr
	}
	return &webhooks.TestEventResult{StatusCode: 503, Latency: 42 * time.Millisecond, ResponseExcerpt: "down", ErrorClass: "server_error"}, nil
}

func receiverRouter(svc *receiverService) http.Handler {
	r := chi.NewRouter()
	rest.Mount(r, svc, rest.AccessDeps{}, rest.AIDeps{})
	return r
}

type generated struct {
	Files []struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Mode    uint32 `json:"mode"`
	} `json:"files"`
	Spec struct {
		Target        string `json:"target"`
		Subscriptions []struct {
			ID           string `json:"id"`
			EventType    string `json:"event_type"`
			EventVersion int    `json:"event_version"`
			Format       string `json:"format"`
			Model        string `json:"model"`
			ModelSource  string `json:"model_source"`
		} `json:"subscriptions"`
		Webhook struct {
			PublicKey string `json:"public_key"`
		} `json:"webhook"`
	} `json:"spec"`
	Run  string `json:"run"`
	Test string `json:"test"`
}

func TestGenerateReceiverCode(t *testing.T) {
	svc := newReceiverService(store.SignatureTypeHMAC)
	h := receiverRouter(svc)
	path := "/v1/consumers/acme/webhooks/" + svc.reg.ID.String() + ":generateReceiverCode"

	rec := do(h, http.MethodPost, path, `{"language":"go","framework":"nethttp"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var out generated
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	for _, f := range out.Files {
		paths[f.Path] = true
	}
	for _, want := range []string{"main.go", "receiver.go", "handlers.go", "models.go", "signature/signature.go", "testdata/spec.json", "README.md"} {
		if !paths[want] {
			t.Errorf("missing %s in %v", want, paths)
		}
	}
	if out.Spec.Target != "go-nethttp" || len(out.Spec.Subscriptions) != 2 || out.Test != "go test ./..." || !strings.Contains(out.Run, "SPARROW_WEBHOOK_SECRET") {
		t.Fatalf("spec/run/test = %+v %q %q", out.Spec, out.Run, out.Test)
	}
	byType := map[string]int{}
	for i, s := range out.Spec.Subscriptions {
		byType[s.EventType] = i
	}
	paid := out.Spec.Subscriptions[byType["order.paid"]]
	if paid.Model != "envelope" || paid.ModelSource != "schema" || paid.EventVersion != 2 {
		t.Errorf("order.paid = %+v, want a schema-typed envelope at version 2", paid)
	}
	// The subscription's Content-Type overrides the webhook's; an event type
	// that cannot be read leaves the handler untyped instead of failing.
	if note := out.Spec.Subscriptions[byType["order.note"]]; note.Format != "text" || note.Model != "raw" {
		t.Errorf("order.note = %+v, want raw text", note)
	}

	// Deterministic: the same request gives the same body.
	if again := do(h, http.MethodPost, path, `{"language":"go","framework":"nethttp"}`); again.Body.String() != rec.Body.String() {
		t.Error("second generation differs")
	}

	// A subset of subscriptions.
	rec = do(h, http.MethodPost, path, `{"language":"python","framework":"fastapi","subscription_ids":["`+svc.subs[1].ID.String()+`"]}`)
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || rec.Code != http.StatusOK || len(out.Spec.Subscriptions) != 1 {
		t.Fatalf("subset: %d %s", rec.Code, rec.Body)
	}
}

func TestGenerateReceiverCodeEd25519EmbedsThePublicKeyOnly(t *testing.T) {
	svc := newReceiverService(store.SignatureTypeEd25519)
	rec := do(receiverRouter(svc), http.MethodPost, "/v1/consumers/acme/webhooks/"+svc.reg.ID.String()+":generateReceiverCode", `{"language":"node","framework":"express"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var out generated
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Spec.Webhook.PublicKey != receiverPublicKey || strings.Contains(out.Run, "SECRET") {
		t.Errorf("public key %q, run %q", out.Spec.Webhook.PublicKey, out.Run)
	}
}

func TestGenerateReceiverCodeErrors(t *testing.T) {
	svc := newReceiverService(store.SignatureTypeHMAC)
	h := receiverRouter(svc)
	path := "/v1/consumers/acme/webhooks/" + svc.reg.ID.String() + ":generateReceiverCode"
	cases := []struct {
		name, path, body string
		want             int
	}{
		{"unknown target", path, `{"language":"cobol","framework":"cics"}`, http.StatusBadRequest},
		{"foreign subscription", path, `{"language":"go","framework":"nethttp","subscription_ids":["` + uuid.NewString() + `"]}`, http.StatusBadRequest},
		{"missing framework", path, `{"language":"go"}`, http.StatusUnprocessableEntity},
		{"unknown webhook", "/v1/consumers/acme/webhooks/" + uuid.NewString() + ":generateReceiverCode", `{"language":"go","framework":"nethttp"}`, http.StatusNotFound},
		{"other consumer", "/v1/consumers/globex/webhooks/" + svc.reg.ID.String() + ":generateReceiverCode", `{"language":"go","framework":"nethttp"}`, http.StatusNotFound},
	}
	for _, c := range cases {
		if rec := do(h, http.MethodPost, c.path, c.body); rec.Code != c.want {
			t.Errorf("%s: status = %d, want %d (%s)", c.name, rec.Code, c.want, rec.Body)
		}
	}

	svc.subs = nil
	if rec := do(h, http.MethodPost, path, `{"language":"go","framework":"nethttp"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("no subscriptions: status = %d, want 400", rec.Code)
	}
}

func TestSendTestEvent(t *testing.T) {
	svc := newReceiverService(store.SignatureTypeHMAC)
	h := receiverRouter(svc)
	path := "/v1/consumers/acme/webhooks/" + svc.reg.ID.String() + ":sendTestEvent"
	subID := svc.subs[0].ID.String()

	rec := do(h, http.MethodPost, path, `{"subscription_id":"`+subID+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var out struct {
		StatusCode      int    `json:"status_code"`
		LatencyMs       int64  `json:"latency_ms"`
		ResponseExcerpt string `json:"response_excerpt"`
		ErrorClass      string `json:"error_class"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.StatusCode != 503 || out.LatencyMs != 42 || out.ResponseExcerpt != "down" || out.ErrorClass != "server_error" {
		t.Errorf("result = %+v", out)
	}
	if len(svc.sent) != 1 || svc.sent[0] != "acme/"+svc.reg.ID.String()+"/"+subID {
		t.Errorf("sent = %v", svc.sent)
	}

	svc.sendErr = svcerrors.Error(svcerrors.ResourceExhausted, "slow down")
	if rec := do(h, http.MethodPost, path, `{"subscription_id":"`+subID+`"}`); rec.Code != http.StatusTooManyRequests {
		t.Errorf("rate limited: status = %d, want 429", rec.Code)
	}
	if rec := do(h, http.MethodPost, path, `{}`); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("missing subscription_id: status = %d, want 422", rec.Code)
	}
}

func TestCapabilitiesListReceiverTargets(t *testing.T) {
	rec := do(receiverRouter(newReceiverService(store.SignatureTypeHMAC)), http.MethodGet, "/v1/capabilities", "")
	var caps struct {
		ReceiverCode struct {
			Targets []struct {
				ID, Language, Framework string
			} `json:"targets"`
		} `json:"receiver_code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &caps); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, tg := range caps.ReceiverCode.Targets {
		ids = append(ids, tg.ID)
	}
	if strings.Join(ids, ",") != "go-nethttp,node-express,python-fastapi" {
		t.Errorf("targets = %v", ids)
	}
}
