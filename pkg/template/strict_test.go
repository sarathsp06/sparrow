package template

import (
	"strings"
	"testing"
)

func TestTransformPayloadWith_StrictMissingKeys(t *testing.T) {
	engine := NewTemplateEngine()
	data := NewWebhookTemplateContext("evt_1", "order.created", "2026-09-30T00:00:00Z", 1, map[string]any{
		"order_id": "o-1",
		"customer": map[string]any{"name": "Ada"},
	})
	strict := ExecOptions{StrictMissingKeys: true}

	tests := []struct {
		name       string
		tmpl       string
		lenient    string
		strictOut  string
		strictFail bool
	}{
		{
			name:      "present field renders the same either way",
			tmpl:      `{{.payload.order_id}}`,
			lenient:   "o-1",
			strictOut: "o-1",
		},
		{
			name:       "missing field renders <no value> leniently and fails strictly",
			tmpl:       `{{.payload.total}}`,
			lenient:    "<no value>",
			strictFail: true,
		},
		{
			name:       "missing nested field fails strictly",
			tmpl:       `{{.payload.shipping.city}}`,
			lenient:    "<no value>",
			strictFail: true,
		},
		{
			name:      "index is safe for optional fields",
			tmpl:      `[{{index .payload "total"}}]`,
			lenient:   "[<no value>]",
			strictOut: "[<no value>]",
		},
		{
			name:      "dig is safe for optional nested fields",
			tmpl:      `{{dig "shipping" "city" "unknown" .payload}}`,
			lenient:   "unknown",
			strictOut: "unknown",
		},
		{
			name:      "dig reads present nested fields",
			tmpl:      `{{dig "customer" "name" "?" .payload}}`,
			lenient:   "Ada",
			strictOut: "Ada",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.TransformPayload(tt.tmpl, data)
			if err != nil {
				t.Fatalf("lenient: unexpected error: %v", err)
			}
			if string(got) != tt.lenient {
				t.Errorf("lenient: got %q, want %q", got, tt.lenient)
			}

			got, err = engine.TransformPayloadWith(tt.tmpl, data, strict)
			if tt.strictFail {
				if err == nil {
					t.Fatalf("strict: expected an error, got %q", got)
				}
				if !strings.Contains(err.Error(), "map has no entry for key") {
					t.Errorf("strict: error should name the missing key, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("strict: unexpected error: %v", err)
			}
			if string(got) != tt.strictOut {
				t.Errorf("strict: got %q, want %q", got, tt.strictOut)
			}
		})
	}
}

func TestTransformPayloadWith_CachesStrictAndLenientSeparately(t *testing.T) {
	engine := NewTemplateEngine()
	data := NewWebhookTemplateContext("e", "n", "t", 1, map[string]any{})
	tmpl := `{{.payload.missing}}`

	// Warm the cache with the lenient parse first; strict must still fail.
	if _, err := engine.TransformPayload(tmpl, data); err != nil {
		t.Fatalf("lenient: %v", err)
	}
	if _, err := engine.TransformPayloadWith(tmpl, data, ExecOptions{StrictMissingKeys: true}); err == nil {
		t.Fatal("strict render reused the lenient cached template")
	}
	// And the lenient parse must not have been turned strict.
	if _, err := engine.TransformPayload(tmpl, data); err != nil {
		t.Fatalf("lenient after strict: %v", err)
	}
}
