package recipes_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/sarathsp06/sparrow/pkg/template"
	"github.com/sarathsp06/sparrow/satellites/recipes"
)

// loadSendgridTemplate reads and param-substitutes sendgrid.yaml's transform
// template, the same way the CLI does at apply time.
func loadSendgridTemplate(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("sendgrid.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var r recipes.Recipe
	if err := yaml.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	return substituteParams(r.Subscription.TransformTemplate, r.Params)
}

// TestSendgridRecipe_HealthChanged renders sendgrid.yaml against a realistic
// sparrow.webhook.health_changed payload (see internal/webhooks/queue/system_events.go)
// and checks the SendGrid v3 Mail Send request it produces: one personalization
// per opted-in recipient, and a subject naming the new health state.
func TestSendgridRecipe_HealthChanged(t *testing.T) {
	tmpl := loadSendgridTemplate(t)
	ctx := template.NewWebhookTemplateContext(
		"evt_health1", "sparrow.webhook.health_changed",
		time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC).Format(time.RFC3339), 1,
		map[string]any{
			"webhook_id": "wh_123",
			"consumer":   "acme",
			"url":        "https://acme.example.com/hooks",
			"old_health": "healthy",
			"new_health": "degraded",
			"alert_recipients": []map[string]string{
				{"email": "ops@acme.example.com"},
				{"email": "oncall@acme.example.com"},
			},
		},
	)

	out, err := template.NewTemplateEngine().Execute(tmpl, ctx)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !json.Valid(out) {
		t.Fatalf("rendered output is not valid JSON:\n%s", out)
	}

	var body struct {
		Personalizations []struct {
			To []struct{ Email string }
		}
		Subject string
	}
	if err := json.Unmarshal(out, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Personalizations) != 2 {
		t.Fatalf("expected 2 personalizations (one per recipient), got %d", len(body.Personalizations))
	}
	if body.Personalizations[0].To[0].Email != "ops@acme.example.com" ||
		body.Personalizations[1].To[0].Email != "oncall@acme.example.com" {
		t.Errorf("unexpected recipient emails: %+v", body.Personalizations)
	}
	if !strings.Contains(body.Subject, "degraded") {
		t.Errorf("expected subject to mention the new health state, got %q", body.Subject)
	}
}

// TestSendgridRecipe_DeliveryFailed renders sendgrid.yaml against a realistic
// sparrow.webhook.delivery_failed payload and checks the same contract.
func TestSendgridRecipe_DeliveryFailed(t *testing.T) {
	tmpl := loadSendgridTemplate(t)
	ctx := template.NewWebhookTemplateContext(
		"evt_fail1", "sparrow.webhook.delivery_failed",
		time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC).Format(time.RFC3339), 1,
		map[string]any{
			"webhook_id":     "wh_123",
			"consumer":       "acme",
			"url":            "https://acme.example.com/hooks",
			"delivery_id":    "del_456",
			"event_id":       "evt_789",
			"attempt":        5,
			"error_category": "server_error",
			"error_message":  "HTTP 500: Internal Server Error",
			"alert_recipients": []map[string]string{
				{"email": "ops@acme.example.com"},
			},
		},
	)

	out, err := template.NewTemplateEngine().Execute(tmpl, ctx)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !json.Valid(out) {
		t.Fatalf("rendered output is not valid JSON:\n%s", out)
	}

	var body struct {
		Personalizations []struct {
			To []struct{ Email string }
		}
		Subject string
		Content []struct{ Value string }
	}
	if err := json.Unmarshal(out, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Personalizations) != 1 {
		t.Fatalf("expected 1 personalization, got %d", len(body.Personalizations))
	}
	if !strings.Contains(body.Subject, "failed") {
		t.Errorf("expected subject to mention delivery failure, got %q", body.Subject)
	}
	if len(body.Content) != 1 || !strings.Contains(body.Content[0].Value, "500") {
		t.Errorf("expected content body to mention the error, got %+v", body.Content)
	}
}
