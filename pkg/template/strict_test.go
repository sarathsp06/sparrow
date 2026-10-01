package template

import (
	"strings"
	"testing"
)

func TestStrictEngine_MissingKeyIsAnError(t *testing.T) {
	data := NewWebhookTemplateContext("id", "order.created", "2026-01-01T00:00:00Z", 1, map[string]any{"id": "ord_1"})
	const tmpl = `{"text": {{ printf "Order %s" .Payload.id | json }}}`

	lenient, err := NewTemplateEngine().Execute(tmpl, data)
	if err != nil {
		t.Fatalf("lenient engine should render: %v", err)
	}
	if !strings.Contains(string(lenient), "nil") && !strings.Contains(string(lenient), "no value") {
		t.Fatalf("lenient output should carry a missing-value marker: %s", lenient)
	}

	strict := NewStrictTemplateEngine()
	if !strict.Strict() {
		t.Fatal("Strict() should be true")
	}
	_, err = strict.Execute(tmpl, data)
	if err == nil || !strings.Contains(err.Error(), `"Payload"`) {
		t.Fatalf("strict engine should fail naming the key, got: %v", err)
	}

	// A present key still renders, and index on a missing key is tolerated,
	// which is the documented way to read optional fields under strict mode.
	out, err := strict.Execute(`{{ .payload.id }}|{{ with index .payload "coupon" }}{{ . }}{{ else }}none{{ end }}`, data)
	if err != nil || string(out) != "ord_1|none" {
		t.Fatalf("strict engine on valid template: %q %v", out, err)
	}

	// Engines keep separate caches, so the same template string parsed by
	// the lenient engine does not leak its options into the strict one.
	if _, err := strict.Execute(tmpl, data); err == nil {
		t.Fatal("strict engine must stay strict for a template the lenient engine already parsed")
	}
}
