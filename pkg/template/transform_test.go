package template

import (
	"math/rand/v2"
	"testing"
)

func TestTransformPayload(t *testing.T) {
	engine := New()

	tests := []struct {
		name     string
		template string
		data     WebhookContext
		expected string
		wantErr  bool
	}{
		{
			name:     "simple template",
			template: `{"event": "{{.event_name}}"}`,
			data: WebhookContext{
				"event_name": "user.created",
			},
			expected: `{"event": "user.created"}`,
			wantErr:  false,
		},
		{
			name:     "with payload",
			template: `{"user_id": "{{.payload.user_id}}"}`,
			data: WebhookContext{
				"payload": map[string]any{"user_id": "123"},
			},
			expected: `{"user_id": "123"}`,
			wantErr:  false,
		},
		{
			name:     "empty template",
			template: "",
			data:     WebhookContext{},
			expected: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.ExecuteWith(tt.template, tt.data, ExecOptions{})

			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error=%v, got error=%v", tt.wantErr, err)
			}

			if tt.template == "" && result != nil {
				t.Error("Expected nil result for empty template")
			}

			if tt.template != "" && string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func BenchmarkTransformPayload(b *testing.B) {
	engine := New()
	templates := []string{
		`{"event": "{{.event_name}}", "id": "{{.event_id}}"}`,
		`{"event": "{{.payload.event}}", "id": "{{.event_id}}"}`,
		`{"event": "{{.event_name | upper}}", "id": "{{.event_id | lower}}"}`,
	}

	data := WebhookContext{
		"event_id":   "event-123",
		"event_name": "user.created",
		"payload": map[string]any{
			"event": "user.created",
		},
	}

	// Pick a random template each iteration to exercise the LRU cache.
	r := rand.New(rand.NewPCG(1, 2))
	for b.Loop() {
		_, _ = engine.ExecuteWith(templates[r.IntN(len(templates))], data, ExecOptions{})
	}
}
