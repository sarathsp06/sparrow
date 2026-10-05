//go:build integration

package integration

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// syncBuffer is a bytes.Buffer safe to read while a process writes to it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestListenSession_CLIReceivesAndForwards runs `sparrow listen --forward`
// against the server: nothing listens on a port, the CLI polls for its
// deliveries, forwards the verified ones and reports the app's response as
// the attempt's result. Ctrl-C deletes the session.
func TestListenSession_CLIReceivesAndForwards(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()
	bin := buildCLI(t)
	const consumer, eventName = "listen-e2e", "listen.order.created"
	registerEventType(t, c, ctx, eventName)

	var mu sync.Mutex
	var bodies []string
	var signed []bool
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(body))
		signed = append(signed, r.Header.Get("webhook-signature") != "")
		mu.Unlock()
		if bytes.Contains(body, []byte("fail")) {
			http.Error(w, "app broke", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer app.Close()

	var out syncBuffer
	cmd := exec.Command(bin, "listen", "--event", eventName, "--forward", app.URL)
	cmd.Env = append(os.Environ(),
		"SPARROW_URL="+env.baseURL,
		"SPARROW_CONSUMER="+consumer,
		"SPARROW_CONFIG="+filepath.Join(t.TempDir(), "no-config.yaml"),
	)
	cmd.Stdout, cmd.Stderr = &out, &out
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	sessionRe := regexp.MustCompile(`listen session ([0-9a-f-]{36}) started`)
	var sessionID string
	require.Eventually(t, func() bool {
		m := sessionRe.FindStringSubmatch(out.String())
		if m != nil {
			sessionID = m[1]
		}
		return m != nil
	}, 30*time.Second, 100*time.Millisecond, "CLI output:\n%s", out.String())
	assert.Contains(t, out.String(), "webhook secret: whsec_")

	// A delivery reaches the local app through the CLI and succeeds with the
	// app's own status.
	ok := pushPayload(t, c, ctx, consumer, eventName, map[string]any{"order_id": "ord_1"})
	d := waitForDelivery(t, c, ctx, consumer, ok, func(d templateDelivery) bool { return d.Status == "success" || d.Status == "failed" })
	require.Equal(t, "success", d.Status, "delivery: %+v\nCLI output:\n%s", d, out.String())
	mu.Lock()
	require.Len(t, bodies, 1)
	assert.Contains(t, bodies[0], "ord_1")
	assert.True(t, signed[0], "the forwarded request keeps Sparrow's signing headers")
	mu.Unlock()
	assert.Contains(t, out.String(), "signature: verified")
	assert.Contains(t, out.String(), "-> 202")

	// The app's 5xx is the attempt's result: retried, like any receiver's.
	bad := pushPayload(t, c, ctx, consumer, eventName, map[string]any{"order_id": "fail"})
	d = waitForDelivery(t, c, ctx, consumer, bad, func(d templateDelivery) bool { return d.Status == "retrying" || d.Status == "failed" })
	assert.Equal(t, "server_error", d.ErrorCategory)
	assert.Contains(t, d.ErrorMessage, "500")

	// A developer's failing app never touches health.
	var health struct {
		Health           string `json:"health"`
		FailedDeliveries int    `json:"failed_deliveries"`
	}
	_, err := c.get(ctx, "/v1/consumers/"+consumer+"/webhooks/"+sessionID+"/health", &health)
	require.NoError(t, err)
	assert.Zero(t, health.FailedDeliveries)

	// Ctrl-C deletes the session and its webhook.
	require.NoError(t, cmd.Process.Signal(os.Interrupt))
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		require.NoError(t, err, "CLI output:\n%s", out.String())
	case <-time.After(30 * time.Second):
		t.Fatalf("CLI did not exit on interrupt; output:\n%s", out.String())
	}
	assert.Contains(t, out.String(), "deleted listen session "+sessionID)
	resp, err := c.get(ctx, "/v1/consumers/"+consumer+"/webhooks/"+sessionID, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestListenSession_OfflineFailsFast: a session nobody polls fails its
// deliveries as connection_refused without waiting for a response.
func TestListenSession_OfflineFailsFast(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()
	const consumer, eventName = "listen-offline", "listen.offline"
	registerEventType(t, c, ctx, eventName)

	var session struct {
		SessionID string `json:"session_id"`
	}
	resp, err := c.post(ctx, "/v1/consumers/"+consumer+"/listen-sessions", map[string]any{"events": []string{eventName}}, &session)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	_, err = env.sqlxDB.ExecContext(ctx, `UPDATE listen_sessions SET last_seen_at = NOW() - INTERVAL '5 minutes' WHERE webhook_id = $1`, session.SessionID)
	require.NoError(t, err)

	start := time.Now()
	ev := pushPayload(t, c, ctx, consumer, eventName, map[string]any{"n": 1})
	d := waitForDelivery(t, c, ctx, consumer, ev, func(d templateDelivery) bool { return d.Status == "retrying" || d.Status == "failed" })
	assert.Equal(t, "connection_refused", d.ErrorCategory)
	assert.Contains(t, d.ErrorMessage, "sparrow listen is not running")
	assert.Less(t, time.Since(start), 8*time.Second, "an offline session must not wait out the request timeout")

	// Answering a delivery nobody claimed is refused.
	resp, err = c.post(ctx, "/v1/consumers/"+consumer+"/listen-sessions/"+session.SessionID+"/deliveries/"+d.DeliveryID+":respond", map[string]any{"status": 200}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Another consumer cannot see the session.
	resp, err = c.post(ctx, "/v1/consumers/someone-else/listen-sessions/"+session.SessionID+":claim?wait_seconds=0", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestListenSession_URLCannotBeRegisteredDirectly: sparrow-cli:// webhooks
// only come from listen sessions.
func TestListenSession_URLCannotBeRegisteredDirectly(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	resp, err := c.post(context.Background(), "/v1/consumers/x/webhooks", map[string]any{
		"url":    "sparrow-cli://00000000-0000-0000-0000-000000000001",
		"events": []string{"a.b"},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
