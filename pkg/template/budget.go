package template

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"text/template"
	"text/template/parse"
	"time"
)

// MaxSteps bounds one execution's range iterations plus template
// invocations. Every range body and every template body starts with a step,
// so nested loops over a large payload and recursive {{template}} calls stop
// here instead of spinning.
const MaxSteps = 1_000_000

// stepLimit is MaxSteps; tests lower it so they check that the budget stops a
// runaway template without depending on how fast the machine runs 1M steps
// (the race detector makes that several times slower).
var stepLimit = MaxSteps

// MaxWorkBytes bounds the bytes one execution's helpers may produce
// on the way to the output: strings built by print, printf, json, replace,
// repeat, ... and lists or maps built by list, dict and append. Without it a
// template can double a string held in a variable ({{$s = print $s $s}}) far
// past MaxOutputBytes without writing a byte.
const MaxWorkBytes = 16 << 20 // 16 MiB

// maxPrintfWidth bounds the width and precision of a printf verb: fmt pads
// in memory, so "%999999999d" allocates a gigabyte from a tiny template.
const maxPrintfWidth = 1000

// ErrBusy is returned when no execution slot frees up within
// ExecutionTimeout. It says nothing about the template: retry later.
var ErrBusy = errors.New("too many templates are rendering; try again")

// execSlots bounds concurrent template executions across all engines in the
// process. A slot is held until the execution really ends, including one the
// caller stopped waiting for.
var execSlots = make(chan struct{}, max(4, 2*runtime.GOMAXPROCS(0)))

// Names of the functions the engine injects into every parsed template.
// They are ordinary template functions, so a template may call them too.
const (
	stepFunc       = "_sparrowStep"
	rangeGuardFunc = "_sparrowRangeable"
)

// budget is one execution's allowance. Only the executing goroutine touches
// steps and bytes; stopped is set by the caller when it stops waiting, so the
// abandoned execution ends at its next step or helper call.
type budget struct {
	steps   int
	bytes   int
	stopped atomic.Bool
}

var errStopped = errors.New("template execution stopped")

func (b *budget) step() error {
	if b.stopped.Load() {
		return errStopped
	}
	b.steps++
	if b.steps > stepLimit {
		return fmt.Errorf("template exceeded %d loop iterations and template calls", stepLimit)
	}
	return nil
}

// charge records n bytes produced by a helper.
func (b *budget) charge(n int) error {
	if b.stopped.Load() {
		return errStopped
	}
	if n > MaxWorkBytes-b.bytes {
		b.bytes = MaxWorkBytes
		return fmt.Errorf("template built more than %d bytes of intermediate values", MaxWorkBytes)
	}
	b.bytes += n
	return nil
}

// remaining is how many bytes may still be charged.
func (b *budget) remaining() int { return MaxWorkBytes - b.bytes }

func (b *budget) str(s string, err error) (string, error) {
	if err != nil {
		return "", err
	}
	if err := b.charge(len(s)); err != nil {
		return "", err
	}
	return s, nil
}

// sizeOf estimates the bytes v takes when printed or encoded, walking at most
// limit+1 of them: a value built by sharing ($l = list $l $l) is far larger
// than the memory it occupies, so the walk must stop early.
func sizeOf(v any, limit int) int {
	n := 0
	var walk func(any)
	walk = func(v any) {
		if n > limit {
			return
		}
		switch x := v.(type) {
		case string:
			n += 16 + len(x)
		case []any:
			n += 24
			for _, e := range x {
				walk(e)
				if n > limit {
					return
				}
			}
		case map[string]any:
			n += 48
			for k, e := range x {
				n += 16 + len(k)
				walk(e)
				if n > limit {
					return
				}
			}
		case []string:
			n += 24
			for _, s := range x {
				n += 16 + len(s)
			}
		default:
			n += 16
		}
	}
	walk(v)
	return n
}

// chargeValue charges the size of a value a helper is about to wrap.
func (b *budget) chargeValue(v any) error {
	return b.charge(sizeOf(v, b.remaining()))
}

// checkPrintfFormat rejects verbs whose width or precision could make fmt
// allocate far more than the template's inputs: '*' (taken from an argument)
// and literals above maxPrintfWidth.
func checkPrintfFormat(format string) error {
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
	verb:
		for i++; i < len(format); i++ {
			switch c := format[i]; {
			case c == '*':
				return fmt.Errorf("printf: '*' width and precision are not allowed")
			case c >= '0' && c <= '9':
				j := i
				for j < len(format) && format[j] >= '0' && format[j] <= '9' {
					j++
				}
				if j-i > 4 || atoi(format[i:j]) > maxPrintfWidth {
					return fmt.Errorf("printf: width and precision are limited to %d", maxPrintfWidth)
				}
				i = j - 1
			case strings.IndexByte("+-# .[]", c) >= 0:
			default:
				break verb
			}
		}
	}
	return nil
}

func atoi(s string) int {
	n := 0
	for _, c := range []byte(s) {
		n = n*10 + int(c-'0')
	}
	return n
}

// budgetFuncs returns the helpers that consume b. They replace the builtins
// print, printf, println, html, js and urlquery and every helper that builds a
// string or a structure; the rest of the function map is unaffected.
func budgetFuncs(b *budget, base template.FuncMap) template.FuncMap {
	call1 := func(name string) func(string) (string, error) {
		f := base[name].(func(string) string)
		return func(s string) (string, error) { return b.str(f(s), nil) }
	}
	return template.FuncMap{
		stepFunc: func() (string, error) { return "", b.step() },
		rangeGuardFunc: func(v any) (any, error) {
			switch reflect.ValueOf(v).Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				return nil, fmt.Errorf("range over an integer is not allowed")
			case reflect.Func, reflect.Chan:
				return nil, fmt.Errorf("range over a %T is not allowed", v)
			}
			return v, nil
		},

		"print":   func(args ...any) (string, error) { return b.str(fmt.Sprint(args...), nil) },
		"println": func(args ...any) (string, error) { return b.str(fmt.Sprintln(args...), nil) },
		"printf": func(format string, args ...any) (string, error) {
			if err := checkPrintfFormat(format); err != nil {
				return "", err
			}
			return b.str(fmt.Sprintf(format, args...), nil)
		},
		"html":     func(args ...any) (string, error) { return b.str(template.HTMLEscaper(args...), nil) },
		"js":       func(args ...any) (string, error) { return b.str(template.JSEscaper(args...), nil) },
		"urlquery": func(args ...any) (string, error) { return b.str(template.URLQueryEscaper(args...), nil) },
		"toString": func(v any) (string, error) { return b.str(fmt.Sprintf("%v", v), nil) },

		"json": func(v any) (string, error) {
			// The encoding of a shared structure is far larger than its
			// memory; structures are charged when built, so this is bounded.
			out, err := json.Marshal(v)
			return b.str(string(out), err)
		},
		"urlencode":    call1("urlencode"),
		"base64":       call1("base64"),
		"upper":        call1("upper"),
		"lower":        call1("lower"),
		"title":        call1("title"),
		"base64decode": func(s string) (string, error) { return b.str(base[`base64decode`].(func(string) (string, error))(s)) },
		"formatTime":   func(layout string, t time.Time) (string, error) { return b.str(t.Format(layout), nil) },
		"replace": func(old, new, s string) (string, error) {
			// Size the result before building it: an empty old inserts new
			// between every character.
			if n := strings.Count(s, old); n > 0 && len(new) > len(old) && n > b.remaining()/(len(new)-len(old)) {
				return "", b.charge(b.remaining() + 1)
			}
			return b.str(strings.ReplaceAll(s, old, new), nil)
		},
		"join": func(sep string, elems []string) (string, error) {
			size := len(sep) * max(len(elems)-1, 0)
			for _, e := range elems {
				size += len(e)
			}
			if err := b.charge(size); err != nil {
				return "", err
			}
			return strings.Join(elems, sep), nil
		},
		"repeat": func(count int, s string) (string, error) {
			const maxRepeat = 1000
			if count < 0 {
				count = 0
			}
			if count > maxRepeat {
				return "", fmt.Errorf("repeat count %d exceeds maximum of %d", count, maxRepeat)
			}
			if err := b.charge(count * len(s)); err != nil {
				return "", err
			}
			return strings.Repeat(s, count), nil
		},
		"split": func(sep, s string) ([]string, error) {
			n := len(s) + 1
			if sep != "" {
				n = strings.Count(s, sep) + 1
			}
			if err := b.charge(16 * n); err != nil {
				return nil, err
			}
			return strings.Split(s, sep), nil
		},

		"list": func(items ...any) ([]any, error) {
			if err := b.chargeValue(items); err != nil {
				return nil, err
			}
			return items, nil
		},
		"append": func(list []any, items ...any) ([]any, error) {
			if err := b.chargeValue(items); err != nil {
				return nil, err
			}
			return append(list, items...), nil
		},
		"dict": func(pairs ...any) (map[string]any, error) {
			m, err := base["dict"].(func(...any) (map[string]any, error))(pairs...)
			if err != nil {
				return nil, err
			}
			if err := b.chargeValue(m); err != nil {
				return nil, err
			}
			return m, nil
		},
		"merge": func(maps ...map[string]any) (map[string]any, error) {
			// Values are shared with the inputs, which were charged when
			// built; only the new top level is new.
			out := base["merge"].(func(...map[string]any) map[string]any)(maps...)
			size := 48
			for k := range out {
				size += 32 + len(k)
			}
			if err := b.charge(size); err != nil {
				return nil, err
			}
			return out, nil
		},
	}
}

// instrument adds the budget to a parsed template: every template body and
// every range body starts with a step, and every range pipeline ends in the
// guard that refuses to iterate an integer. Range over an integer literal is
// already rejected at parse time; the guard catches integers computed at run
// time ({{$n := 2000000000}}{{range $n}}{{end}}), which spin without output.
func instrument(tmpl *template.Template) {
	for _, t := range tmpl.Templates() {
		if t.Tree == nil || t.Root == nil {
			continue
		}
		instrumentNode(t.Root)
		t.Root.Nodes = append([]parse.Node{stepAction(t.Root.Position())}, t.Root.Nodes...)
	}
}

func instrumentNode(node parse.Node) {
	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return
		}
		for _, c := range n.Nodes {
			instrumentNode(c)
		}
	case *parse.RangeNode:
		n.Pipe.Cmds = append(n.Pipe.Cmds, command(rangeGuardFunc, n.Position()))
		instrumentNode(n.List)
		instrumentNode(n.ElseList)
		if n.List != nil {
			n.List.Nodes = append([]parse.Node{stepAction(n.Position())}, n.List.Nodes...)
		}
	case *parse.IfNode:
		instrumentNode(n.List)
		instrumentNode(n.ElseList)
	case *parse.WithNode:
		instrumentNode(n.List)
		instrumentNode(n.ElseList)
	}
}

func command(fn string, pos parse.Pos) *parse.CommandNode {
	return &parse.CommandNode{NodeType: parse.NodeCommand, Pos: pos, Args: []parse.Node{parse.NewIdentifier(fn).SetPos(pos)}}
}

// stepAction is {{_sparrowStep}}, which prints nothing.
func stepAction(pos parse.Pos) *parse.ActionNode {
	return &parse.ActionNode{NodeType: parse.NodeAction, Pos: pos, Pipe: &parse.PipeNode{
		NodeType: parse.NodePipe, Pos: pos, Cmds: []*parse.CommandNode{command(stepFunc, pos)},
	}}
}
