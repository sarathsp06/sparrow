package template

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"text/template"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// cacheSize is the maximum number of parsed templates an engine keeps.
const cacheSize = 100

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
	buf     *bytes.Buffer
	limit   int
	written int
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
	cache *lru.Cache[string, *template.Template]
}

// NewTemplateEngine creates a new template engine with default helpers
func NewTemplateEngine() *TemplateEngine {
	// lru.New only fails for a non-positive size.
	cache, _ := lru.New[string, *template.Template](cacheSize)
	return &TemplateEngine{funcs: GetFunctionMap(), cache: cache}
}

// ExecOptions controls how a template is executed.
type ExecOptions struct {
	// StrictMissingKeys makes reading a key the data does not have an error
	// (text/template's missingkey=error) instead of rendering "<no value>".
	// With it, a template that reads a field removed from the payload fails
	// instead of silently producing a wrong body. Read optional fields with
	// index or dig, which never error on a missing key. default does not
	// help here: the lookup in `default "x" .payload.plan` fails before
	// default runs; use `dig "plan" "x" .payload` instead.
	StrictMissingKeys bool
}

// ExecuteWith processes a template with the given data and options.
// Output is limited to MaxTemplateOutputBytes and execution time is limited
// to TemplateExecutionTimeout to prevent denial-of-service via crafted
// templates that consume unbounded CPU or produce unbounded output.
func (e *TemplateEngine) ExecuteWith(tmplStr string, data any, opts ExecOptions) ([]byte, error) {
	if tmplStr == "" {
		return nil, nil
	}

	// Get or create cached template
	tmpl, err := e.getOrParseTemplate(tmplStr, opts.StrictMissingKeys)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer([]byte{}) // non-nil Bytes() even for empty output

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

	return buf.Bytes(), nil
}

// getOrParseTemplate retrieves a cached template or parses and caches a new
// one. Strict and lenient parses of the same source are cached separately,
// since the missingkey option is set on the parsed template.
func (e *TemplateEngine) getOrParseTemplate(tmplStr string, strict bool) (*template.Template, error) {
	key := hashTemplate(tmplStr)
	if strict {
		key = hashTemplate("missingkey=error\x00" + tmplStr)
	}

	// Try to get from cache
	if tmpl, found := e.cache.Get(key); found {
		return tmpl, nil
	}

	// Parse new template
	tmpl, err := template.New("webhook").Funcs(e.funcs).Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	if strict {
		tmpl = tmpl.Option("missingkey=error")
	}

	// Add to cache
	e.cache.Add(key, tmpl)

	return tmpl, nil
}

// WebhookTemplateContext is the data context passed to Go templates.
// It uses snake_case keys so that templates reference fields as
// {{.event_id}}, {{.event_name}}, {{.timestamp}}, {{.attempt}}, and
// {{.payload}}.
// Consumer, WebhookID, and DeliveryID are conveyed via HTTP headers
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

// hashTemplate generates a cache key from the template string
func hashTemplate(tmplStr string) string {
	h := sha256.Sum256([]byte(tmplStr))
	return hex.EncodeToString(h[:])
}
