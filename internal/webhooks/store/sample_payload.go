package store

import (
	"encoding/json"
	"fmt"

	"github.com/sarathsp06/schemagen"
)

// SamplePayload builds an example payload from an event type schema: every
// field when all is true, only required fields otherwise. A non-zero seed
// makes the output deterministic. An empty schema yields an empty payload.
func SamplePayload(schema map[string]any, all bool, seed int64) (map[string]any, error) {
	if len(schema) == 0 {
		return map[string]any{}, nil
	}
	b, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}
	g := schemagen.NewGenerator().SetGenerateAllFields(all)
	if seed != 0 {
		g.SetSeed(seed)
	}
	sample, err := g.Generate(b)
	if err != nil {
		return nil, fmt.Errorf("failed to generate sample payload: %w", err)
	}
	if m, ok := sample.(map[string]any); ok {
		return m, nil
	}
	return map[string]any{}, nil
}
