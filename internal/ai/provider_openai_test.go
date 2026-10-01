package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

// fakeOpenAI serves /v1/chat/completions the way Ollama, vLLM or OpenAI do.
type fakeOpenAI struct {
	replies  []string
	status   int
	finish   string
	requests []map[string]any
	auth     []string
}

func (f *fakeOpenAI) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		f.auth = append(f.auth, r.Header.Get("Authorization"))
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		f.requests = append(f.requests, req)
		if f.status != 0 {
			w.WriteHeader(f.status)
			_, _ = w.Write([]byte(`{"error":{"message":"nope"}}`))
			return
		}
		reply := f.replies[0]
		f.replies = f.replies[1:]
		finish := f.finish
		if finish == "" {
			finish = "stop"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-1", "object": "chat.completion", "model": req["model"],
			"choices": []map[string]any{{"index": 0, "finish_reason": finish, "message": map[string]any{"role": "assistant", "content": reply}}},
		})
	}
}

func newOpenAIDrafter(t *testing.T, f *fakeOpenAI, apiKey string) *Drafter {
	t.Helper()
	srv := httptest.NewServer(f.handler(t))
	t.Cleanup(srv.Close)
	render := func(_ context.Context, _, tmpl string, _ map[string]any) (string, error) {
		if strings.Contains(tmpl, ".Payload") {
			return `{"t": "%!s(<nil>)"}`, nil
		}
		return `{"t": "ord_1"}`, nil
	}
	d, err := New(Config{Provider: ProviderOpenAI, APIKey: apiKey, Model: "llama3.2", BaseURL: srv.URL + "/v1", MaxAttempts: 3}, render,
		[]HelperFunc{{Name: "json", Description: "doc"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestOpenAIProvider_DraftsAndRepairs(t *testing.T) {
	f := &fakeOpenAI{replies: []string{
		"Here you go:\n```json\n{\"template\":\"{{ .Payload.id | json }}\",\"notes\":\"first\"}\n```",
		`{"template":"{\"t\": {{ .payload.id | json }}}","notes":"fixed"}`,
	}}
	d := newOpenAIDrafter(t, f, "")
	res, err := d.DraftTemplate(context.Background(), Request{EventName: "e", Instructions: "x", SamplePayload: map[string]any{"id": "ord_1"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Attempts != 2 || res.Notes != "fixed" || res.Model != "llama3.2" {
		t.Fatalf("unexpected result %+v", res)
	}
	if d.Provider() != "openai" {
		t.Fatalf("provider = %s", d.Provider())
	}
	if len(f.requests) != 2 {
		t.Fatalf("calls = %d", len(f.requests))
	}
	// Request shape: system first, json_schema response_format, no auth header without a key.
	msgs := f.requests[0]["messages"].([]any)
	if msgs[0].(map[string]any)["role"] != "system" || f.requests[0]["model"] != "llama3.2" {
		t.Fatalf("bad first request: %v", f.requests[0])
	}
	rf := f.requests[0]["response_format"].(map[string]any)
	if rf["type"] != "json_schema" {
		t.Fatalf("response_format = %v", rf)
	}
	if f.auth[0] != "" {
		t.Fatalf("no key configured but Authorization sent: %q", f.auth[0])
	}
	// The repair turn carries the assistant's previous reply and the error.
	second, _ := json.Marshal(f.requests[1]["messages"])
	if !strings.Contains(string(second), `"role":"assistant"`) || !strings.Contains(string(second), "does not exist in the data") {
		t.Fatalf("repair turn malformed: %s", second)
	}
}

func TestOpenAIProvider_AuthAndErrors(t *testing.T) {
	f := &fakeOpenAI{replies: []string{`{"template":"{{ .payload.id | json }}","notes":""}`}}
	d := newOpenAIDrafter(t, f, "sk-local")
	if _, err := d.DraftTemplate(context.Background(), Request{EventName: "e", Instructions: "x"}); err != nil {
		t.Fatal(err)
	}
	if f.auth[0] != "Bearer sk-local" {
		t.Fatalf("Authorization = %q", f.auth[0])
	}

	for status, want := range map[int]svcerrors.Status{401: svcerrors.FailedPrecondition, 404: svcerrors.FailedPrecondition, 429: svcerrors.ResourceExhausted, 503: svcerrors.Unavailable} {
		f := &fakeOpenAI{status: status}
		d := newOpenAIDrafter(t, f, "")
		_, err := d.DraftTemplate(context.Background(), Request{EventName: "e", Instructions: "x"})
		if got := svcerrors.Classify(err, "").Status; got != want {
			t.Errorf("HTTP %d: status %v, want %v (err=%v)", status, got, want, err)
		}
	}

	f = &fakeOpenAI{replies: []string{"blocked"}, finish: "content_filter"}
	d = newOpenAIDrafter(t, f, "")
	_, err := d.DraftTemplate(context.Background(), Request{EventName: "e", Instructions: "x"})
	if got := svcerrors.Classify(err, "").Status; got != svcerrors.FailedPrecondition {
		t.Fatalf("content_filter: status %v (err=%v)", got, err)
	}
}

func TestNew_ProviderValidation(t *testing.T) {
	r := func(context.Context, string, string, map[string]any) (string, error) { return "", nil }
	if _, err := New(Config{Provider: ProviderOpenAI, Model: "m"}, r, nil, nil); err == nil || !strings.Contains(err.Error(), "base URL") {
		t.Fatalf("openai without base URL: %v", err)
	}
	if _, err := New(Config{Provider: "gemini", Model: "m", APIKey: "k"}, r, nil, nil); err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("unknown provider: %v", err)
	}
	if d, err := New(Config{Provider: ProviderOpenAI, Model: "m", BaseURL: "http://localhost:11434/v1"}, r, nil, nil); err != nil || d.Provider() != "openai" {
		t.Fatalf("openai without key should work for local servers: %v", err)
	}
}

func TestParseDraft_ToleratesFencesAndPreamble(t *testing.T) {
	for _, raw := range []string{
		`{"template":"a","notes":"b"}`,
		"```json\n{\"template\":\"a\",\"notes\":\"b\"}\n```",
		"Sure! Here is the template:\n\n{\"template\":\"a\",\"notes\":\"b\"}",
	} {
		dr, err := parseDraft(raw)
		if err != nil || dr.Template != "a" || dr.Notes != "b" {
			t.Errorf("%q: %+v %v", raw, dr, err)
		}
	}
	if _, err := parseDraft("no json here"); err == nil {
		t.Error("expected error for non-JSON")
	}
	if _, err := parseDraft(`{"template":"","notes":"x"}`); err == nil {
		t.Error("expected error for empty template")
	}
}
