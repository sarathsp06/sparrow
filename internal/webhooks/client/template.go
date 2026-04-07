package client

import (
	"context"
	"fmt"
	"text/template"
	"time"
)

// MaxTemplateOutputBytes is the maximum allowed output size from a template
// execution. Templates producing output larger than this are aborted to
// prevent denial-of-service via crafted templates.
const MaxTemplateOutputBytes = 1 * 1024 * 1024 // 1 MB

// TemplateExecutionTimeout limits CPU time for template execution.
// Go's text/template has no built-in cancellation, so we run Execute in
// a goroutine and abandon it on timeout. This prevents a malicious
// template with tight loops from monopolising a worker indefinitely.
const TemplateExecutionTimeout = 5 * time.Second

// limitedWriter wraps a bytes.Buffer and enforces a maximum write size.
// Once the limit is exceeded, all subsequent writes return an error.
type limitedWriter struct {
	buf     writerWithBytes
	limit   int
	written int
}

// writerWithBytes is the write interface needed from bytes.Buffer.
type writerWithBytes interface {
	Write(p []byte) (int, error)
	Len() int
	Bytes() []byte
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.written+len(p) > w.limit {
		return 0, fmt.Errorf("template output exceeds maximum size of %d bytes", w.limit)
	}
	n, err := w.buf.Write(p)
	w.written += n
	return n, err
}

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

// Execute processes a template with the given data.
// Output is limited to MaxTemplateOutputBytes and execution time is limited
// to TemplateExecutionTimeout to prevent denial-of-service via crafted
// templates that consume unbounded CPU or produce unbounded output.
func (e *TemplateEngine) Execute(tmplStr string, data any) ([]byte, error) {
	if tmplStr == "" {
		return nil, nil
	}

	// Get or create cached template
	tmpl, err := e.getOrParseTemplate(tmplStr)
	if err != nil {
		return nil, err
	}

	// Get buffer from pool
	buf := GetBuffer()
	defer PutBuffer(buf)

	// Wrap with a size-limited writer to prevent runaway output
	lw := &limitedWriter{buf: buf, limit: MaxTemplateOutputBytes}

	// SEC: Run template execution with a CPU timeout. Go's text/template
	// has no cancellation support, so we run Execute in a goroutine and
	// abandon it if it exceeds the deadline. The goroutine will eventually
	// terminate when it hits the output size limit or completes, but the
	// caller is not blocked past the timeout.
	ctx, cancel := context.WithTimeout(context.Background(), TemplateExecutionTimeout)
	defer cancel()

	execErr := make(chan error, 1)
	go func() {
		execErr <- tmpl.Execute(lw, data)
	}()

	select {
	case err := <-execErr:
		if err != nil {
			return nil, fmt.Errorf("failed to execute template: %w", err)
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("template execution timed out after %v (possible infinite loop)", TemplateExecutionTimeout)
	}

	// Copy bytes since we're returning the buffer to the pool
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
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

// WebhookTemplateContext is the data context passed to Go templates.
// It uses snake_case keys so that templates reference fields as
// {{.event_id}}, {{.event_name}}, {{.timestamp}}, {{.attempt}}, and
// {{.payload}}.
// Namespace, WebhookID, and DeliveryID are conveyed via HTTP headers
// and are intentionally omitted from the template context.
type WebhookTemplateContext = map[string]any

// NewWebhookTemplateContext builds a template context map with snake_case keys.
func NewWebhookTemplateContext(eventID, eventName, timestamp string, attempt int, payload map[string]any) WebhookTemplateContext {
	return WebhookTemplateContext{
		"event_id":   eventID,
		"event_name": eventName,
		"timestamp":  timestamp,
		"attempt":    attempt,
		"payload":    payload,
	}
}

// TransformPayload applies the template if enabled
func (e *TemplateEngine) TransformPayload(tmplStr string, data WebhookTemplateContext) ([]byte, error) {
	if tmplStr == "" {
		return nil, nil
	}
	return e.Execute(tmplStr, data)
}
