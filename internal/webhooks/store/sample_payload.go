package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"

	"github.com/sarathsp06/schemagen"
)

// Limits on a generated sample. schemagen trusts the schema's size keywords
// and allocates them up front, so {"minLength": 1e11} is an unrecoverable
// out-of-memory crash and nested minItems multiply. A sample is an example,
// not a conformance test: sizes above these caps are clamped, and a schema
// whose sample still needs more than maxSampleNodes values gets an error.
const (
	maxSampleItems  = 100  // minItems, maxItems, minContains, maxContains, minProperties, maxProperties
	maxSampleLength = 4096 // minLength, maxLength
	maxSampleNodes  = 10000
)

var errSampleTooLarge = errors.New("schema needs a sample payload larger than the limit")

// SamplePayload builds an example payload from an event type schema: every
// field when all is true, only required fields otherwise. A non-zero seed
// makes the output deterministic. An empty schema yields an empty payload.
func SamplePayload(schema map[string]any, all bool, seed int64) (map[string]any, error) {
	if len(schema) == 0 {
		return map[string]any{}, nil
	}
	b, err := json.Marshal(clampSizes(schema))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}
	g := schemagen.NewGenerator().SetGenerateAllFields(all)
	if seed != 0 {
		g.SetSeed(seed)
	}
	sample, err := g.GenerateWithContext(newNodeBudget(maxSampleNodes), b)
	if err != nil {
		return nil, fmt.Errorf("failed to generate sample payload: %w", err)
	}
	if m, ok := sample.(map[string]any); ok {
		return m, nil
	}
	return map[string]any{}, nil
}

var sizeCaps = map[string]float64{
	"minItems": maxSampleItems, "maxItems": maxSampleItems,
	"minContains": maxSampleItems, "maxContains": maxSampleItems,
	"minProperties": maxSampleItems, "maxProperties": maxSampleItems,
	"minLength": maxSampleLength, "maxLength": maxSampleLength,
}

// clampSizes returns a copy of v with every size keyword above its cap
// lowered to the cap, at any depth. The caller's schema is not modified.
func clampSizes(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := maps.Clone(t)
		for k, val := range t {
			if limit, ok := sizeCaps[k]; ok {
				if n, ok := toFloat(val); ok && n > limit {
					out[k] = limit
					continue
				}
			}
			out[k] = clampSizes(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = clampSizes(val)
		}
		return out
	default:
		return v
	}
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return math.Inf(1), true
		}
		return f, true
	}
	return 0, false
}

// nodeBudget is a context that is cancelled after n values are generated.
// schemagen checks Done once per value it generates, on one goroutine, so
// counting the calls bounds the total work of nested arrays and objects.
type nodeBudget struct {
	context.Context
	left int
	done chan struct{}
}

func newNodeBudget(n int) *nodeBudget {
	return &nodeBudget{Context: context.Background(), left: n, done: make(chan struct{})}
}

func (b *nodeBudget) Done() <-chan struct{} {
	if b.left == 0 {
		return b.done
	}
	b.left--
	if b.left == 0 {
		close(b.done)
	}
	return b.done
}

func (b *nodeBudget) Err() error {
	if b.left == 0 {
		return errSampleTooLarge
	}
	return nil
}
