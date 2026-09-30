package webhooks

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// js decodes a JSON Schema literal the way stored and incoming schemas are
// decoded (numbers as float64, arrays as []any).
func js(t *testing.T, s string) map[string]any {
	t.Helper()
	if s == "" {
		return nil
	}
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(s), &m))
	return m
}

const orderV1 = `{
  "type": "object",
  "required": ["order_id", "total", "customer"],
  "properties": {
    "order_id": {"type": "string"},
    "total":    {"type": "number"},
    "coupon":   {"type": "string"},
    "customer": {
      "type": "object",
      "required": ["email"],
      "properties": {"email": {"type": "string"}, "phone": {"type": "string"}}
    },
    "items": {"type": "array", "items": {"type": "object", "required": ["sku"], "properties": {"sku": {"type": "string"}, "qty": {"type": "integer"}}}}
  }
}`

func TestClassifySchemaChange(t *testing.T) {
	tests := []struct {
		name     string
		old, new string
		breaking bool
		reasons  []string
	}{
		{name: "identical", old: orderV1, new: orderV1},
		{name: "first schema is not a change", old: "", new: orderV1},
		{name: "schema removed", old: orderV1, new: "", breaking: true, reasons: []string{"(root): schema removed; every required property is gone"}},
		{
			name: "add optional property",
			old:  `{"type":"object","required":["a"],"properties":{"a":{"type":"string"}}}`,
			new:  `{"type":"object","required":["a"],"properties":{"a":{"type":"string"},"b":{"type":"string"}}}`,
		},
		{
			name: "add required property",
			old:  `{"type":"object","required":["a"],"properties":{"a":{"type":"string"}}}`,
			new:  `{"type":"object","required":["a","b"],"properties":{"a":{"type":"string"},"b":{"type":"string"}}}`,
		},
		{
			name: "remove optional property",
			old:  `{"type":"object","required":["a"],"properties":{"a":{"type":"string"},"b":{"type":"string"}}}`,
			new:  `{"type":"object","required":["a"],"properties":{"a":{"type":"string"}}}`,
		},
		{
			name: "remove required property", breaking: true,
			old:     `{"type":"object","required":["a","b"],"properties":{"a":{"type":"string"},"b":{"type":"string"}}}`,
			new:     `{"type":"object","required":["a"],"properties":{"a":{"type":"string"}}}`,
			reasons: []string{"b: removed required property"},
		},
		{
			name: "required becomes optional", breaking: true,
			old:     `{"type":"object","required":["a"],"properties":{"a":{"type":"string"}}}`,
			new:     `{"type":"object","properties":{"a":{"type":"string"}}}`,
			reasons: []string{"a: no longer required, so it may be absent"},
		},
		{
			name: "incompatible type change", breaking: true,
			old:     `{"type":"object","required":["a"],"properties":{"a":{"type":"string"}}}`,
			new:     `{"type":"object","required":["a"],"properties":{"a":{"type":"integer"}}}`,
			reasons: []string{"a: type changed from string to integer"},
		},
		{
			name: "integer widened to number could break integer formatting", breaking: true,
			old:     `{"type":"object","properties":{"n":{"type":"integer"}}}`,
			new:     `{"type":"object","properties":{"n":{"type":"number"}}}`,
			reasons: []string{"n: type changed from integer to number"},
		},
		{
			name: "number narrowed to integer",
			old:  `{"type":"object","properties":{"n":{"type":"number"}}}`,
			new:  `{"type":"object","properties":{"n":{"type":"integer"}}}`,
		},
		{
			name: "type union gains a type", breaking: true,
			old:     `{"type":"object","properties":{"n":{"type":"string"}}}`,
			new:     `{"type":"object","properties":{"n":{"type":["string","null"]}}}`,
			reasons: []string{"n: type changed from string to null|string"},
		},
		{
			name: "value constraints change",
			old:  `{"type":"object","properties":{"s":{"type":"string","enum":["a","b"],"maxLength":3}}}`,
			new:  `{"type":"object","properties":{"s":{"type":"string","enum":["a","b","c"],"maxLength":10,"format":"email"}}}`,
		},
		{
			name: "nested required removed", breaking: true,
			old:     orderV1,
			new:     `{"type":"object","required":["order_id","total","customer"],"properties":{"order_id":{"type":"string"},"total":{"type":"number"},"coupon":{"type":"string"},"customer":{"type":"object","properties":{"phone":{"type":"string"}}},"items":{"type":"array","items":{"type":"object","required":["sku"],"properties":{"sku":{"type":"string"},"qty":{"type":"integer"}}}}}}`,
			reasons: []string{"customer.email: removed required property"},
		},
		{
			name: "array item type changed", breaking: true,
			old:     orderV1,
			new:     `{"type":"object","required":["order_id","total","customer"],"properties":{"order_id":{"type":"string"},"total":{"type":"number"},"coupon":{"type":"string"},"customer":{"type":"object","required":["email"],"properties":{"email":{"type":"string"},"phone":{"type":"string"}}},"items":{"type":"array","items":{"type":"object","required":["sku"],"properties":{"sku":{"type":"integer"},"qty":{"type":"integer"}}}}}}`,
			reasons: []string{"items[].sku: type changed from string to integer"},
		},
		{
			name: "several breaking changes are all listed", breaking: true,
			old:     `{"type":"object","required":["a","b"],"properties":{"a":{"type":"string"},"b":{"type":"string"}}}`,
			new:     `{"type":"object","properties":{"a":{"type":"integer"}}}`,
			reasons: []string{"a: no longer required, so it may be absent", "a: type changed from string to integer", "b: removed required property"},
		},
		{
			name: "change under oneOf cannot be proven safe", breaking: true,
			old:     `{"type":"object","properties":{"p":{"oneOf":[{"type":"string"},{"type":"integer"}]}}}`,
			new:     `{"type":"object","properties":{"p":{"oneOf":[{"type":"string"},{"type":"boolean"}]}}}`,
			reasons: []string{"p: changed under oneOf, which cannot be checked for compatibility"},
		},
		{
			name: "adding $ref cannot be proven safe", breaking: true,
			old:     `{"type":"object","properties":{"p":{"type":"object"}}}`,
			new:     `{"type":"object","properties":{"p":{"$ref":"#/$defs/P"}}}`,
			reasons: []string{"p: changed under $ref, which cannot be checked for compatibility"},
		},
		{
			name: "unchanged oneOf elsewhere does not taint the rest",
			old:  `{"type":"object","properties":{"p":{"oneOf":[{"type":"string"}]},"q":{"type":"string"}}}`,
			new:  `{"type":"object","properties":{"p":{"oneOf":[{"type":"string"}]},"q":{"type":"string"},"r":{"type":"string"}}}`,
		},
		{
			name: "type removed allows anything", breaking: true,
			old:     `{"type":"object","properties":{"p":{"type":"string"}}}`,
			new:     `{"type":"object","properties":{"p":{}}}`,
			reasons: []string{"p: type string removed, so any type is now allowed"},
		},
		{
			name: "type added where there was none",
			old:  `{"type":"object","properties":{"p":{}}}`,
			new:  `{"type":"object","properties":{"p":{"type":"string"}}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifySchemaChange(js(t, tt.old), js(t, tt.new))
			assert.Equal(t, tt.breaking, got.Breaking, "reasons: %v", got.Reasons)
			if tt.breaking {
				assert.Equal(t, tt.reasons, got.Reasons)
				assert.Equal(t, "breaking", got.Result())
			} else {
				assert.Empty(t, got.Reasons)
				assert.Equal(t, "compatible", got.Result())
			}
		})
	}
}
