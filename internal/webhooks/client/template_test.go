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
	result1, err := engine.Execute(tmpl, data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Second execution - should use cache and produce same result
	result2, err := engine.Execute(tmpl, data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(result1) != string(result2) {
		t.Errorf("Expected same result from cached execution, got %q and %q", string(result1), string(result2))
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

func TestNewTemplateEngineWithCacheSize(t *testing.T) {
	maxSize := 50
	engine := NewTemplateEngineWithCacheSize(maxSize)

	if engine == nil {
		t.Fatal("Expected non-nil template engine")
	}

	// Verify engine works with custom cache size
	result, err := engine.Execute(`Hello {{.Name}}`, map[string]any{"Name": "World"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if string(result) != "Hello World" {
		t.Errorf("Expected 'Hello World', got %q", string(result))
	}
}

func TestExecuteComplexTemplate(t *testing.T) {
	engine := NewTemplateEngine()

	tmpl := `
{
  "event_id": "{{.event_id}}",
  "event_name": "{{.event_name | upper}}",
  "payload": {{.payload | json}},
  "user_email": "{{.payload.email | lower}}"
}
`
	data := WebhookTemplateContext{
		"event_id":   "event-123",
		"event_name": "user.created",
		"payload": map[string]any{
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
	tmpl := `{"id": "{{.event_id}}", "name": "{{.event_name | upper}}", "payload": {{.payload | json}}}`
	data := WebhookTemplateContext{
		"event_id":   "event-123",
		"event_name": "user.created",
		"payload": map[string]any{
			"email": "test@example.com",
			"name":  "Test User",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(tmpl, data)
	}
}
