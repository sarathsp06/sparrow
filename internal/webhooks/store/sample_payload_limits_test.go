package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Size keywords are clamped before generation: these schemas used to
// allocate their full size up front and crash the process out of memory.
func TestSamplePayload_ClampsHugeSizes(t *testing.T) {
	schema := map[string]any{
		"type":     "object",
		"required": []any{"s", "a"},
		"properties": map[string]any{
			"s": map[string]any{"type": "string", "minLength": 1e11},
			"a": map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "minItems": 1e11},
		},
	}
	got, err := SamplePayload(schema, true, 1)
	require.NoError(t, err)
	assert.Len(t, got["s"], maxSampleLength)
	assert.Len(t, got["a"], maxSampleItems)
	assert.Equal(t, 1e11, schema["properties"].(map[string]any)["s"].(map[string]any)["minLength"], "caller's schema modified")
}

// Nested arrays multiply; the node budget stops generation with an error.
func TestSamplePayload_NestedArraysHitNodeBudget(t *testing.T) {
	var leaf any = map[string]any{"type": "integer"}
	for range 8 {
		leaf = map[string]any{"type": "array", "items": leaf, "minItems": 50}
	}
	schema := map[string]any{"type": "object", "required": []any{"x"}, "properties": map[string]any{"x": leaf}}
	_, err := SamplePayload(schema, true, 1)
	require.ErrorIs(t, err, errSampleTooLarge)
}
