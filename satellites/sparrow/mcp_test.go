package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const fakeSecret = "whsec_MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3" // base64 of 0123456789abcdef01234567

// fakeSparrow is a minimal stand-in for the REST API: it records every
// request (method + path) and answers with canned JSON.
type fakeSparrow struct {
	*httptest.Server
	mu     sync.Mutex
	last   *http.Request
	seen   []string
	bodies map[string]string // "METHOD path" -> last request body
}

func newFakeSparrow(t *testing.T) *fakeSparrow {
	t.Helper()
	f := &fakeSparrow{bodies: map[string]string{}}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.last = r
		f.seen = append(f.seen, r.Method+" "+r.URL.Path)
		f.bodies[r.Method+" "+r.URL.Path] = string(body)
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/attempts"):
			_, _ = w.Write([]byte(`{"items":[{"success":false,"response_code":502,"error_category":"server_error"}]}`))
		case strings.HasSuffix(r.URL.Path, ":retry"):
			_, _ = w.Write([]byte(`{"count":1,"delivery_ids":["d1"]}`))
		case strings.HasSuffix(r.URL.Path, ":testTemplate"):
			_, _ = w.Write([]byte(`{"output":"{\"ok\":true}"}`))
		case strings.HasSuffix(r.URL.Path, "/events"):
			_, _ = w.Write([]byte(`{"event_id":"evt_1","duplicate":false}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/webhooks"):
			_, _ = w.Write([]byte(`{"webhook_id":"wh_1","url":"x","http_config":{"webhook_secret":"` + fakeSecret + `"}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/subscriptions"):
			_, _ = w.Write([]byte(`{"subscription_id":"sub_1"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/subscriptions"):
			_, _ = w.Write([]byte(`{"items":[{"subscription_id":"sub_1","event_name":"order.created"}]}`))
		case r.Method == http.MethodPatch:
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			_, _ = w.Write([]byte(`{"items":[]}`))
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeSparrow) lastReq() *http.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.last
}

func (f *fakeSparrow) body(method, path string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.bodies[method+" "+path]
}

func (f *fakeSparrow) sawPath(method, suffix string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.seen {
		if strings.HasPrefix(s, method+" ") && strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

func connectMCP(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func toolNames(t *testing.T, cs *mcp.ClientSession) []string {
	t.Helper()
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(res.Tools))
	for _, tl := range res.Tools {
		names = append(names, tl.Name)
	}
	sort.Strings(names)
	return names
}

// call invokes a tool and returns its structured result as JSON text.
func call(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("%s: tool error: %v", name, res.Content)
	}
	b, _ := json.Marshal(res.StructuredContent)
	return string(b)
}

func TestMCPReadOnlyHidesWriteTools(t *testing.T) {
	client := newAPIClient(config{ServerURL: newFakeSparrow(t).URL})

	readOnly := []string{"export_event_types", "generate_receiver_code", "get_delivery_attempts", "get_event_type", "get_recipe", "get_stats", "get_webhook", "list_deliveries", "list_event_types", "list_events", "list_recipes", "list_subscriptions", "list_webhooks", "test_template", "validate_recipe"}
	srv, _ := newMCPServer(client, "acme", false)
	if got := toolNames(t, connectMCP(t, srv)); strings.Join(got, ",") != strings.Join(readOnly, ",") {
		t.Fatalf("read-only tools = %v, want %v", got, readOnly)
	}

	want := append(readOnly, "apply_recipe", "create_subscription", "delete_subscription", "delete_webhook", "list_received", "push_event", "register_webhook", "retry_delivery", "start_listen", "stop_listen", "update_subscription", "import_event_types", "update_event_type")
	sort.Strings(want)
	srv, _ = newMCPServer(client, "acme", true)
	if got := toolNames(t, connectMCP(t, srv)); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("allow-write tools = %v, want %v", got, want)
	}
}

func TestMCPToolsCallTheAPI(t *testing.T) {
	api := newFakeSparrow(t)
	client := newAPIClient(config{ServerURL: api.URL, APIKey: "k"})
	srv, _ := newMCPServer(client, "acme", true)
	cs := connectMCP(t, srv)

	call(t, cs, "list_deliveries", map[string]any{"status": "failed", "limit": 5, "webhook_id": "wh_1", "event_name": "order.created", "cursor": "c2"})
	last := api.lastReq()
	q := last.URL.Query()
	if last.URL.Path != "/v1/consumers/acme/deliveries" || q.Get("status") != "failed" || q.Get("limit") != "5" ||
		q.Get("webhook_id") != "wh_1" || q.Get("event_name") != "order.created" || q.Get("cursor") != "c2" || q.Has("offset") {
		t.Fatalf("list_deliveries hit %s", last.URL)
	}
	if last.Header.Get("X-API-Key") != "k" {
		t.Fatal("api key header missing")
	}

	out := call(t, cs, "get_delivery_attempts", map[string]any{"delivery_id": "d1"})
	if last = api.lastReq(); last.URL.Path != "/v1/consumers/acme/deliveries/d1/attempts" || !strings.Contains(out, "server_error") {
		t.Fatalf("attempts: path %s out %s", last.URL.Path, out)
	}

	out = call(t, cs, "test_template", map[string]any{"event_name": "order.created", "template": "{}"})
	if last = api.lastReq(); last.URL.Path != "/v1/subscriptions:testTemplate" || !strings.Contains(out, "output") {
		t.Fatalf("test_template: path %s out %s", last.URL.Path, out)
	}
	if b := api.body(http.MethodPost, "/v1/subscriptions:testTemplate"); strings.Contains(b, "payload") {
		t.Fatalf("test_template without payload must not send one: %s", b)
	}
	call(t, cs, "test_template", map[string]any{"event_name": "order.created", "template": "{}", "payload": map[string]any{"id": "ord_7"}})
	if b := api.body(http.MethodPost, "/v1/subscriptions:testTemplate"); !strings.Contains(b, `"payload":{"id":"ord_7"}`) {
		t.Fatalf("test_template payload not sent: %s", b)
	}

	out = call(t, cs, "push_event", map[string]any{"event": "order.created", "payload": map[string]any{"id": 1}})
	if last = api.lastReq(); last.Method != http.MethodPost || last.URL.Query().Get("event") != "order.created" || !strings.Contains(out, "evt_1") {
		t.Fatalf("push_event: %s %s -> %s", last.Method, last.URL, out)
	}

	out = call(t, cs, "retry_delivery", map[string]any{"delivery_id": "d1"})
	if last = api.lastReq(); last.URL.Path != "/v1/consumers/acme/deliveries/d1:retry" || !strings.Contains(out, `"count":1`) {
		t.Fatalf("retry_delivery: %s -> %s", last.URL.Path, out)
	}

	out = call(t, cs, "register_webhook", map[string]any{"url": "https://example.com/hook", "events": []string{"order.created"}})
	if last = api.lastReq(); last.URL.Path != "/v1/consumers/acme/webhooks" || !strings.Contains(out, "wh_1") {
		t.Fatalf("register_webhook: %s -> %s", last.URL.Path, out)
	}

	out = call(t, cs, "create_subscription", map[string]any{"webhook_id": "wh_1", "event_name": "order.paid", "transform_template": "{}"})
	if last = api.lastReq(); last.URL.Path != "/v1/consumers/acme/subscriptions" || !strings.Contains(out, "sub_1") {
		t.Fatalf("create_subscription: %s -> %s", last.URL.Path, out)
	}

	call(t, cs, "delete_webhook", map[string]any{"webhook_id": "wh_1"})
	if last = api.lastReq(); last.Method != http.MethodDelete || last.URL.Path != "/v1/consumers/acme/webhooks/wh_1" {
		t.Fatalf("delete_webhook: %s %s", last.Method, last.URL.Path)
	}
}

func TestMCPPushAutoCreatesEventType(t *testing.T) {
	var calls []string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case strings.HasSuffix(r.URL.Path, "/events") && len(calls) == 1:
			http.Error(w, `{"detail":"unknown event type"}`, http.StatusNotFound)
		case strings.HasSuffix(r.URL.Path, "/events"):
			_, _ = w.Write([]byte(`{"event_id":"evt_2"}`))
		default:
			w.WriteHeader(http.StatusCreated)
		}
	}))
	t.Cleanup(api.Close)
	srv, _ := newMCPServer(newAPIClient(config{ServerURL: api.URL}), "acme", true)
	out := call(t, connectMCP(t, srv), "push_event", map[string]any{"event": "new.type", "payload": map[string]any{}})
	if !strings.Contains(out, `"event_type_created":true`) || !strings.Contains(out, "evt_2") {
		t.Fatalf("push_event = %s", out)
	}
	if want := []string{"POST /v1/consumers/acme/events", "POST /v1/event-types", "POST /v1/consumers/acme/events"}; strings.Join(calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls = %v", calls)
	}
}

func TestMCPListenRoundTrip(t *testing.T) {
	api := newFakeSparrow(t)
	srv, listener := newMCPServer(newAPIClient(config{ServerURL: api.URL}), "acme", true)
	cs := connectMCP(t, srv)

	out := call(t, cs, "start_listen", map[string]any{"events": []string{"order.created"}, "public_url": "http://example.test/hook"})
	var st listenStatus
	if err := json.Unmarshal([]byte(out), &st); err != nil || !st.Running || st.WebhookID != "wh_1" {
		t.Fatalf("start_listen = %s (%v)", out, err)
	}
	if !api.sawPath(http.MethodPost, "/webhooks") {
		t.Fatal("start_listen did not register a webhook")
	}
	if _, err := listener.start(context.Background(), startListenIn{Events: []string{"x"}}); err != errListenerRunning {
		t.Fatalf("second start_listen err = %v, want %v", err, errListenerRunning)
	}

	target := "http://127.0.0.1" + st.LocalAddr[strings.LastIndex(st.LocalAddr, ":"):]
	body := []byte(`{"event_name":"order.created","event_id":"evt_9","payload":{"n":1}}`)

	forged, _ := http.Post(target, "application/json", bytes.NewReader(body)) //nolint:noctx
	if forged.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unsigned delivery got %d, want 401", forged.StatusCode)
	}
	req := signedRequest(t, fakeSecret, body)
	req.RequestURI, req.URL.Scheme, req.URL.Host = "", "http", strings.TrimPrefix(target, "http://")
	signed, err := http.DefaultClient.Do(req)
	if err != nil || signed.StatusCode != http.StatusOK {
		t.Fatalf("signed delivery: %v %v", err, signed)
	}

	out = call(t, cs, "list_received", map[string]any{})
	if !strings.Contains(out, `"event_id":"evt_9"`) || strings.Contains(out, "forged") || !strings.Contains(out, "Webhook-Signature") {
		t.Fatalf("list_received = %s", out)
	}

	out = call(t, cs, "stop_listen", map[string]any{})
	if !strings.Contains(out, `"running":false`) || !api.sawPath(http.MethodDelete, "/webhooks/wh_1") {
		t.Fatalf("stop_listen = %s, delete seen = %v", out, api.sawPath(http.MethodDelete, "/webhooks/wh_1"))
	}
	if _, err := listener.stop(context.Background()); err != nil {
		t.Fatalf("stop twice should be a no-op: %v", err)
	}
	if _, err := http.Get(target); err == nil { //nolint:noctx
		t.Fatal("receiver still accepting connections after stop_listen")
	}
}

func TestMCPToolErrorIsNotProtocolError(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"detail":"nope"}`, http.StatusNotFound)
	}))
	t.Cleanup(api.Close)
	srv, _ := newMCPServer(newAPIClient(config{ServerURL: api.URL}), "acme", false)
	cs := connectMCP(t, srv)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_stats", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected a tool error for a 404")
	}
}

func TestMCPEventTypeTools(t *testing.T) {
	api := newFakeSparrow(t)
	client := newAPIClient(config{ServerURL: api.URL})
	srv, _ := newMCPServer(client, "acme", true)
	cs := connectMCP(t, srv)

	call(t, cs, "get_event_type", map[string]any{"name": "order.created", "versions": true})
	if !api.sawPath(http.MethodGet, "/v1/event-types/order.created") || !api.sawPath(http.MethodGet, "/v1/event-types/order.created/versions") {
		t.Fatal("get_event_type with versions should read the type and its versions")
	}

	call(t, cs, "list_events", map[string]any{"event": "order.created", "labels": map[string]string{"tier": "gold", "env": "prod"}})
	last := api.lastReq()
	q := last.URL.Query()
	if last.URL.Path != "/v1/consumers/acme/events" || q.Get("event") != "order.created" || q.Get("limit") != "20" ||
		q.Get("labels") != "env=prod,tier=gold" || q.Has("prepare_repush") {
		t.Fatalf("list_events hit %s", last.URL)
	}

	call(t, cs, "export_event_types", map[string]any{"prefix": "order."})
	if b := api.body(http.MethodPost, "/v1/event-types:export"); b != `{"prefix":"order."}` {
		t.Fatalf("export body = %s", b)
	}

	bundle := map[string]any{"apiVersion": "sparrow/v1", "items": []any{map[string]any{"name": "order.created"}}}
	call(t, cs, "import_event_types", map[string]any{"bundle": bundle})
	var imp map[string]any
	_ = json.Unmarshal([]byte(api.body(http.MethodPost, "/v1/event-types:import")), &imp)
	if imp["dry_run"] != true || imp["apiVersion"] != "sparrow/v1" || imp["items"] == nil {
		t.Fatalf("import must default to a dry run and send the bundle as-is: %v", imp)
	}
	call(t, cs, "import_event_types", map[string]any{"bundle": bundle, "dry_run": false, "accept_edited": true, "pause_affected": true})
	_ = json.Unmarshal([]byte(api.body(http.MethodPost, "/v1/event-types:import")), &imp)
	if ack, _ := imp["acknowledge"].([]any); imp["dry_run"] != false || len(ack) != 1 || ack[0] != "items_changed" || imp["subscription_policy"] != "pause" {
		t.Fatalf("import options not sent: %v", imp)
	}

	schema := map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}}
	call(t, cs, "update_event_type", map[string]any{"name": "order.created", "event_schema": schema, "allow_breaking": true})
	last = api.lastReq()
	var upd map[string]any
	_ = json.Unmarshal([]byte(api.body(http.MethodPatch, "/v1/event-types/order.created")), &upd)
	if last.Method != http.MethodPatch || last.URL.Query().Get("allow_breaking") != "true" || upd["event_schema"] == nil || len(upd) != 1 {
		t.Fatalf("update_event_type: %s %s body %v", last.Method, last.URL, upd)
	}

	for name, args := range map[string]map[string]any{
		"update_event_type":  {"name": "order.created"},
		"export_event_types": {"names": []string{"a"}, "all": true},
		"import_event_types": {"bundle": map[string]any{"apiVersion": "x"}},
	} {
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil || !res.IsError {
			t.Errorf("%s(%v) should be a tool error, got err=%v res=%v", name, args, err, res)
		}
	}
}
