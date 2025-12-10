package client

import (
	"bytes"
	"fmt"
	"text/template"
	"time"
)

// TemplateEngine handles payload transformations using Go templates
type TemplateEngine struct {
	funcs template.FuncMap
	cache *TemplateCache
}

// NewTemplateEngine creates a new template engine with default helpers
func NewTemplateEngine() *TemplateEngine {
	return NewTemplateEngineWithCacheSize(DefaultCacheSize)
}

// NewTemplateEngineWithCacheSize creates a new template engine with custom cache size
func NewTemplateEngineWithCacheSize(maxSize int) *TemplateEngine {
	return &TemplateEngine{
		funcs: GetFunctionMap(),
		cache: NewTemplateCache(maxSize),
	}
}

// Execute processes a template with the given data
func (e *TemplateEngine) Execute(tmplStr string, data any) ([]byte, error) {
	if tmplStr == "" {
		return nil, nil
	}

	// Get or create cached template
	tmpl, err := e.getOrParseTemplate(tmplStr)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// getOrParseTemplate retrieves a cached template or parses and caches a new one
func (e *TemplateEngine) getOrParseTemplate(tmplStr string) (*template.Template, error) {
	key := hashTemplate(tmplStr)

	// Try to get from cache
	if tmpl, found := e.cache.Get(key); found {
		return tmpl, nil
	}

	// Parse new template
	tmpl, err := template.New("webhook").Funcs(e.funcs).Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// Add to cache
	e.cache.Put(key, tmpl)

	return tmpl, nil
}

// CacheStats returns current cache statistics
func (e *TemplateEngine) CacheStats() (size, maxSize int) {
	return e.cache.Stats()
}

// ClearCache removes all cached templates
func (e *TemplateEngine) ClearCache() {
	e.cache.Clear()
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

// ValidateTemplateWithPayload validates a template by executing it with a specific payload
func (e *TemplateEngine) ValidateTemplateWithPayload(tmplStr string, payload map[string]any) error {
	if tmplStr == "" {
		return nil
	}

	// Create test data using the provided payload
	testData := map[string]any{
		"Event": map[string]any{
			"ID":        "test-event-id",
			"Namespace": "default",
			"Event":     "test.event",
			"CreatedAt": time.Now().Format(time.RFC3339),
		},
		"Webhook": map[string]any{
			"ID":  "test-webhook-id",
			"URL": "https://example.com/webhook",
		},
		"Payload": payload,
	}

	_, err := e.Execute(tmplStr, testData)
	if err != nil {
		return fmt.Errorf("template validation failed with payload: %w", err)
	}

	return nil
}

type WebhookTemplateContext struct {
	EventID   string
	EventName string
	Payload   map[string]any
}

// TransformPayload applies the template if enabled
func (e *TemplateEngine) TransformPayload(tmplStr string, data WebhookTemplateContext) ([]byte, error) {
	if tmplStr == "" {
		return nil, nil
	}
	return e.Execute(tmplStr, data)
}
