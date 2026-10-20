package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sarathsp06/sparrow/pkg/receivercode"
)

// scaffoldServer fakes generateReceiverCode and sendTestEvent.
type scaffoldServer struct {
	mu       sync.Mutex
	bodies   map[string]string
	tests    []string
	limitOne bool // answer the first test event of each run with 429
	limited  bool
	status   int // receiver status reported by sendTestEvent
}

func startScaffoldServer(t *testing.T) *scaffoldServer {
	t.Helper()
	s := &scaffoldServer{bodies: map[string]string{}, status: 200}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		defer s.mu.Unlock()
		s.bodies[r.URL.Path] = string(raw)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, ":generateReceiverCode"):
			_, _ = w.Write([]byte(`{"files":[{"path":"main.go","content":"package main\n","mode":420},{"path":"scripts/run.sh","content":"#!/bin/sh\n","mode":493}],
				"spec":{"subscriptions":[{"id":"sub_1","event_type":"order.paid","notes":["best effort"]},{"id":"sub_2","event_type":"order.note"}]},
				"run":"go run .","test":"go test ./..."}`))
		case strings.HasSuffix(r.URL.Path, ":sendTestEvent"):
			if s.limitOne && !s.limited {
				s.limited = true
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"detail":"one test event per webhook every 2s"}`))
				return
			}
			s.tests = append(s.tests, string(raw))
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": s.status, "latency_ms": 3, "response_excerpt": "nope", "error_class": map[bool]string{true: "", false: "client_error"}[s.status < 300]})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	pushEnv(t, srv)
	return s
}

func TestScaffoldWritesTheProject(t *testing.T) {
	s := startScaffoldServer(t)
	dir := filepath.Join(t.TempDir(), "rcv")

	out, err := runCLICmd(t, "", "scaffold", "--webhook", "wh_1", "--lang", "go", "--framework", "nethttp", "-o", dir, "--subscription", "sub_1")
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(s.bodies["/v1/consumers/acme/webhooks/wh_1:generateReceiverCode"]), &body)
	if body["language"] != "go" || body["framework"] != "nethttp" || len(body["subscription_ids"].([]any)) != 1 {
		t.Errorf("request body = %v", body)
	}
	if !strings.Contains(out, "Wrote 2 files") || !strings.Contains(out, "go test ./...") || !strings.Contains(out, "note: order.paid: best effort") {
		t.Errorf("output:\n%s", out)
	}
	info, err := os.Stat(filepath.Join(dir, "scripts", "run.sh"))
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("run.sh: %v %v", info, err)
	}

	// A second run refuses to overwrite, and --force does.
	if _, err := runCLICmd(t, "", "scaffold", "--webhook", "wh_1", "--lang", "go", "--framework", "nethttp", "-o", dir); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("overwrite without --force: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("edited"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCLICmd(t, "", "scaffold", "--webhook", "wh_1", "--lang", "go", "--framework", "nethttp", "-o", dir, "--force"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "main.go")); string(b) != "package main\n" {
		t.Errorf("--force kept %q", b)
	}
}

func TestScaffoldSendsTestEvents(t *testing.T) {
	testEventRetryWait = time.Millisecond
	s := startScaffoldServer(t)
	s.limitOne = true
	out, err := runCLICmd(t, "", "scaffold", "--webhook", "wh_1", "--lang", "python", "--framework", "fastapi", "-o", t.TempDir(), "--test")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(s.tests) != 2 || !strings.Contains(s.tests[0], "sub_1") || !strings.Contains(s.tests[1], "sub_2") {
		t.Errorf("test events = %v (a 429 must be retried)", s.tests)
	}
	if !strings.Contains(out, "✓ order.paid: 200 in 3 ms") {
		t.Errorf("output:\n%s", out)
	}

	s.status = 401
	out, err = runCLICmd(t, "", "scaffold", "--webhook", "wh_1", "--lang", "python", "--framework", "fastapi", "-o", t.TempDir(), "--test")
	if err == nil || !strings.Contains(out, "✗ order.paid: 401 client_error") {
		t.Errorf("failing receiver: err %v\n%s", err, out)
	}
}

func TestScaffoldOffline(t *testing.T) {
	t.Setenv("SPARROW_URL", "http://127.0.0.1:1") // never called
	dir := t.TempDir()
	schema := filepath.Join(dir, "schema.json")
	if err := os.WriteFile(schema, []byte(`{"type":"object","required":["id"],"properties":{"id":{"type":"string"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "rcv")
	out, err := runCLICmd(t, "", "scaffold", "--offline", "--schema", schema, "--event-type", "order.paid", "--lang", "node", "--framework", "express", "-o", outDir)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	models, err := os.ReadFile(filepath.Join(outDir, "src", "models.ts"))
	if err != nil || !strings.Contains(string(models), "export interface OrderPaidPayload") {
		t.Fatalf("models.ts: %v\n%s", err, models)
	}
	if _, err := os.Stat(filepath.Join(outDir, "src", "sparrow-verify.ts")); err != nil {
		t.Fatal("vendored helper missing")
	}
	if !strings.Contains(out, "npm start") {
		t.Errorf("output:\n%s", out)
	}
}

func TestScaffoldRejectsBadFlags(t *testing.T) {
	t.Setenv("SPARROW_CONFIG", filepath.Join(t.TempDir(), "missing.yaml"))
	bad := filepath.Join(t.TempDir(), "bad.json")
	_ = os.WriteFile(bad, []byte("{"), 0o600)
	cases := map[string][]string{
		"no lang":           {"scaffold", "--webhook", "wh"},
		"no webhook":        {"scaffold", "--lang", "go", "--framework", "nethttp"},
		"offline + test":    {"scaffold", "--offline", "--test", "--lang", "go", "--framework", "nethttp"},
		"offline no input":  {"scaffold", "--offline", "--lang", "go", "--framework", "nethttp"},
		"offline bad json":  {"scaffold", "--offline", "--sample", bad, "--lang", "go", "--framework", "nethttp"},
		"offline bad lang":  {"scaffold", "--offline", "--sample", bad + "x", "--lang", "go", "--framework", "nethttp"},
		"offline ed25519":   {"scaffold", "--offline", "--schema", bad, "--signing", "ed25519", "--lang", "go", "--framework", "nethttp"},
		"unknown framework": {"scaffold", "--offline", "--transform", bad, "--lang", "go", "--framework", "gin", "-o", t.TempDir()},
	}
	for name, args := range cases {
		if _, err := runCLICmd(t, "", args...); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestWriteProjectRefusesEscapingPaths(t *testing.T) {
	for _, p := range []string{"../evil", "/etc/passwd", "a/../../b"} {
		if err := writeProject(t.TempDir(), []receivercode.File{{Path: p, Content: "x"}}, true); err == nil {
			t.Errorf("%s: written", p)
		}
	}
}

func TestMCPGenerateReceiverCode(t *testing.T) {
	s := startScaffoldServer(t)
	srv, _ := newMCPServer(newAPIClient(config{ServerURL: os.Getenv("SPARROW_URL")}), "acme", false)
	out := call(t, connectMCP(t, srv), "generate_receiver_code", map[string]any{"webhook_id": "wh_1", "language": "go", "framework": "nethttp"})
	if !strings.Contains(out, `"path":"main.go"`) || !strings.Contains(out, `"run":"go run ."`) {
		t.Errorf("tool output = %s", out)
	}
	if !strings.Contains(s.bodies["/v1/consumers/acme/webhooks/wh_1:generateReceiverCode"], `"framework":"nethttp"`) {
		t.Errorf("request = %v", s.bodies)
	}
	_ = mcp.Tool{}
}
