package template

import (
	"encoding/json"
	"strings"
	"testing"
)

// FuzzExecute runs arbitrary templates against arbitrary JSON payloads.
// Templates are written by API users and run on the server, so for any input
// the engine must not panic, must keep output within MaxOutputBytes, and
// must agree with Validate on whether the template parses.
func FuzzExecute(f *testing.F) {
	seeds := []struct{ tmpl, payload string }{
		{`{{.payload.name}}`, `{"name":"a"}`},
		{`{"text": {{json .payload}}}`, `{"a":[1,2,{"b":null}]}`},
		{`{{range .payload.items}}{{.}},{{end}}`, `{"items":[1,"x",true]}`},
		{`{{printf "%5d|%-8s" 3 "x"}}`, `{}`},
		{`{{printf "%999999999d" 1}}`, `{}`},
		{`{{$s := "ab"}}{{range .payload.items}}{{$s = print $s $s}}{{end}}{{len $s}}`, `{"items":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30]}`},
		{`{{define "x"}}{{template "x" .}}{{end}}{{template "x" .}}`, `{}`},
		{`{{range 2000000000}}{{end}}`, `{}`},
		{`{{range .payload.a}}{{range .}}{{range .}}{{range .}}{{end}}{{end}}{{end}}{{end}}`, `{"a":[[[[1]]]]}`},
		{`{{dig "a" "b" "none" .payload}} {{default "d" .payload.missing}}`, `{"a":{"b":"c"}}`},
		{`{{repeat 100000 "xxxxxxxxxx"}}`, `{}`},
		{`{{.event_name}} {{.attempt}} {{index .payload "k"}}`, `{"k":"v"}`},
		{`{{`, `{}`},
	}
	for _, s := range seeds {
		f.Add(s.tmpl, []byte(s.payload), false)
	}
	engine := New()
	f.Fuzz(func(t *testing.T, tmpl string, payload []byte, strict bool) {
		var data map[string]any
		if json.Unmarshal(payload, &data) != nil {
			data = map[string]any{}
		}
		ctx := NewWebhookContext("evt_1", "order.created", "2026-01-01T00:00:00Z", 1, data)

		out, err := engine.ExecuteWith(tmpl, ctx, ExecOptions{StrictMissingKeys: strict})
		if len(out) > MaxOutputBytes {
			t.Fatalf("output is %d bytes, over MaxOutputBytes", len(out))
		}
		if err != nil && strings.Contains(err.Error(), "timed out") {
			t.Fatalf("template ran past ExecutionTimeout; the step/work budget should stop it first: %q", tmpl)
		}

		parseErr := err != nil && strings.HasPrefix(err.Error(), "failed to parse template")
		if tmpl == "" {
			return
		}
		if valid := engine.Validate(tmpl) == nil; valid == parseErr {
			t.Fatalf("Validate=%v but ExecuteWith parse error=%v (%v)", valid, parseErr, err)
		}
	})
}
