package recipes_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/sarathsp06/sparrow/pkg/template"
	"github.com/sarathsp06/sparrow/satellites/recipes"
)

// paramToken matches the exact substitution token the CLI replaces at apply
// time: {{param "name"}} — plain string replacement, no template evaluation.
var paramToken = regexp.MustCompile(`\{\{param "([^"]+)"\}\}`)

// sampleContext mirrors what Sparrow passes to transform templates per
// delivery (see internal/webhooks/queue/webhook_worker.go).
func sampleContext() template.WebhookTemplateContext {
	return template.NewWebhookTemplateContext(
		"evt_0195c2a1",
		"user.created",
		time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC).Format(time.RFC3339),
		1,
		map[string]any{
			"user_id": 42,
			"email":   "ada@example.com",
			"note":    "quotes \" and\nnewlines must survive JSON encoding",
		},
	)
}

// substituteParams replaces every declared param's token with a dummy value,
// the same way the CLI does before registering the recipe.
func substituteParams(s string, params []recipes.Param) string {
	for _, p := range params {
		s = strings.ReplaceAll(s, `{{param "`+p.Name+`"}}`, "dummy-"+p.Name)
	}
	return s
}

// tokenRefs returns the param names referenced by {{param "x"}} tokens in s.
func tokenRefs(s string) []string {
	var names []string
	for _, m := range paramToken.FindAllStringSubmatch(s, -1) {
		names = append(names, m[1])
	}
	return names
}

func TestRecipes(t *testing.T) {
	files, err := filepath.Glob("*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 5 {
		t.Fatalf("expected at least 5 recipes, found %d", len(files))
	}

	embedded, err := recipes.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(embedded) != len(files) {
		t.Fatalf("embedded recipes = %d, want %d", len(embedded), len(files))
	}

	engine := template.NewTemplateEngine()

	for _, file := range files {
		t.Run(strings.TrimSuffix(file, ".yaml"), func(t *testing.T) {
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}

			var r recipes.Recipe
			if err := yaml.Unmarshal(raw, &r); err != nil {
				t.Fatalf("parse: %v", err)
			}

			// Required top-level fields.
			if r.Version != 1 {
				t.Errorf("version = %d, want 1", r.Version)
			}
			if want := strings.TrimSuffix(filepath.Base(file), ".yaml"); r.Name != want {
				t.Errorf("name = %q, want %q (filename)", r.Name, want)
			}
			if r.Description == "" {
				t.Error("description is empty")
			}
			if r.Webhook.URL == "" {
				t.Error("webhook.url is empty")
			}
			if r.Subscription.TransformTemplate == "" {
				t.Error("subscription.transform_template is empty")
			}

			// Every {{param "x"}} token must reference a declared param.
			declared := make(map[string]bool, len(r.Params))
			for _, p := range r.Params {
				declared[p.Name] = true
			}
			substitutable := []string{r.Webhook.URL, r.Subscription.TransformTemplate}
			for _, v := range r.Webhook.Headers {
				substitutable = append(substitutable, v)
			}
			for _, v := range r.Webhook.SecretHeaders {
				substitutable = append(substitutable, v)
			}
			for _, s := range substitutable {
				for _, name := range tokenRefs(s) {
					if !declared[name] {
						t.Errorf("token {{param %q}} references undeclared param", name)
					}
				}
			}

			// After substitution, the transform template must parse and
			// render with the standard delivery context.
			tmpl := substituteParams(r.Subscription.TransformTemplate, r.Params)
			out, err := engine.Execute(tmpl, sampleContext())
			if err != nil {
				t.Fatalf("render transform_template: %v", err)
			}

			// Every destination except twilio (form-encoded) speaks JSON.
			if r.Name != "twilio" && !json.Valid(out) {
				t.Errorf("rendered output is not valid JSON:\n%s", out)
			}
		})
	}
}

func TestSendGridActivationParams(t *testing.T) {
	raw, err := os.ReadFile("sendgrid.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var r recipes.Recipe
	if err := yaml.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}

	params := make(map[string]recipes.Param, len(r.Params))
	for _, p := range r.Params {
		params[p.Name] = p
	}

	if !params["api_key"].Secret || !params["api_key"].ActivationRequired {
		t.Fatalf("api_key must be marked as a secret activation param: %+v", params["api_key"])
	}
	fromEmail := params["from_email"]
	if fromEmail.Default != "alerts@example.com" || !fromEmail.ActivationRequired || !fromEmail.MustOverrideDefault {
		t.Fatalf("from_email must expose the placeholder without accepting it for activation: %+v", fromEmail)
	}
	if params["from_name"].Default != "Sparrow" {
		t.Fatalf("from_name default = %q, want Sparrow", params["from_name"].Default)
	}
}

// TestPagerdutyRecipe_Severity checks that the payload's own severity wins and
// the severity param is only a fallback.
func TestPagerdutyRecipe_Severity(t *testing.T) {
	raw, err := os.ReadFile("pagerduty.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var r recipes.Recipe
	if err := yaml.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	tmpl := substituteParams(r.Subscription.TransformTemplate, r.Params)
	engine := template.NewTemplateEngine()

	for payloadSev, want := range map[string]string{"critical": "critical", "": "dummy-severity"} {
		payload := map[string]any{"service": "api"}
		if payloadSev != "" {
			payload["severity"] = payloadSev
		}
		ctx := template.NewWebhookTemplateContext("evt_pd1", "incident.opened",
			time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC).Format(time.RFC3339), 1, payload)
		out, err := engine.Execute(tmpl, ctx)
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		var body struct {
			Payload struct{ Severity string }
		}
		if err := json.Unmarshal(out, &body); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, out)
		}
		if body.Payload.Severity != want {
			t.Errorf("severity = %q, want %q (payload severity %q)", body.Payload.Severity, want, payloadSev)
		}
	}
}
