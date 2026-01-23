package client

import (
	"strings"
	"testing"
)

func TestNewTemplateEngine(t *testing.T) {
	engine := NewTemplateEngine()

	if engine == nil {
		t.Fatal("Expected non-nil template engine")
		return
	}

	if engine.cache == nil {
		t.Error("Expected cache to be initialized")
	}

	if engine.funcs == nil {
		t.Error("Expected function map to be initialized")
	}
}

func TestExecuteSimpleTemplate(t *testing.T) {
	engine := NewTemplateEngine()

	tmpl := `Hello {{.Name}}`
	data := map[string]any{"Name": "World"}

	result, err := engine.Execute(tmpl, data)
	if err != nil {
		t.Fatalf("Unexpected error executing template: %v", err)
	}

	expected := "Hello World"
	if string(result) != expected {
		t.Errorf("Expected %q, got %q", expected, string(result))
	}
}

func TestExecuteEmptyTemplate(t *testing.T) {
	engine := NewTemplateEngine()

	result, err := engine.Execute("", nil)
	if err != nil {
		t.Fatalf("Unexpected error with empty template: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result for empty template, got %v", result)
	}
}

func TestExecuteWithTemplateFunctions(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		{
			name:     "upper function",
			template: `{{.Name | upper}}`,
			data:     map[string]any{"Name": "hello"},
			expected: "HELLO",
		},
		{
			name:     "lower function",
			template: `{{.Name | lower}}`,
			data:     map[string]any{"Name": "HELLO"},
			expected: "hello",
		},
		{
			name:     "json function",
			template: `{{.Data | json}}`,
			data:     map[string]any{"Data": map[string]any{"key": "value"}},
			expected: `{"key":"value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Execute(tt.template, tt.data)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func TestExecuteInvalidTemplate(t *testing.T) {
	engine := NewTemplateEngine()

	tmpl := `{{.Name | invalid_function}}`
	data := map[string]any{"Name": "World"}

	_, err := engine.Execute(tmpl, data)
	if err == nil {
		t.Error("Expected error with invalid template function")
	}
}

func TestTemplateCaching(t *testing.T) {
	engine := NewTemplateEngine()

	tmpl := `Hello {{.Name}}`
	data := map[string]any{"Name": "World"}

	// First execution - should cache
	_, err := engine.Execute(tmpl, data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	size, _ := engine.CacheStats()
	if size != 1 {
		t.Errorf("Expected cache size 1, got %d", size)
	}

	// Second execution - should use cache
	_, err = engine.Execute(tmpl, data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	size, _ = engine.CacheStats()
	if size != 1 {
		t.Errorf("Expected cache size still 1, got %d", size)
	}
}

func TestClearCache(t *testing.T) {
	engine := NewTemplateEngine()

	tmpl := `Hello {{.Name}}`
	data := map[string]any{"Name": "World"}

	// Execute to populate cache
	_, err := engine.Execute(tmpl, data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	size, _ := engine.CacheStats()
	if size != 1 {
		t.Errorf("Expected cache size 1, got %d", size)
	}

	// Clear cache
	engine.ClearCache()

	size, _ = engine.CacheStats()
	if size != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", size)
	}
}

func TestValidateTemplate(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name      string
		template  string
		expectErr bool
	}{
		{
			name:      "valid template",
			template:  `Hello {{.Name}}`,
			expectErr: false,
		},
		{
			name:      "empty template",
			template:  "",
			expectErr: false,
		},
		{
			name:      "invalid syntax",
			template:  `{{.Name}`,
			expectErr: true,
		},
		{
			name:      "unclosed action",
			template:  `Hello {{.Name`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateTemplate(tt.template)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error=%v, got error=%v", tt.expectErr, err)
			}
		})
	}
}

func TestValidateTemplateWithTestData(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name      string
		template  string
		expectErr bool
	}{
		{
			name:      "valid template with event data",
			template:  `Event ID: {{.Event.ID}}`,
			expectErr: false,
		},
		{
			name:      "valid template with payload",
			template:  `User: {{.Payload.user_id}}`,
			expectErr: false,
		},
		{
			name:      "empty template",
			template:  "",
			expectErr: false,
		},
		{
			name:      "invalid syntax",
			template:  `{{.Event.ID}`,
			expectErr: true,
		},
		{
			name:      "template with functions",
			template:  `{{.Event.Event | upper}}`,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateTemplateWithTestData(tt.template)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error=%v, got error=%v", tt.expectErr, err)
			}
		})
	}
}

func TestValidateTemplateWithPayload(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name      string
		template  string
		payload   map[string]any
		expectErr bool
	}{
		{
			name:     "valid template with custom payload",
			template: `User ID: {{.Payload.user_id}}, Email: {{.Payload.email}}`,
			payload: map[string]any{
				"user_id": "12345",
				"email":   "test@example.com",
			},
			expectErr: false,
		},
		{
			name:     "valid template with nested payload",
			template: `Name: {{.Payload.user.name}}, Age: {{.Payload.user.age}}`,
			payload: map[string]any{
				"user": map[string]any{
					"name": "John Doe",
					"age":  30,
				},
			},
			expectErr: false,
		},
		{
			name:      "empty template",
			template:  "",
			payload:   map[string]any{},
			expectErr: false,
		},
		{
			name:     "invalid syntax",
			template: `{{.Payload.user_id}`,
			payload: map[string]any{
				"user_id": "12345",
			},
			expectErr: true,
		},
		{
			name:     "template with functions on custom payload",
			template: `{{.Payload.email | upper}}`,
			payload: map[string]any{
				"email": "test@example.com",
			},
			expectErr: false,
		},
		{
			name:     "accessing missing field should not error during validation",
			template: `{{.Payload.missing_field}}`,
			payload: map[string]any{
				"user_id": "12345",
			},
			expectErr: false, // Go templates return empty string for missing fields
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateTemplateWithPayload(tt.template, tt.payload)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error=%v, got error=%v", tt.expectErr, err)
			}
		})
	}
}

func TestNewTemplateEngineWithCacheSize(t *testing.T) {
	maxSize := 50
	engine := NewTemplateEngineWithCacheSize(maxSize)

	if engine == nil {
		t.Fatal("Expected non-nil template engine")
	}

	_, max := engine.CacheStats()
	if max != maxSize {
		t.Errorf("Expected max cache size %d, got %d", maxSize, max)
	}
}

func TestExecuteComplexTemplate(t *testing.T) {
	engine := NewTemplateEngine()

	tmpl := `
{
  "event_id": "{{.EventID}}",
  "event_name": "{{.EventName | upper}}",
  "payload": {{.Payload | json}},
  "user_email": "{{.Payload.email | lower}}"
}
`
	data := WebhookTemplateContext{
		EventID:   "event-123",
		EventName: "user.created",
		Payload: map[string]any{
			"email": "USER@EXAMPLE.COM",
			"name":  "Test User",
		},
	}

	result, err := engine.Execute(tmpl, data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultStr := string(result)

	if !strings.Contains(resultStr, "event-123") {
		t.Error("Expected result to contain event ID")
	}

	if !strings.Contains(resultStr, "USER.CREATED") {
		t.Error("Expected result to contain uppercased event name")
	}

	if !strings.Contains(resultStr, "user@example.com") {
		t.Error("Expected result to contain lowercased email")
	}
}

func BenchmarkExecuteSimple(b *testing.B) {
	engine := NewTemplateEngine()
	tmpl := `Hello {{.Name}}`
	data := map[string]any{"Name": "World"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(tmpl, data)
	}
}

func BenchmarkExecuteComplex(b *testing.B) {
	engine := NewTemplateEngine()
	tmpl := `{"id": "{{.EventID}}", "name": "{{.EventName | upper}}", "payload": {{.Payload | json}}}`
	data := WebhookTemplateContext{
		EventID:   "event-123",
		EventName: "user.created",
		Payload: map[string]any{
			"email": "test@example.com",
			"name":  "Test User",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(tmpl, data)
	}
}
