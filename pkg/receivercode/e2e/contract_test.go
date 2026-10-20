// Package e2e is the behavioural contract of generated receivers: it replays
// a project's test vectors against the receiver running as a real server
// and checks status codes and handler calls (read from the test-mode
// output). contract.sh generates, builds, starts and checks one target; CI
// runs it for every target (.github/workflows/receivercode-e2e.yml).
package e2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sarathsp06/sparrow/pkg/receivercode"
)

var (
	targetFlag = flag.String("target", "", "target id the receiver was generated for")
	urlFlag    = flag.String("url", "", "base URL of the running receiver")
	logFlag    = flag.String("log", "", "file the receiver's stdout goes to")
	specFlag   = flag.String("spec", "", "the project's testdata/spec.json")
)

// handlerCall is one test-mode line.
type handlerCall struct {
	Handler        string `json:"handler"`
	SubscriptionID string `json:"subscription_id"`
	EventType      string `json:"event_type"`
	WebhookID      string `json:"webhook_id"`
}

func TestContract(t *testing.T) {
	if *urlFlag == "" || *logFlag == "" || *specFlag == "" {
		t.Skip("needs -url, -log and -spec; run contract.sh")
	}
	raw, err := os.ReadFile(*specFlag)
	if err != nil {
		t.Fatal(err)
	}
	var spec receivercode.Spec
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatal(err)
	}
	if *targetFlag != "" && spec.Target != *targetFlag {
		t.Fatalf("spec is for %s, not %s", spec.Target, *targetFlag)
	}
	base := strings.TrimSuffix(*urlFlag, "/")
	waitHealthy(t, base+spec.Receiver.HealthPath)

	client := &http.Client{Timeout: 10 * time.Second}
	for _, v := range spec.Vectors {
		before := len(calls(t))
		headers, body := spec.Test.Rebuild(v, time.Now().Unix())
		req, err := http.NewRequest(http.MethodPost, base+spec.Receiver.WebhookPath, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		for k, val := range headers {
			req.Header.Set(k, val)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s/%s: %v", v.SubscriptionID, v.Name, err)
		}
		reply, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		name := v.SubscriptionID + "/" + v.Name
		if resp.StatusCode != v.Expect.Status {
			t.Errorf("%s: status %d, want %d (%s)", name, resp.StatusCode, v.Expect.Status, reply)
			continue
		}

		got := newCalls(t, before, v.Expect.Handler != "")
		if v.Expect.Handler == "" {
			if len(got) != 0 {
				t.Errorf("%s: handler called, want none: %+v", name, got)
			}
			continue
		}
		if len(got) != 1 {
			t.Errorf("%s: %d handler calls, want 1", name, len(got))
			continue
		}
		call := got[0]
		if call.Handler != v.Expect.Handler || call.WebhookID != v.MessageID {
			t.Errorf("%s: call %+v, want handler %s for %s", name, call, v.Expect.Handler, v.MessageID)
		}
	}
}

func waitHealthy(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for {
		resp, err := http.Get(url) //nolint:gosec // the URL comes from the test's own flags
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("receiver not healthy at %s: %v", url, err)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// newCalls returns the handler calls logged after the first before. A
// receiver writes the line before it answers, but give a slow file system
// a moment when one is expected.
func newCalls(t *testing.T, before int, expectOne bool) []handlerCall {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		all := calls(t)
		if !expectOne || len(all) > before || time.Now().After(deadline) {
			return all[before:]
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// calls reads every test-mode line in the log; other output is ignored.
func calls(t *testing.T) []handlerCall {
	t.Helper()
	f, err := os.Open(*logFlag)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var out []handlerCall
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		var c handlerCall
		if len(line) == 0 || line[0] != '{' || json.Unmarshal(line, &c) != nil || c.Handler == "" {
			continue
		}
		out = append(out, c)
	}
	return out
}
