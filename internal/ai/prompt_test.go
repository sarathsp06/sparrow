package ai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sarathsp06/sparrow/pkg/template"
)

func TestPrompts_HaveThreePartsInOrder(t *testing.T) {
	p := NewPromptBuilder([]HelperFunc{{Name: "json", Description: "doc"}})
	sys := p.systemPrompt(false)
	for _, want := range []string{"# Part 1", ".payload", "missingkey=error", "float64", "dig", "### json", "JSON object"} {
		if !strings.Contains(sys, want) {
			t.Fatalf("system prompt missing %q:\n%s", want, sys)
		}
	}
	if strings.Contains(sys, "FENCE") {
		t.Fatal("FENCE placeholder left in the system prompt")
	}

	req := Request{
		DocsURL:       "https://docs.example.com/hooks",
		EventName:     "order.created",
		Schema:        map[string]any{"type": "object", "required": []any{"id"}, "properties": map[string]any{"id": map[string]any{"type": "string"}}},
		SamplePayload: map[string]any{"id": "1"},
		Instructions:  "do it",
		Recipe: &Recipe{
			Name: "pagerduty", Description: "Trigger alerts", URL: "https://events.pagerduty.com/v2/enqueue",
			ContentType: "application/json", Guidance: "Summary names what is broken and where.",
			Params:   []RecipeParam{{Name: "routing_key", Prompt: "Routing key", Help: "From the integration page.", Example: "R0ABC", Enum: []string{"a", "b"}, Default: "a"}},
			Template: "{\"routing_key\": \"{{param \"routing_key\"}}\", \"text\": {{ printf \"```%s```\" .event_name | json }}}",
		},
		TargetExample:   `{"text": "..."}`,
		CurrentTemplate: `{{ .payload.id }}`,
	}
	user := p.userPrompt(req, "POST a JSON body with title and body fields")
	order := []string{"# Part 2", "pagerduty", "events.pagerduty.com", "application/json", "What a good message looks like", "Summary names what is broken", "routing_key: Routing key. From the integration page (one of a, b; e.g. R0ABC; default a)",
		"keep them unchanged", `{{param "routing_key"}}`, "# Part 3", "order.created", "## Payload fields", ".payload.id", `"type": "object"`, `"id": "1"`,
		`{"text": "..."}`, "docs.example.com/hooks", "title and body fields", "Current template to refine", "## Instructions", "do it"}
	at := 0
	for _, want := range order {
		i := strings.Index(user[at:], want)
		if i < 0 {
			t.Fatalf("user prompt missing %q after offset %d:\n%s", want, at, user)
		}
		at += i
	}

	// A reference template containing ``` gets a longer fence, so the
	// block is not closed early.
	if !strings.Contains(user, "````\n{\"routing_key\"") {
		t.Fatalf("reference template with ``` should use a longer fence:\n%s", user)
	}

	chat := p.ChatPrompt(Request{EventName: "e", Instructions: "do it", SamplePayload: map[string]any{"id": "1"}}, "")
	if strings.Contains(chat, "JSON object") || !strings.Contains(chat, "fenced code block") || !strings.Contains(chat, "# Part 1") || !strings.Contains(chat, "# Part 3") || strings.Contains(chat, "# Part 2") {
		t.Fatalf("chat prompt should be self-contained, ask for a code block, and skip part 2 without a recipe:\n%s", chat)
	}
}

// The worked example in the reference must itself follow the rules: render
// strictly, with and without the optional field, to valid JSON.
func TestTemplateReference_ExampleRenders(t *testing.T) {
	ref := templateReference
	start := strings.Index(ref, "FENCE\n") + len("FENCE\n")
	end := strings.LastIndex(ref, "FENCE")
	tmpl := ref[start:end]
	engine := template.NewTemplateEngine()
	for _, payload := range []map[string]any{
		{"id": "ord_1", "total": 49.5, "customer": map[string]any{"email": "a@example.com"}, "coupon": "SPRING"},
		{"id": "ord_2", "total": 1500000.0, "customer": map[string]any{"email": "b@example.com"}},
	} {
		data := template.NewWebhookTemplateContext("evt_1", "order.paid", "2026-03-01T12:00:00Z", 1, payload)
		out, err := engine.TransformPayloadWith(tmpl, data, template.ExecOptions{StrictMissingKeys: true})
		if err != nil {
			t.Fatalf("example does not render: %v", err)
		}
		if !json.Valid(out) || checkRendered(string(out)) != nil {
			t.Fatalf("example renders invalid output:\n%s", out)
		}
	}
}

func TestPayloadOutline(t *testing.T) {
	schema := map[string]any{
		"type":     "object",
		"required": []any{"id", "items", "customer"},
		"properties": map[string]any{
			"id":       map[string]any{"type": "string", "description": "Order id,\n  e.g. ord_123", "format": "uuid"},
			"status":   map[string]any{"type": "string", "enum": []any{"paid", "refunded"}},
			"coupon":   map[string]any{"type": "string"},
			"note":     map[string]any{"type": []any{"string", "null"}},
			"shipping": map[string]any{"type": "object", "required": []any{"city"}, "properties": map[string]any{"city": map[string]any{"type": "string"}}},
			"customer": map[string]any{"type": "object", "required": []any{"email"}, "properties": map[string]any{"email": map[string]any{"type": "string"}, "phone": map[string]any{"type": "string"}}},
			"items":    map[string]any{"type": "array", "items": map[string]any{"type": "object", "required": []any{"sku"}, "properties": map[string]any{"sku": map[string]any{"type": "string"}}}},
		},
	}
	sample := map[string]any{
		"id": "ord_1", "note": nil, "first-name": "Ada",
		"customer": map[string]any{"email": "a@example.com"},
		"shipping": map[string]any{"city": "Oslo"},
		"items":    []any{map[string]any{"sku": "A1", "qty": 2.0}},
	}
	out := payloadOutline(schema, sample)
	rows := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		if f := strings.Fields(line); len(f) > 1 && (strings.HasPrefix(f[0], ".payload")) {
			rows[f[0]] = line
		}
	}
	for path, want := range map[string][]string{
		".payload.id":             {"string", "required", `"ord_1"`, "# format uuid; Order id, e.g. ord_123"},
		".payload.status":         {"optional", `# one of "paid", "refunded"`},
		".payload.coupon":         {"string", "optional", "(absent)"},
		".payload.note":           {"optional, nullable", "null"},
		`.payload["first-name"]`:  {"string", "optional"},
		".payload.customer":       {"object", "required", "{1 keys}"},
		".payload.customer.email": {"required"},
		".payload.shipping.city":  {"required in parent", `"Oslo"`},
		".payload.customer.phone": {"optional", "(absent)"},
		".payload.items":          {"array", "required", "[1 items]"},
		".payload.items[].sku":    {"required", `"A1"`},
		".payload.items[].qty":    {"number", "optional", "2"},
	} {
		line, ok := rows[path]
		if !ok {
			t.Fatalf("outline missing %s:\n%s", path, out)
		}
		for _, w := range want {
			if !strings.Contains(line, w) {
				t.Errorf("%s: want %q in %q", path, w, line)
			}
		}
	}

	// Without a schema nothing is known to be required.
	if out := payloadOutline(nil, map[string]any{"id": "x"}); !strings.Contains(out, ".payload.id  string  unknown") {
		t.Fatalf("schemaless outline:\n%s", out)
	}
	if payloadOutline(nil, nil) != "" {
		t.Fatal("empty outline expected")
	}
}
