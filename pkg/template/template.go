package template

import (
	"context"
	"fmt"
	"text/template"
	"text/template/parse"
	"time"
)

// MaxTemplateOutputBytes is the maximum allowed output size from a template
// execution. Templates producing output larger than this are aborted to
// prevent denial-of-service via crafted templates.
const MaxTemplateOutputBytes = 1 * 1024 * 1024 // 1 MB

// TemplateExecutionTimeout bounds how long a caller waits for a template.
// Go's text/template has no cancellation, so Execute runs in a goroutine
// that is abandoned on timeout and keeps running until it finishes or hits
// MaxTemplateOutputBytes. The timeout therefore frees the caller, not the
// CPU: loops that produce no output are rejected at parse time instead
// (see checkTemplateLoops).
const TemplateExecutionTimeout = 5 * time.Second

// MaxRangeNesting is the deepest allowed nesting of range actions. Each
// level multiplies the iteration count, so deep nesting over even modest
// payload arrays can spin for a very long time without writing output.
const MaxRangeNesting = 3

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

// Execute processes a template with the given data, rendering missing keys
// as "<no value>". See ExecuteWith for strict rendering.
func (e *TemplateEngine) Execute(tmplStr string, data any) ([]byte, error) {
	return e.ExecuteWith(tmplStr, data, ExecOptions{})
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

	// Get buffer from pool
	buf := getBuffer()

	// Wrap with a size-limited writer to prevent runaway output
	lw := &limitedWriter{buf: buf, limit: MaxTemplateOutputBytes}

	// SEC: Wait at most TemplateExecutionTimeout. Go's text/template has no
	// cancellation, so on timeout the goroutine is abandoned and keeps
	// running until it completes or hits the output size limit. Loops that
	// could run without output are rejected at parse time (checkTemplateLoops).
	ctx, cancel := context.WithTimeout(context.Background(), TemplateExecutionTimeout)
	defer cancel()

	execErr := make(chan error, 1)
	go func() {
		execErr <- tmpl.Execute(lw, data)
	}()

	select {
	case err := <-execErr:
		// Goroutine finished — we own the buffer, safe to pool it.
		defer putBuffer(buf)
		if err != nil {
			return nil, fmt.Errorf("failed to execute template: %w", err)
		}
	case <-ctx.Done():
		// Timeout — the goroutine is still running and writing to buf.
		// We must NOT return buf to the pool; let it be GC'd when the
		// goroutine finishes.
		return nil, fmt.Errorf("template execution timed out after %v (possible infinite loop)", TemplateExecutionTimeout)
	}

	// Copy bytes since we're returning the buffer to the pool
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
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
	if err := checkTemplateLoops(tmpl); err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	if strict {
		tmpl = tmpl.Option("missingkey=error")
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

	tmpl, err := template.New("webhook_validation").Funcs(e.funcs).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("invalid template syntax: %w", err)
	}
	if err := checkTemplateLoops(tmpl); err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}

	return nil
}

// checkTemplateLoops rejects loops whose cost does not depend on the payload
// and cannot be stopped once started: range over an integer literal
// ({{range 2000000000}}{{end}} spins without output, so neither the output
// limit nor the abandoned-goroutine timeout ends it) and range nested deeper
// than MaxRangeNesting. It walks every template defined in the parse,
// including {{define}} blocks.
func checkTemplateLoops(tmpl *template.Template) error {
	for _, t := range tmpl.Templates() {
		if t.Tree == nil {
			continue
		}
		if err := checkLoopNode(t.Root, 0); err != nil {
			return err
		}
	}
	return nil
}

func checkLoopNode(node parse.Node, depth int) error {
	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return nil
		}
		for _, c := range n.Nodes {
			if err := checkLoopNode(c, depth); err != nil {
				return err
			}
		}
	case *parse.RangeNode:
		if depth+1 > MaxRangeNesting {
			return fmt.Errorf("range nested deeper than %d levels", MaxRangeNesting)
		}
		if rangesOverNumberLiteral(n.Pipe) {
			return fmt.Errorf("range over an integer literal is not allowed")
		}
		if err := checkLoopNode(n.List, depth+1); err != nil {
			return err
		}
		return checkLoopNode(n.ElseList, depth)
	case *parse.IfNode:
		if err := checkLoopNode(n.List, depth); err != nil {
			return err
		}
		return checkLoopNode(n.ElseList, depth)
	case *parse.WithNode:
		if err := checkLoopNode(n.List, depth); err != nil {
			return err
		}
		return checkLoopNode(n.ElseList, depth)
	}
	return nil
}

// rangesOverNumberLiteral reports whether a range pipeline is a bare number,
// possibly parenthesised: {{range 5}} or {{range (5)}}.
func rangesOverNumberLiteral(pipe *parse.PipeNode) bool {
	for pipe != nil && len(pipe.Cmds) == 1 && len(pipe.Cmds[0].Args) == 1 {
		switch arg := pipe.Cmds[0].Args[0].(type) {
		case *parse.NumberNode:
			return true
		case *parse.PipeNode:
			pipe = arg
		default:
			return false
		}
	}
	return false
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

// TransformPayload applies the template if enabled
func (e *TemplateEngine) TransformPayload(tmplStr string, data WebhookTemplateContext) ([]byte, error) {
	return e.TransformPayloadWith(tmplStr, data, ExecOptions{})
}

// TransformPayloadWith applies the template with the given options.
func (e *TemplateEngine) TransformPayloadWith(tmplStr string, data WebhookTemplateContext, opts ExecOptions) ([]byte, error) {
	if tmplStr == "" {
		return nil, nil
	}
	return e.ExecuteWith(tmplStr, data, opts)
}
