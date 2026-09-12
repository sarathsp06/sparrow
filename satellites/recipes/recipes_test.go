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

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/satellites/recipes"
)

// paramToken matches the exact substitution token the CLI replaces at apply
// time: {{param "name"}} — plain string replacement, no template evaluation.
var paramToken = regexp.MustCompile(`\{\{param "([^"]+)"\}\}`)

// sampleContext mirrors what Sparrow passes to transform templates per
// delivery (see internal/webhooks/queue/webhook_worker.go).
func sampleContext() client.WebhookTemplateContext {
	return client.NewWebhookTemplateContext(
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

	engine := client.NewTemplateEngine()

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

			// Every destination except ntfy speaks JSON.
			if r.Name != "ntfy" && !json.Valid(out) {
				t.Errorf("rendered output is not valid JSON:\n%s", out)
			}
		})
	}
}
