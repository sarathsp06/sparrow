//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildCLI compiles satellites/sparrow into a temp dir and returns the binary path.
func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "sparrow")
	cmd := exec.Command("go", "build", "-o", bin, "github.com/sarathsp06/sparrow/satellites/sparrow")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "go build satellites/sparrow: %s", out)
	return bin
}

// runCLI executes the built binary against env's server and returns stdout.
func runCLI(t *testing.T, env *testEnv, bin, namespace string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = append(os.Environ(),
		"SPARROW_URL="+env.baseURL,
		"SPARROW_NAMESPACE="+namespace,
		"SPARROW_CONFIG="+filepath.Join(t.TempDir(), "no-config.yaml"),
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	require.NoError(t, cmd.Run(), "sparrow %s\nstdout: %s\nstderr: %s", strings.Join(args, " "), stdout.String(), stderr.String())
	return stdout.String()
}

// TestCLI_E2E covers push (with event-type auto-create), use (recipe apply
// with transform), and tail --once against a real server stack.
func TestCLI_E2E(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()
	bin := buildCLI(t)

	const (
		namespace = "cli-e2e"
		eventName = "cli.order.created"
	)

	// ── 1. push: auto-creates the event type and prints the event id ─────
	out := runCLI(t, env, bin, namespace, "push", eventName, "-d", `{"order_id":"ord_1","amount":42}`, "-l", "env=test")
	lines := strings.Fields(strings.TrimSpace(out))
	eventID := lines[len(lines)-1] // last token: the id (first line may note auto-creation)
	require.NotEmpty(t, eventID)

	var eventOut struct {
		EventID string `json:"event_id"`
		Event   string `json:"event"`
	}
	resp, err := c.get(ctx, "/v1/events/"+eventID, &eventOut)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "pushed event should be retrievable")
	assert.Equal(t, eventName, eventOut.Event)

	// ── 2. use: apply the fixture recipe against a local receiver ────────
	var mu sync.Mutex
	var received []byte
	gotDelivery := make(chan struct{})
	var once sync.Once
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// The ord_1 event pushed before `use` may fan out late on a slow
		// runner and land here too — only the ord_2 delivery counts.
		if bytes.Contains(body, []byte("ord_2")) {
			mu.Lock()
			received = body
			mu.Unlock()
			once.Do(func() { close(gotDelivery) })
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	recipePath, err := filepath.Abs(filepath.Join("..", "..", "satellites", "sparrow", "testdata", "echo.yaml"))
	require.NoError(t, err)
	out = runCLI(t, env, bin, namespace, "use", "echo",
		"--file", recipePath,
		"--event", eventName,
		"--param", "webhook_url="+receiver.URL,
	)
	assert.Contains(t, out, "webhook ")
	assert.Contains(t, out, "applied")

	// Push another event; the receiver must get the TRANSFORMED body.
	out = runCLI(t, env, bin, namespace, "push", eventName, "-d", `{"order_id":"ord_2","amount":7}`)
	secondEventID := strings.TrimSpace(out)

	select {
	case <-gotDelivery:
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for recipe delivery")
	}
	mu.Lock()
	body := received
	mu.Unlock()

	var transformed struct {
		Kind  string         `json:"kind"`
		Event string         `json:"event"`
		Data  map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &transformed), "receiver body should be the rendered template, got: %s", body)
	assert.Equal(t, "echo", transformed.Kind, "body must be destination-shaped, not the envelope")
	assert.Equal(t, eventName, transformed.Event)
	assert.Equal(t, "ord_2", transformed.Data["order_id"])
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(body, &envelope))
	assert.NotContains(t, envelope, "event_id", "envelope fields must not leak into transformed body")

	// ── 3. tail deliveries --once prints the delivery row ────────────────
	require.Eventually(t, func() bool {
		out = runCLI(t, env, bin, namespace, "tail", "deliveries", "--once")
		return strings.Contains(out, secondEventID) && strings.Contains(out, "success")
	}, 30*time.Second, time.Second, "tail --once should print the successful delivery, last output:\n%s", out)
	assert.Contains(t, out, receiver.URL, "tail should resolve webhook_id to the destination URL")
}
