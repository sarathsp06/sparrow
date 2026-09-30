package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bundleServer fakes the export and import endpoints and records the last
// request body per path.
type bundleServer struct {
	bodies      map[string]map[string]any
	importReply string
}

func startBundleServer(t *testing.T, importReply string) *bundleServer {
	t.Helper()
	bs := &bundleServer{bodies: map[string]map[string]any{}, importReply: importReply}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		bs.bodies[r.URL.Path] = body
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/event-types:export":
			// Field order as the server writes it; the CLI must keep it.
			_, _ = w.Write([]byte(`{"apiVersion":"sparrow/v1","kind":"EventTypeList","stamp":{"sparrow_version":"1.4.0","format":1,"sha256":"abc"},"items":[{"name":"order.created","active":true},{"name":"order.shipped","active":true}]}`))
		case "/v1/event-types:import":
			_, _ = w.Write([]byte(bs.importReply))
		case "/v1/event-types/order.created/versions":
			_, _ = w.Write([]byte(`{"items":[{"version":2,"created_at":"2026-09-30T10:00:00Z","event_schema":{"type":"object"}},{"version":1,"created_at":"2026-09-01T10:00:00Z","event_schema":{"type":"object"},"schema_defined_at":"2026-09-02T10:00:00Z"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	pushEnv(t, srv)
	return bs
}

func runCLICmd(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	root := newRootCmd(&out)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestEventsExport(t *testing.T) {
	bs := startBundleServer(t, "")

	out, err := runCLICmd(t, "", "events", "export", "order.created", "order.shipped")
	if err != nil {
		t.Fatal(err)
	}
	names, _ := bs.bodies["/v1/event-types:export"]["names"].([]any)
	if len(names) != 2 || names[0] != "order.created" {
		t.Errorf("request names = %v", bs.bodies["/v1/event-types:export"])
	}
	if !strings.HasPrefix(out, "{\n  \"apiVersion\": \"sparrow/v1\",\n  \"kind\": \"EventTypeList\",") {
		t.Errorf("bundle should keep the server's field order, indented:\n%s", out)
	}

	path := filepath.Join(t.TempDir(), "events.json")
	out, err = runCLICmd(t, "", "events", "export", "--prefix", "order.", "-f", path)
	if err != nil {
		t.Fatal(err)
	}
	if bs.bodies["/v1/event-types:export"]["prefix"] != "order." {
		t.Errorf("prefix not sent: %v", bs.bodies["/v1/event-types:export"])
	}
	if !strings.Contains(out, "wrote 2 event type(s)") {
		t.Errorf("summary: %s", out)
	}
	file, err := os.ReadFile(path)
	if err != nil || !json.Valid(file) {
		t.Fatalf("file not written as JSON: %v", err)
	}
}

func TestEventsExportNeedsOneSelector(t *testing.T) {
	startBundleServer(t, "")
	for _, args := range [][]string{
		{"events", "export"},
		{"events", "export", "--all", "--prefix", "x"},
		{"events", "export", "a.b", "--all"},
	} {
		if _, err := runCLICmd(t, "", args...); err == nil || !strings.Contains(err.Error(), "exactly one") {
			t.Errorf("%v: expected a selector error, got %v", args, err)
		}
	}
}

func writeBundle(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "events.json")
	data := `{"apiVersion":"sparrow/v1","kind":"EventTypeList","stamp":{"sparrow_version":"1.4.0","format":1,"sha256":"abc"},"items":[{"name":"order.created"}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEventsImport_Applied(t *testing.T) {
	bs := startBundleServer(t, `{"applied":true,"imported_at":"2026-09-30T12:00:00Z","stamp":{"status":"warning","exported_by":"1.4.0","server":"1.5.0","warnings":["version_differs"]},"items":[{"name":"order.created","action":"new_version","version":3,"previous_version":2,"changes":["schema"],"compatibility":{"result":"compatible"},"subscriptions":{"passed":2,"without_transform":1}}]}`)

	out, err := runCLICmd(t, "", "events", "import", "-f", writeBundle(t), "--accept-version-mismatch", "--allow-breaking", "--pause-affected")
	if err != nil {
		t.Fatalf("applied import should succeed: %v\n%s", err, out)
	}
	body := bs.bodies["/v1/event-types:import"]
	if body["apiVersion"] != "sparrow/v1" || body["stamp"] == nil || body["items"] == nil {
		t.Errorf("the file must be sent as-is: %v", body)
	}
	ack, _ := body["acknowledge"].([]any)
	if len(ack) != 2 || ack[0] != "version_differs" || ack[1] != "format_unsupported" {
		t.Errorf("acknowledge = %v", body["acknowledge"])
	}
	if body["allow_breaking"] != true || body["subscription_policy"] != "pause" || body["dry_run"] != false {
		t.Errorf("options not sent: %v", body)
	}
	for _, want := range []string{"exported by 1.4.0, this server is 1.5.0", "order.created", "new_version", "v2→v3", "1 subscription(s) without a transform", "imported 1 event type(s)"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestEventsImport_BlockedExitsNonZero(t *testing.T) {
	startBundleServer(t, `{"applied":false,"stamp":{"status":"valid","exported_by":"1.4.0","server":"1.4.0"},"blocked_by":["breaking"],"items":[{"name":"order.created","action":"new_version","version":3,"previous_version":2,"blocked":true,"compatibility":{"result":"breaking","reasons":["total: removed required property"]},"subscriptions":{"failures":[{"subscription_id":"s-1","consumer":"shop","payload":"sample","error":"map has no entry for key \"total\""}],"passed":0}}]}`)

	out, err := runCLICmd(t, "", "events", "import", "-f", writeBundle(t))
	if err == nil || !strings.Contains(err.Error(), "nothing was imported") {
		t.Fatalf("expected a blocked error, got %v", err)
	}
	for _, want := range []string{"blocked", "breaking: total: removed required property", "template fails (sample payload): subscription s-1 in shop", "--allow-breaking"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestEventsImport_DryRun(t *testing.T) {
	bs := startBundleServer(t, `{"applied":false,"dry_run":true,"stamp":{"status":"unsigned","server":"1.4.0"},"items":[{"name":"order.created","action":"created","version":1}]}`)

	out, err := runCLICmd(t, "", "events", "import", "-f", writeBundle(t), "--dry-run")
	if err != nil {
		t.Fatalf("an unblocked dry run succeeds: %v", err)
	}
	if bs.bodies["/v1/event-types:import"]["dry_run"] != true {
		t.Error("dry_run not sent")
	}
	for _, want := range []string{"no stamp (hand-written bundle)", "dry run: nothing was written"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestEventsImport_Stdin(t *testing.T) {
	bs := startBundleServer(t, `{"applied":true,"stamp":{"status":"unsigned"},"items":[]}`)
	if _, err := runCLICmd(t, `{"items":[{"name":"a.b"}]}`, "events", "import", "-f", "-"); err != nil {
		t.Fatal(err)
	}
	if bs.bodies["/v1/event-types:import"]["items"] == nil {
		t.Error("stdin bundle not sent")
	}
	if _, err := runCLICmd(t, `{"nope":1}`, "events", "import", "-f", "-"); err == nil || !strings.Contains(err.Error(), "no items") {
		t.Errorf("expected a missing-items error, got %v", err)
	}
}

func TestEventsVersions(t *testing.T) {
	startBundleServer(t, "")
	out, err := runCLICmd(t, "", "events", "versions", "order.created")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "v2") || !strings.Contains(out, "schema added 2026-09-02T10:00:00Z") {
		t.Errorf("versions output:\n%s", out)
	}
	if strings.Index(out, "v2") > strings.Index(out, "v1 ") {
		t.Errorf("newest first:\n%s", out)
	}
}
