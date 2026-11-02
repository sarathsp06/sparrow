package store

import (
	"testing"
	"time"

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

// The node budget counts schemagen's ctx.Done calls, which is how schemagen
// works today, not its API. If an upgrade checks Done less often, nested
// minItems at the cap (100^4 values here) runs away; fail fast, naming the
// cause, instead of hanging until the test timeout.
func TestSamplePayload_NodeBudgetStillCountsSchemagen(t *testing.T) {
	var leaf any = map[string]any{"type": "integer"}
	for range 4 {
		leaf = map[string]any{"type": "array", "items": leaf, "minItems": 100}
	}
	schema := map[string]any{"type": "object", "required": []any{"x"}, "properties": map[string]any{"x": leaf}}
	errc := make(chan error, 1)
	go func() {
		_, err := SamplePayload(schema, true, 1)
		errc <- err
	}()
	select {
	case err := <-errc:
		require.ErrorIs(t, err, errSampleTooLarge)
	case <-time.After(10 * time.Second):
		t.Fatal("node budget did not stop generation: schemagen no longer calls ctx.Done once per generated value; see nodeBudget")
	}
}
