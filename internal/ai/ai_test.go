package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

// fakeModel serves the Messages API shape: each call pops the next canned
// reply and records the request body it received.
type fakeModel struct {
	replies  []string // structured-output JSON per call
	refuse   bool
	requests []map[string]any
}

func (f *fakeModel) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("bad request body: %v", err)
		}
		f.requests = append(f.requests, req)
		if len(f.replies) == 0 {
			t.Fatalf("unexpected extra model call")
		}
		reply := f.replies[0]
		f.replies = f.replies[1:]
		stop := "end_turn"
		if f.refuse {
			stop = "refusal"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": "test-model",
			"content":     []map[string]any{{"type": "text", "text": reply}},
			"stop_reason": stop,
			"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}
}

func newDrafter(t *testing.T, f *fakeModel, render Renderer) *Drafter {
	t.Helper()
	srv := httptest.NewServer(f.handler(t))
	t.Cleanup(srv.Close)
	d, err := New(Config{APIKey: "test", Model: "test-model", BaseURL: srv.URL, MaxAttempts: 3}, render,
		[]HelperFunc{{Name: "json", Description: "# json\n\nConverts a value to JSON."}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDraftTemplate_RepairsUntilItRenders(t *testing.T) {
	f := &fakeModel{replies: []string{
		`{"template":"{{ .Payload.id }}","notes":"first"}`,
		`{"template":"{{ .payload.id | json }}","notes":"fixed"}`,
	}}
	var seen []string
	render := func(_ context.Context, eventName, tmpl string, _ map[string]any) (string, error) {
		seen = append(seen, tmpl)
		if strings.Contains(tmpl, ".Payload") {
			return "", errors.New(`template: :1:3: executing "" at <.Payload.id>: map has no entry for key "Payload"`)
		}
		return `"ord_1"`, nil
	}
	d := newDrafter(t, f, render)

	res, err := d.DraftTemplate(context.Background(), Request{
		EventName:     "order.created",
		SamplePayload: map[string]any{"id": "ord_1"},
		Instructions:  "just the id",
	})
	if err != nil {
		t.Fatalf("DraftTemplate: %v", err)
	}
	if res.Attempts != 2 || res.Template != "{{ .payload.id | json }}" || res.Rendered != `"ord_1"` || res.Notes != "fixed" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(seen) != 2 {
		t.Fatalf("renderer called %d times, want 2", len(seen))
	}

	// The repair turn carries the render error back to the model, and every
	// call asks for structured JSON output.
	if len(f.requests) != 2 {
		t.Fatalf("model called %d times, want 2", len(f.requests))
	}
	second, _ := json.Marshal(f.requests[1]["messages"])
	if !strings.Contains(string(second), "map has no entry for key") {
		t.Fatalf("repair turn does not include the render error: %s", second)
	}
	for i, r := range f.requests {
		oc, _ := r["output_config"].(map[string]any)
		format, _ := oc["format"].(map[string]any)
		if format["type"] != "json_schema" {
			t.Fatalf("request %d missing json_schema output format: %v", i, r["output_config"])
		}
		if r["model"] != "test-model" {
			t.Fatalf("request %d model = %v", i, r["model"])
		}
	}
	// The sample payload and instructions are in the first user turn.
	first, _ := json.Marshal(f.requests[0]["messages"])
	if !strings.Contains(string(first), "ord_1") || !strings.Contains(string(first), "just the id") {
		t.Fatalf("first turn missing grounding: %s", first)
	}
}

func TestCheckRendered(t *testing.T) {
	cases := []struct {
		name, rendered string
		wantErr        string
	}{
		{"valid json", `{"text": "Order ord_1"}`, ""},
		{"plain text", `Order ord_1 for 42.5`, ""},
		{"missing key", `{"text": "Order <no value>"}`, "does not exist"},
		{"nil in printf", `Order %!s(<nil>) for 42.5`, "does not exist"},
		{"marker escaped by json helper", `{"text": "Order %!s(\u003cnil\u003e)"}`, "does not exist"},
		{"no value escaped by json helper", `{"text": "\u003cno value\u003e"}`, "does not exist"},
		{"hand-quoted json", `{"text": "He said "hi""}`, "not valid JSON"},
		{"trailing comma", `{"a": 1,}`, "not valid JSON"},
		{"json array", `[{"id": "1"}]`, ""},
		{"empty", ``, ""},
	}
	for _, c := range cases {
		err := checkRendered(c.rendered)
		switch {
		case c.wantErr == "" && err != nil:
			t.Errorf("%s: unexpected error %v", c.name, err)
		case c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)):
			t.Errorf("%s: error %v, want containing %q", c.name, err, c.wantErr)
		}
	}
}

func TestDraftTemplate_RepairsSilentlyWrongField(t *testing.T) {
	// The renderer never errors (like the real engine on a missing key); the
	// post-render check must still push the draft back for repair.
	f := &fakeModel{replies: []string{
		`{"template":"{\"t\": {{ printf \"%s\" .Payload.id | json }}}","notes":""}`,
		`{"template":"{\"t\": {{ .payload.id | json }}}","notes":"ok"}`,
	}}
	render := func(_ context.Context, _, tmpl string, _ map[string]any) (string, error) {
		if strings.Contains(tmpl, ".Payload") {
			return `{"t": "%!s(<nil>)"}`, nil
		}
		return `{"t": "ord_1"}`, nil
	}
	d := newDrafter(t, f, render)
	res, err := d.DraftTemplate(context.Background(), Request{EventName: "e", Instructions: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Attempts != 2 || res.Rendered != `{"t": "ord_1"}` {
		t.Fatalf("unexpected result %+v", res)
	}
	second, _ := json.Marshal(f.requests[1]["messages"])
	if !strings.Contains(string(second), "does not exist in the data") {
		t.Fatalf("repair turn lacks the missing-field explanation: %s", second)
	}
}

func TestDraftTemplate_RendersAgainstRequestPayload(t *testing.T) {
	f := &fakeModel{replies: []string{`{"template":"{{ .payload.id | json }}","notes":""}`}}
	var got map[string]any
	render := func(_ context.Context, _, _ string, payload map[string]any) (string, error) {
		got = payload
		return `"x"`, nil
	}
	d := newDrafter(t, f, render)
	if _, err := d.DraftTemplate(context.Background(), Request{EventName: "e", Instructions: "x", SamplePayload: map[string]any{"id": "provided"}}); err != nil {
		t.Fatal(err)
	}
	if got["id"] != "provided" {
		t.Fatalf("renderer got %v, want the request's sample payload", got)
	}
}

func TestDraftTemplate_DocsURL(t *testing.T) {
	f := &fakeModel{replies: []string{`{"template":"{{ .payload.id | json }}","notes":""}`}}
	srv := httptest.NewServer(f.handler(t))
	t.Cleanup(srv.Close)
	render := func(context.Context, string, string, map[string]any) (string, error) { return `"x"`, nil }

	// No fetcher configured: docs_url is refused before any model call.
	d, _ := New(Config{APIKey: "k", Model: "m", BaseURL: srv.URL}, render, nil, nil)
	_, err := d.DraftTemplate(context.Background(), Request{EventName: "e", Instructions: "x", DocsURL: "https://example.com"})
	if got := svcerrors.Classify(err, "").Status; got != svcerrors.InvalidArgument || len(f.requests) != 0 {
		t.Fatalf("without fetcher: status=%v calls=%d", got, len(f.requests))
	}

	// Fetcher error surfaces as InvalidArgument naming docs_url.
	failing := func(context.Context, string) (string, error) {
		return "", errors.New("host resolves to blocked address")
	}
	d, _ = New(Config{APIKey: "k", Model: "m", BaseURL: srv.URL}, render, nil, failing)
	_, err = d.DraftTemplate(context.Background(), Request{EventName: "e", Instructions: "x", DocsURL: "https://example.com"})
	if err == nil || !strings.Contains(err.Error(), "docs_url") || !strings.Contains(err.Error(), "blocked address") {
		t.Fatalf("fetch error not surfaced: %v", err)
	}

	// Fetched text reaches the prompt.
	ok := func(_ context.Context, u string) (string, error) { return "Body: {title, body}. From " + u, nil }
	d, _ = New(Config{APIKey: "k", Model: "m", BaseURL: srv.URL}, render, nil, ok)
	if _, err = d.DraftTemplate(context.Background(), Request{EventName: "e", Instructions: "x", DocsURL: "https://example.com/docs"}); err != nil {
		t.Fatal(err)
	}
	first, _ := json.Marshal(f.requests[0]["messages"])
	if !strings.Contains(string(first), "Body: {title, body}") || !strings.Contains(string(first), "Receiver documentation") {
		t.Fatalf("docs text missing from prompt: %s", first)
	}
}

func TestDraftTemplate_GivesUpAfterMaxAttempts(t *testing.T) {
	f := &fakeModel{replies: []string{
		`{"template":"{{ bad }}","notes":""}`,
		`{"template":"{{ bad }}","notes":""}`,
		`{"template":"{{ bad }}","notes":""}`,
	}}
	render := func(context.Context, string, string, map[string]any) (string, error) {
		return "", errors.New(`function "bad" not defined`)
	}
	d := newDrafter(t, f, render)
	_, err := d.DraftTemplate(context.Background(), Request{EventName: "e", Instructions: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := svcerrors.Classify(err, "").Status; got != svcerrors.InvalidArgument {
		t.Fatalf("status = %v, want InvalidArgument", got)
	}
	if !strings.Contains(err.Error(), "after 3 attempts") {
		t.Fatalf("error = %v", err)
	}
}

func TestDraftTemplate_RefusalIsFailedPrecondition(t *testing.T) {
	f := &fakeModel{replies: []string{`{"template":"x","notes":""}`}, refuse: true}
	d := newDrafter(t, f, func(context.Context, string, string, map[string]any) (string, error) { return "", nil })
	_, err := d.DraftTemplate(context.Background(), Request{EventName: "e", Instructions: "x"})
	if got := svcerrors.Classify(err, "").Status; got != svcerrors.FailedPrecondition {
		t.Fatalf("status = %v, want FailedPrecondition (err=%v)", got, err)
	}
}

func TestDraftTemplate_ValidatesInput(t *testing.T) {
	f := &fakeModel{}
	d := newDrafter(t, f, func(context.Context, string, string, map[string]any) (string, error) { return "", nil })
	for _, req := range []Request{{EventName: "e"}, {Instructions: "x"}} {
		_, err := d.DraftTemplate(context.Background(), req)
		if got := svcerrors.Classify(err, "").Status; got != svcerrors.InvalidArgument {
			t.Fatalf("req %+v: status = %v, want InvalidArgument", req, got)
		}
	}
	if len(f.requests) != 0 {
		t.Fatal("model should not be called for invalid input")
	}
}

func TestPrompts_IncludeRecipeExampleAndCurrent(t *testing.T) {
	d := NewPromptBuilder([]HelperFunc{{Name: "json", Description: "doc"}})
	sys := d.systemPrompt(false)
	if !strings.Contains(sys, ".payload") || !strings.Contains(sys, "### json") || !strings.Contains(sys, "JSON object") {
		t.Fatalf("system prompt missing data model or helper catalog:\n%s", sys)
	}
	user := d.userPrompt(Request{
		DocsURL:         "https://docs.example.com/hooks",
		EventName:       "order.created",
		Schema:          map[string]any{"type": "object"},
		SamplePayload:   map[string]any{"id": "1"},
		Instructions:    "do it",
		RecipeName:      "slack",
		RecipeTemplate:  `{"blocks": []}`,
		TargetExample:   `{"text": "..."}`,
		CurrentTemplate: `{{ .payload.id }}`,
	}, "POST a JSON body with title and body fields")
	chat := d.ChatPrompt(Request{EventName: "e", Instructions: "do it", SamplePayload: map[string]any{"id": "1"}}, "")
	if strings.Contains(chat, "JSON object") || !strings.Contains(chat, "fenced code block") || !strings.Contains(chat, "=== Your task ===") || !strings.Contains(chat, "### json") {
		t.Fatalf("chat prompt should be self-contained and ask for a code block:\n%s", chat)
	}
	for _, want := range []string{"docs.example.com/hooks", "title and body fields", "order.created", `"type": "object"`, `"id": "1"`, "Destination recipe: slack", `{"blocks": []}`, `{"text": "..."}`, "Current template to refine", "do it"} {
		if !strings.Contains(user, want) {
			t.Fatalf("user prompt missing %q:\n%s", want, user)
		}
	}
}

func TestNew_RequiresConfig(t *testing.T) {
	r := func(context.Context, string, string, map[string]any) (string, error) { return "", nil }
	if _, err := New(Config{Model: "m"}, r, nil, nil); err == nil {
		t.Fatal("expected error without api key")
	}
	if _, err := New(Config{APIKey: "k"}, r, nil, nil); err == nil {
		t.Fatal("expected error without model")
	}
	if _, err := New(Config{APIKey: "k", Model: "m"}, nil, nil, nil); err == nil {
		t.Fatal("expected error without renderer")
	}
}
