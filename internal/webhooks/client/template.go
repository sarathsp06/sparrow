package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"text/template"
	"time"
)

// TemplateEngine handles payload transformations using Go templates
type TemplateEngine struct {
	funcs template.FuncMap
}

// NewTemplateEngine creates a new template engine with default helpers
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		funcs: template.FuncMap{
			"json": func(v any) (string, error) {
				b, err := json.Marshal(v)
				return string(b), err
			},
			"urlencode": func(s string) string {
				return url.QueryEscape(s)
			},
			"base64": func(s string) string {
				return base64.StdEncoding.EncodeToString([]byte(s))
			},
			"now": func() time.Time {
				return time.Now()
			},
		},
	}
}

// Execute processes a template with the given data
func (e *TemplateEngine) Execute(tmplStr string, data any) ([]byte, error) {
	if tmplStr == "" {
		return nil, nil
	}

	tmpl, err := template.New("webhook").Funcs(e.funcs).Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// ValidateTemplate validates a template string without executing it
func (e *TemplateEngine) ValidateTemplate(tmplStr string) error {
	if tmplStr == "" {
		return nil
	}

	_, err := template.New("webhook_validation").Funcs(e.funcs).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("invalid template syntax: %w", err)
	}

	return nil
}

// ValidateTemplateWithTestData validates a template by executing it with sample test data
func (e *TemplateEngine) ValidateTemplateWithTestData(tmplStr string) error {
	if tmplStr == "" {
		return nil
	}

	// Create sample test data that matches the template context structure
	testData := map[string]any{
		"Event": map[string]any{
			"ID":        "test-event-id",
			"Namespace": "default",
			"Event":     "user.signup",
			"CreatedAt": time.Now().Format(time.RFC3339),
		},
		"Webhook": map[string]any{
			"ID":  "test-webhook-id",
			"URL": "https://example.com/webhook",
		},
		"Payload": map[string]any{
			"user_id": "12345",
			"email":   "test@example.com",
			"name":    "Test User",
			"plan":    "premium",
		},
	}

	_, err := e.Execute(tmplStr, testData)
	if err != nil {
		return fmt.Errorf("template validation failed with test data: %w", err)
	}

	return nil
}
