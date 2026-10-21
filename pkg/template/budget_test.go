package template

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// Each of these spun a core or ate gigabytes before the budget; each must now
// fail fast with an error naming the limit it hit.
func TestExecuteStopsRunawayTemplates(t *testing.T) {
	payload := map[string]any{"n": 2000000000.0, "items": make([]any, 2000)}
	cases := []struct {
		name, tmpl, wantErr string
	}{
		{"range over int variable", `{{$n := 2000000000}}{{range $n}}{{end}}`, "range over an integer"},
		{"range over computed int", `{{range toInt .payload.n}}{{end}}`, "range over an integer"},
		{"range over len", `{{range len "abc"}}{{end}}`, "range over an integer"},
		{"string doubling with print", `{{$s := "xxxxxxxx"}}{{range .payload.items}}{{$s = print $s $s}}{{end}}`, "intermediate values"},
		{"string doubling with js", `{{$s := "\\"}}{{range .payload.items}}{{$s = js $s}}{{end}}`, "intermediate values"},
		{"structure doubling", `{{$l := list 1}}{{range .payload.items}}{{$l = list $l $l}}{{end}}{{json $l}}`, "intermediate values"},
		{"dict doubling", `{{$m := dict}}{{range .payload.items}}{{$m = dict "a" $m "b" $m}}{{end}}{{$m}}`, "intermediate values"},
		{"repeat of a large string", `{{$s := repeat 1000 "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}}{{$t := repeat 1000 $s}}`, "intermediate values"},
		{"replace with empty old", `{{$s := repeat 1000 "xxxxxxxxxxxxxxxx"}}{{$t := replace "" $s $s}}`, "intermediate values"},
		{"printf width", `{{printf "%999999999d" 1}}`, "width and precision"},
		{"printf star width", `{{printf "%*d" 999999999 1}}`, "not allowed"},
		{"nested loops over a payload array", `{{range .payload.items}}{{range $.payload.items}}{{range $.payload.items}}{{end}}{{end}}{{end}}`, "loop iterations"},
		{"exponential template recursion", `{{define "a"}}{{if lt (len .) 12}}{{$n := print . "x"}}{{template "a" $n}}{{template "a" $n}}{{template "a" $n}}{{template "a" $n}}{{end}}{{end}}{{template "a" ""}}`, "loop iterations"},
	}
	// The step cases would need 1M steps; at this limit they stop in
	// milliseconds even under -race, so the timing check below holds.
	stepLimit = 20_000
	t.Cleanup(func() { stepLimit = MaxSteps })
	engine := New()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			_, err := engine.ExecuteWith(tc.tmpl, map[string]any{"payload": payload}, ExecOptions{})
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not mention %q", err, tc.wantErr)
			}
			if d := time.Since(start); d > 3*time.Second {
				t.Fatalf("took %v; the budget should stop it well before the timeout", d)
			}
		})
	}
}

// The budget must not change what ordinary templates render.
func TestExecuteBudgetKeepsNormalTemplates(t *testing.T) {
	payload := map[string]any{
		"items": []any{map[string]any{"id": "a"}, map[string]any{"id": "b"}},
		"tags":  "x,y",
		"m":     map[string]any{"k": "v"},
	}
	cases := []struct {
		name, tmpl, want string
		strict           bool
	}{
		{"range over payload array", `{{range $i, $e := .payload.items}}{{$i}}={{$e.id}};{{end}}`, "0=a;1=b;", false},
		{"range else on empty", `{{range .payload.none}}x{{else}}empty{{end}}`, "empty", false},
		{"range over missing key, strict, via index", `{{range index .payload "none"}}x{{else}}empty{{end}}`, "empty", true},
		{"range over map", `{{range $k, $v := .payload.m}}{{$k}}:{{$v}}{{end}}`, "k:v", false},
		{"range over split", `{{range split "," .payload.tags}}[{{.}}]{{end}}`, "[x][y]", false},
		{"range with break", `{{range .payload.items}}{{.id}}{{break}}{{end}}`, "a", false},
		{"accumulate with append", `{{$o := list}}{{range .payload.items}}{{$o = append $o .id}}{{end}}{{json $o}}`, `["a","b"]`, false},
		{"printf", `{{printf "%05.1f|%-4s|" 3.14159 "ab"}}`, "003.1|ab  |", false},
		{"define and template", `{{define "x"}}<{{.}}>{{end}}{{template "x" "y"}}`, "<y>", false},
		{"print builtins", `{{print "a" 1}} {{println "b"}}{{html "<"}} {{js "'"}} {{urlquery "a b"}}`, "a1 b\n&lt; \\' a+b", false},
	}
	engine := New()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := engine.ExecuteWith(tc.tmpl, map[string]any{"payload": payload}, ExecOptions{StrictMissingKeys: tc.strict})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(out) != tc.want {
				t.Fatalf("got %q, want %q", out, tc.want)
			}
		})
	}
}

// Concurrent executions share a fixed number of slots; every caller still
// gets its result, and a template that is stopped frees its slot.
func TestExecuteBoundsConcurrency(t *testing.T) {
	engine := New()
	n := 4 * cap(execSlots)
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for range n {
		wg.Go(func() {
			if _, err := engine.ExecuteWith(`{{range .}}{{.}}{{end}}`, []any{1, 2, 3}, ExecOptions{}); err != nil {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent execution failed: %v", err)
	}
	if len(execSlots) != 0 {
		t.Fatalf("%d execution slots still held after all executions returned", len(execSlots))
	}
}

func TestCheckPrintfFormat(t *testing.T) {
	for format, ok := range map[string]bool{
		"%d": true, "%5.2f": true, "%-10s|%+v": true, "100%%": true, "%[1]d": true,
		"%1001d": false, "%.99999f": false, "%*d": false, "%.*f": false,
	} {
		if err := checkPrintfFormat(format); (err == nil) != ok {
			t.Errorf("checkPrintfFormat(%q) = %v, want ok=%v", format, err, ok)
		}
	}
}

type slowData struct{ d time.Duration }

func (s slowData) Slow() string { time.Sleep(s.d); return "done" }

// Waiting for a slot does not count against the run's timeout: a template
// that gets its slot late still has the whole timeout to render.
func TestExecuteTimeoutStartsWithTheSlot(t *testing.T) {
	defer func(d time.Duration) { execTimeout = d }(execTimeout)
	execTimeout = 400 * time.Millisecond

	for range cap(execSlots) {
		execSlots <- struct{}{}
	}
	go func() {
		time.Sleep(300 * time.Millisecond) // most of the timeout spent waiting
		for range cap(execSlots) {
			<-execSlots
		}
	}()

	out, err := New().ExecuteWith(`{{.Slow}}`, slowData{200 * time.Millisecond}, ExecOptions{})
	if err != nil || string(out) != "done" {
		t.Fatalf("out %q, err %v: want the render to get its own timeout after the wait", out, err)
	}
}
