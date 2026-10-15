package template

import (
	"strings"
	"testing"
	"time"
)

func TestNewTemplateEngine(t *testing.T) {
	engine := New()

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
	engine := New()

	tmpl := `Hello {{.Name}}`
	data := map[string]any{"Name": "World"}

	result, err := engine.ExecuteWith(tmpl, data, ExecOptions{})
	if err != nil {
		t.Fatalf("Unexpected error executing template: %v", err)
	}

	expected := "Hello World"
	if string(result) != expected {
		t.Errorf("Expected %q, got %q", expected, string(result))
	}
}

func TestExecuteEmptyTemplate(t *testing.T) {
	engine := New()

	result, err := engine.ExecuteWith("", nil, ExecOptions{})
	if err != nil {
		t.Fatalf("Unexpected error with empty template: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result for empty template, got %v", result)
	}
}

func TestExecuteWithTemplateFunctions(t *testing.T) {
	engine := New()

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
			result, err := engine.ExecuteWith(tt.template, tt.data, ExecOptions{})
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func TestExecuteReshapeFunctions(t *testing.T) {
	engine := New()

	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		{
			name:     "dict to json builds object",
			template: `{{ dict "user" .payload.id "amt" .payload.amount | json }}`,
			data:     map[string]any{"payload": map[string]any{"id": "u1", "amount": 5.0}},
			expected: `{"amt":5,"user":"u1"}`,
		},
		{
			name:     "list to json builds array",
			template: `{{ list "a" "b" | json }}`,
			data:     map[string]any{},
			expected: `["a","b"]`,
		},
		{
			name:     "merge later key wins",
			template: `{{ merge (dict "a" 1 "b" 2) (dict "a" 9) | json }}`,
			data:     map[string]any{},
			expected: `{"a":9,"b":2}`,
		},
		{
			name:     "mul then toInt for cents",
			template: `{{ toInt (mul .payload.amount 100) }}`,
			data:     map[string]any{"payload": map[string]any{"amount": 5.5}},
			expected: "550",
		},
		{
			name:     "add coerces numeric string",
			template: `{{ add .payload.a "2.5" }}`,
			data:     map[string]any{"payload": map[string]any{"a": 5.0}},
			expected: "7.5",
		},
		{
			name:     "dig returns nested value",
			template: `{{ dig "customer" "id" "none" .payload }}`,
			data:     map[string]any{"payload": map[string]any{"customer": map[string]any{"id": "c1"}}},
			expected: "c1",
		},
		{
			name:     "dig returns default on missing key",
			template: `{{ dig "customer" "email" "none" .payload }}`,
			data:     map[string]any{"payload": map[string]any{"customer": map[string]any{"id": "c1"}}},
			expected: "none",
		},
		{
			name:     "structural reshape end to end",
			template: `{{ dict "id" (dig "customer" "id" "" .payload) "cents" (toInt (mul .payload.amount 100)) | json }}`,
			data:     map[string]any{"payload": map[string]any{"customer": map[string]any{"id": "c1"}, "amount": 12.0}},
			expected: `{"cents":1200,"id":"c1"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.ExecuteWith(tt.template, tt.data, ExecOptions{})
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func TestReshapeFunctionErrors(t *testing.T) {
	engine := New()

	tests := []struct {
		name     string
		template string
	}{
		{name: "dict odd args", template: `{{ dict "a" | json }}`},
		{name: "div by zero", template: `{{ div 1 0 }}`},
		{name: "mod by zero", template: `{{ mod 1 0 }}`},
		{name: "add non-numeric", template: `{{ add "abc" 1 }}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := engine.ExecuteWith(tt.template, map[string]any{}, ExecOptions{}); err == nil {
				t.Errorf("Expected error for %q, got none", tt.template)
			}
		})
	}
}

func TestExecuteInvalidTemplate(t *testing.T) {
	engine := New()

	tmpl := `{{.Name | invalid_function}}`
	data := map[string]any{"Name": "World"}

	_, err := engine.ExecuteWith(tmpl, data, ExecOptions{})
	if err == nil {
		t.Error("Expected error with invalid template function")
	}
}

func TestTemplateCaching(t *testing.T) {
	engine := New()

	tmpl := `Hello {{.Name}}`
	data := map[string]any{"Name": "World"}

	// First execution - should cache
	result1, err := engine.ExecuteWith(tmpl, data, ExecOptions{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Second execution - should use cache and produce same result
	result2, err := engine.ExecuteWith(tmpl, data, ExecOptions{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(result1) != string(result2) {
		t.Errorf("Expected same result from cached execution, got %q and %q", string(result1), string(result2))
	}
}

func TestExecuteComplexTemplate(t *testing.T) {
	engine := New()

	tmpl := `
{
  "event_id": "{{.event_id}}",
  "event_name": "{{.event_name | upper}}",
  "payload": {{.payload | json}},
  "user_email": "{{.payload.email | lower}}"
}
`
	data := WebhookContext{
		"event_id":   "event-123",
		"event_name": "user.created",
		"payload": map[string]any{
			"email": "USER@EXAMPLE.COM",
			"name":  "Test User",
		},
	}

	result, err := engine.ExecuteWith(tmpl, data, ExecOptions{})
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
	engine := New()
	tmpl := `Hello {{.Name}}`
	data := map[string]any{"Name": "World"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.ExecuteWith(tmpl, data, ExecOptions{})
	}
}

func BenchmarkExecuteComplex(b *testing.B) {
	engine := New()
	tmpl := `{"id": "{{.event_id}}", "name": "{{.event_name | upper}}", "payload": {{.payload | json}}}`
	data := WebhookContext{
		"event_id":   "event-123",
		"event_name": "user.created",
		"payload": map[string]any{
			"email": "test@example.com",
			"name":  "Test User",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.ExecuteWith(tmpl, data, ExecOptions{})
	}
}

func TestExecuteTimeout(t *testing.T) {
	engine := New()

	// This template creates a deeply nested loop that burns CPU.
	// With the timeout, it should be interrupted within ExecutionTimeout.
	tmpl := `{{range $a := split "," "a,b,c,d,e,f,g,h,i,j"}}{{range $b := split "," "a,b,c,d,e,f,g,h,i,j"}}{{range $c := split "," "a,b,c,d,e,f,g,h,i,j"}}{{range $d := split "," "a,b,c,d,e,f,g,h,i,j"}}{{range $e := split "," "a,b,c,d,e,f,g,h,i,j"}}{{range $f := split "," "a,b,c,d,e,f,g,h,i,j"}}{{range $g := split "," "a,b,c,d,e,f,g,h,i,j"}}{{$a}}{{end}}{{end}}{{end}}{{end}}{{end}}{{end}}{{end}}`

	start := time.Now()
	_, err := engine.ExecuteWith(tmpl, nil, ExecOptions{})
	elapsed := time.Since(start)

	// Should have errored (either timeout or output size limit)
	if err == nil {
		t.Fatal("Expected error from expensive template, got nil")
	}

	// Should complete within a reasonable margin above the timeout
	maxExpected := ExecutionTimeout + 2*time.Second
	if elapsed > maxExpected {
		t.Errorf("Template execution took %v, expected to be interrupted within %v", elapsed, maxExpected)
	}
}

func TestRejectsUnboundedLoops(t *testing.T) {
	engine := New()

	rejected := map[string]string{
		"integer literal":        `{{range 2000000000}}{{end}}`,
		"parenthesised literal":  `{{range $i := (2000000000)}}{{end}}`,
		"literal inside define":  `{{define "x"}}{{range 10}}{{end}}{{end}}{{template "x"}}`,
		"nesting past the limit": `{{range $.a}}{{range $.a}}{{range $.a}}{{range $.a}}{{end}}{{end}}{{end}}{{end}}`,
	}
	for name, tmpl := range rejected {
		t.Run(name, func(t *testing.T) {
			if _, err := engine.ExecuteWith(tmpl, map[string]any{"a": []int{1}}, ExecOptions{}); err == nil {
				t.Errorf("ExecuteWith(%q) succeeded, want rejection", tmpl)
			}
		})
	}

	allowed := `{{range $.a}}{{range $.a}}{{range $.a}}x{{end}}{{end}}{{end}}`
	out, err := engine.ExecuteWith(allowed, map[string]any{"a": []int{1, 2}}, ExecOptions{})
	if err != nil {
		t.Fatalf("ExecuteWith(%q): %v", allowed, err)
	}
	if string(out) != "xxxxxxxx" {
		t.Errorf("Execute(%q) = %q, want %q", allowed, out, "xxxxxxxx")
	}
}
