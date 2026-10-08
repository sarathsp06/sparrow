package rest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/sarathsp06/sparrow/internal/ai"
	"github.com/sarathsp06/sparrow/internal/rest"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// eventOnlyService implements just GetEvent; every other method panics via
// the nil embedded interface, which is what we want in these tests.
type eventOnlyService struct {
	webhooks.WebhookServiceInterface
	events        map[string]*store.EventRegistration
	alertDelivery bool
}

func (s eventOnlyService) AlertDeliveryConfigured(context.Context) (bool, error) {
	return s.alertDelivery, nil
}

func (s eventOnlyService) GetEvent(_ context.Context, name string) (*store.EventRegistration, error) {
	return s.events[name], nil
}

func (s eventOnlyService) GetTemplateFunctions() []webhooks.TemplateFunctionInfo {
	return []webhooks.TemplateFunctionInfo{{Name: "json", Description: "# json\n\nConverts a value to JSON."}}
}

type stubDrafter struct {
	got ai.Request
	res *ai.Result
	err error
}

func (s *stubDrafter) DraftTemplate(_ context.Context, req ai.Request) (*ai.Result, error) {
	s.got = req
	return s.res, s.err
}
func (s *stubDrafter) Model() string    { return "stub-model" }
func (s *stubDrafter) Provider() string { return "anthropic" }

func aiRouter(t *testing.T, drafter rest.TemplateDrafter) http.Handler {
	t.Helper()
	r := chi.NewRouter()
	svc := eventOnlyService{events: map[string]*store.EventRegistration{
		"order.created": {Name: "order.created", Schema: map[string]any{"type": "object"}, SamplePayload: map[string]any{"id": "ord_1"}},
	}}
	rest.Mount(r, svc, rest.AccessDeps{}, rest.AIDeps{Drafter: drafter})
	return r
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCapabilities_ReflectsDrafter(t *testing.T) {
	var caps struct {
		AIDrafting struct {
			Enabled bool   `json:"enabled"`
			Model   string `json:"model"`
		} `json:"ai_drafting"`
	}

	rec := do(aiRouter(t, nil), http.MethodGet, "/v1/capabilities", "")
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &caps)
	if caps.AIDrafting.Enabled || caps.AIDrafting.Model != "" {
		t.Fatalf("disabled server reports %+v", caps)
	}

	rec = do(aiRouter(t, &stubDrafter{}), http.MethodGet, "/v1/capabilities", "")
	_ = json.Unmarshal(rec.Body.Bytes(), &caps)
	if !caps.AIDrafting.Enabled || caps.AIDrafting.Model != "stub-model" {
		t.Fatalf("enabled server reports %+v", caps)
	}
}

func TestCapabilities_ReportsAlertDelivery(t *testing.T) {
	for _, configured := range []bool{false, true} {
		r := chi.NewRouter()
		rest.Mount(r, eventOnlyService{alertDelivery: configured}, rest.AccessDeps{}, rest.AIDeps{})
		rec := do(r, http.MethodGet, "/v1/capabilities", "")
		if rec.Code != 200 {
			t.Fatalf("status %d: %s", rec.Code, rec.Body)
		}
		var caps struct {
			AlertDelivery struct {
				Configured bool `json:"configured"`
			} `json:"alert_delivery"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &caps)
		if caps.AlertDelivery.Configured != configured {
			t.Fatalf("alert_delivery.configured = %v, want %v", caps.AlertDelivery.Configured, configured)
		}
	}
}

func TestDraftTemplate_DisabledIs503(t *testing.T) {
	rec := do(aiRouter(t, nil), http.MethodPost, "/v1/subscriptions:draftTemplate",
		`{"event_name":"order.created","instructions":"just the id"}`)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, want 503: %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "SPARROW_AI_API_KEY") {
		t.Fatalf("body should tell the operator how to enable it: %s", rec.Body)
	}
}

func TestDraftTemplate_GroundsRequestAndReturnsDraft(t *testing.T) {
	d := &stubDrafter{res: &ai.Result{Template: "{{ .payload.id | json }}", Rendered: `"ord_1"`, Notes: "used id", Attempts: 1, Model: "stub-model"}}
	rec := do(aiRouter(t, d), http.MethodPost, "/v1/subscriptions:draftTemplate",
		`{"event_name":"order.created","instructions":"just the id","recipe":"slack","target_example":"{\"x\":1}","current_template":"old","docs_url":"https://docs.example.com/hooks","sample_payload":{"id":"ord_9"}}`)
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if d.got.EventName != "order.created" || d.got.SamplePayload["id"] != "ord_9" || d.got.Schema["type"] != "object" {
		t.Fatalf("event not grounded (provided sample should win): %+v", d.got)
	}
	if d.got.DocsURL != "https://docs.example.com/hooks" {
		t.Fatalf("docs_url not passed: %+v", d.got)
	}
	if !strings.Contains(rec.Body.String(), `"sample_source":"provided"`) {
		t.Fatalf("sample_source should be provided: %s", rec.Body)
	}

	if d.got.RecipeName != "slack" || !strings.Contains(d.got.RecipeTemplate, "blocks") {
		t.Fatalf("recipe not resolved: %+v", d.got)
	}
	if d.got.TargetExample != `{"x":1}` || d.got.CurrentTemplate != "old" || d.got.Instructions != "just the id" {
		t.Fatalf("optional inputs not passed: %+v", d.got)
	}
	var out struct {
		Template, Rendered, Notes, Model string
		Attempts                         int
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Template != d.res.Template || out.Rendered != d.res.Rendered || out.Notes != "used id" || out.Attempts != 1 || out.Model != "stub-model" {
		t.Fatalf("unexpected body: %s", rec.Body)
	}
	// Without sample_payload the registered sample grounds the draft.
	rec = do(aiRouter(t, d), http.MethodPost, "/v1/subscriptions:draftTemplate", `{"event_name":"order.created","instructions":"x"}`)
	if rec.Code != 200 || d.got.SamplePayload["id"] != "ord_1" || !strings.Contains(rec.Body.String(), `"sample_source":"registered"`) {
		t.Fatalf("registered sample not used: %d %s %+v", rec.Code, rec.Body, d.got)
	}
}

func TestDraftTemplatePrompt_WorksWithoutAProvider(t *testing.T) {
	rec := do(aiRouter(t, nil), http.MethodPost, "/v1/subscriptions:draftTemplatePrompt",
		`{"event_name":"order.created","instructions":"short Slack text","recipe":"slack","sample_payload":{"id":"ord_9","email":"a@example.com"}}`)
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var out struct {
		Prompt       string `json:"prompt"`
		SampleSource string `json:"sample_source"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	for _, want := range []string{"order.created", `"id": "ord_9"`, "a@example.com", "short Slack text", "Destination recipe: slack", "### json", "fenced code block", "=== Your task ==="} {
		if !strings.Contains(out.Prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
	if out.SampleSource != "provided" {
		t.Fatalf("sample_source = %q", out.SampleSource)
	}

	// Capabilities tell the UI to show the copy-prompt button instead.
	rec = do(aiRouter(t, nil), http.MethodGet, "/v1/capabilities", "")
	if !strings.Contains(rec.Body.String(), `"prompt_only":true`) {
		t.Fatalf("capabilities should report prompt_only: %s", rec.Body)
	}

	// docs_url without a fetcher is refused clearly; unknown event is 404.
	rec = do(aiRouter(t, nil), http.MethodPost, "/v1/subscriptions:draftTemplatePrompt",
		`{"event_name":"order.created","instructions":"x","docs_url":"https://example.com/docs"}`)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "docs_url") {
		t.Fatalf("docs_url without fetcher: %d %s", rec.Code, rec.Body)
	}
	rec = do(aiRouter(t, nil), http.MethodPost, "/v1/subscriptions:draftTemplatePrompt", `{"event_name":"nope","instructions":"x"}`)
	if rec.Code != 404 {
		t.Fatalf("unknown event: %d", rec.Code)
	}
}

func TestDraftTemplate_NotFound(t *testing.T) {
	d := &stubDrafter{res: &ai.Result{Template: "x"}}
	rec := do(aiRouter(t, d), http.MethodPost, "/v1/subscriptions:draftTemplate",
		`{"event_name":"nope","instructions":"x"}`)
	if rec.Code != 404 {
		t.Fatalf("unknown event: status %d, want 404: %s", rec.Code, rec.Body)
	}
	rec = do(aiRouter(t, d), http.MethodPost, "/v1/subscriptions:draftTemplate",
		`{"event_name":"order.created","instructions":"x","recipe":"no-such-recipe"}`)
	if rec.Code != 404 {
		t.Fatalf("unknown recipe: status %d, want 404: %s", rec.Code, rec.Body)
	}
	rec = do(aiRouter(t, d), http.MethodPost, "/v1/subscriptions:draftTemplate",
		`{"event_name":"order.created"}`)
	if rec.Code != 422 && rec.Code != 400 {
		t.Fatalf("missing instructions: status %d, want validation error: %s", rec.Code, rec.Body)
	}
}
