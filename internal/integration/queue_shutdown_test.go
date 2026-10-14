//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestQueueStopCancelsAfterSoftStop checks the shutdown budget is a hard
// bound: a delivery to a receiver slower than the budget is cancelled once
// the soft stop runs out, Stop returns in time, and the delivery is left for
// a retry instead of being failed.
func TestQueueStopCancelsAfterSoftStop(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-release:
		case <-r.Context().Done():
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(slow.Close)
	t.Cleanup(func() { close(release) }) // runs first: lets slow.Close return

	const (
		consumer  = "queue-shutdown"
		eventName = "queue.shutdown"
	)
	registerEventType(t, c, ctx, eventName)
	registerWebhookPipeline(t, c, ctx, consumer, eventName, slow.URL, 3)
	eventID := pushTestEvent(t, c, ctx, consumer, eventName)

	select {
	case <-started:
	case <-time.After(30 * time.Second):
		t.Fatal("the delivery never reached the receiver")
	}

	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	begin := time.Now()
	err := env.queueMgr.Stop(stopCtx)
	elapsed := time.Since(begin)
	require.Less(t, elapsed, 4*time.Second, "Stop overran its 2s budget")
	require.NoError(t, err, "the hard stop should have finished within the budget")

	var status string
	require.NoError(t, env.sqlxDB.GetContext(ctx, &status,
		`SELECT status::text FROM webhook_deliveries WHERE event_id = $1`, eventID))
	require.Contains(t, []string{"pending", "retrying"}, status, "a cancelled attempt must not fail the delivery")
}
