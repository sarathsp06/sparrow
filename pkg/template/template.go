package template

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"text/template"
	"text/template/parse"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// cacheSize is the maximum number of parsed templates an engine keeps.
const cacheSize = 100

// MaxOutputBytes is the maximum allowed output size from a template
// execution. Templates producing output larger than this are aborted to
// prevent denial-of-service via crafted templates.
const MaxOutputBytes = 1 * 1024 * 1024 // 1 MB

// ExecutionTimeout bounds how long a template runs, and separately how long
// a caller waits for an execution slot (ErrBusy). Go's text/template has no
// cancellation, so Execute runs in a goroutine; on timeout the caller stops
// waiting and the execution ends at its next loop iteration, template call or
// string-building helper (see budget.go). MaxSteps and
// MaxWorkBytes bound what it can do until then.
const ExecutionTimeout = 5 * time.Second

// execTimeout is ExecutionTimeout, shortened by tests.
var execTimeout = ExecutionTimeout

// MaxRangeNesting is the deepest allowed nesting of range actions. Each
// level multiplies the iteration count, so deep nesting over even modest
// payload arrays can spin for a very long time without writing output.
const MaxRangeNesting = 3

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

// Engine handles payload transformations using Go templates
type Engine struct {
	// base is the documented helper set; funcs adds the budget-aware
	// replacements, which each execution rebinds to its own budget.
	base  template.FuncMap
	funcs template.FuncMap
	cache *lru.Cache[string, *template.Template]
}

// New creates a new template engine with default helpers
func New() *Engine {
	// lru.New only fails for a non-positive size.
	cache, _ := lru.New[string, *template.Template](cacheSize)
	base := FuncMap()
	funcs := make(template.FuncMap, len(base))
	for name, fn := range base {
		funcs[name] = fn
	}
	for name, fn := range budgetFuncs(&budget{}, base) {
		funcs[name] = fn
	}
	return &Engine{base: base, funcs: funcs, cache: cache}
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
// Output is limited to MaxOutputBytes, work to MaxSteps and
// MaxWorkBytes, and the run to ExecutionTimeout, to prevent
// denial-of-service via crafted templates that consume unbounded CPU or
// memory. It returns ErrBusy when no execution slot frees up within
// ExecutionTimeout; the run's own timeout starts once it has a slot, so
// waiting under load never counts against the template.
func (e *Engine) ExecuteWith(tmplStr string, data any, opts ExecOptions) ([]byte, error) {
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
	lw := &limitedWriter{buf: buf, limit: MaxOutputBytes}

	wait := time.NewTimer(execTimeout)
	select {
	case execSlots <- struct{}{}:
		wait.Stop()
	case <-wait.C:
		return nil, ErrBusy
	}

	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()

	// Each execution gets its own budget: the cached template is shared, so
	// the budget-aware helpers are rebound on a clone (which shares the
	// parse trees).
	b := &budget{}
	run, err := tmpl.Clone()
	if err != nil {
		<-execSlots
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	run.Funcs(budgetFuncs(b, e.base))

	// SEC: Go's text/template has no cancellation. On timeout the caller
	// stops waiting and sets stopped, which ends the execution at its next
	// step or helper call; the slot is released only when it really ends.
	execErr := make(chan error, 1)
	go func() {
		defer func() { <-execSlots }()
		execErr <- run.Execute(lw, data)
	}()

	select {
	case err := <-execErr:
		if err != nil {
			return nil, fmt.Errorf("failed to execute template: %w", err)
		}
	case <-ctx.Done():
		b.stopped.Store(true)
		return nil, fmt.Errorf("template execution timed out after %v (possible infinite loop)", execTimeout)
	}

	return buf.Bytes(), nil
}

// Validate parses tmplStr with the same helpers and limits ExecuteWith
// uses, without executing it. It catches what is knowable without data
// (syntax errors, unknown helpers, rejected loops); errors that depend on the
// payload, such as a missing key, only show at execution.
func (e *Engine) Validate(tmplStr string) error {
	_, err := e.getOrParseTemplate(tmplStr, false)
	return err
}

// getOrParseTemplate retrieves a cached template or parses and caches a new
// one. Strict and lenient parses of the same source are cached separately,
// since the missingkey option is set on the parsed template.
func (e *Engine) getOrParseTemplate(tmplStr string, strict bool) (*template.Template, error) {
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
	instrument(tmpl)
	if strict {
		tmpl = tmpl.Option("missingkey=error")
	}

	// Add to cache
	e.cache.Add(key, tmpl)

	return tmpl, nil
}

// checkTemplateLoops rejects at parse time what the budget would only stop at
// run time: range over an integer literal ({{range 2000000000}}{{end}}) and
// range nested deeper than MaxRangeNesting. It walks every template defined in the parse,
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

// WebhookContext is the data context passed to Go templates.
// It uses snake_case keys so that templates reference fields as
// {{.event_id}}, {{.event_name}}, {{.timestamp}}, {{.attempt}}, and
// {{.payload}}.
// Consumer, WebhookID, and DeliveryID are conveyed via HTTP headers
// and are intentionally omitted from the template context.
type WebhookContext = map[string]any

// NewWebhookContext builds a template context map with snake_case keys.
func NewWebhookContext(eventID, eventName, timestamp string, attempt int, payload map[string]any) WebhookContext {
	return WebhookContext{
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
