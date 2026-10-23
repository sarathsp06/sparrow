//go:build integration

package integration

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/server"
)

// TestTenantFairness runs a capped tenant with a slow receiver next to an
// uncapped one through the public package: the capped tenant never has more
// than its cap in flight, its jobs run on its own queue, the uncapped
// tenant's event is delivered while the capped tenant still has a backlog,
// and waiting at the cap uses no delivery attempt.
func TestTenantFairness(t *testing.T) {
	ctx := context.Background()
	slowTenant := uuid.New()
	sp := startEmbeddedSparrow(t, server.WithTenantLimits(
		map[string]int{"small": 4},
		func(_ context.Context, id uuid.UUID) server.TenantLimits {
			if id == slowTenant {
				return server.TenantLimits{MaxInFlight: 1, Queue: "small"}
			}
			return server.TenantLimits{}
		},
	))
	require.NoError(t, sp.srv.CreateTenant(ctx, slowTenant, "slow"))
	fastTenant := sp.addTenant(t, "fast")

	// The slow receiver takes a second per request and records the most
	// requests it ever had in flight at once.
	var inFlight, maxInFlight, slowDone atomic.Int32
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		n := inFlight.Add(1)
		for {
			m := maxInFlight.Load()
			if n <= m || maxInFlight.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(time.Second)
		inFlight.Add(-1)
		slowDone.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(slow.Close)
	fast := newCountingReceiver(t)

	const consumer = "acme"
	const event = "order.created"
	setup := func(c *restClient, url string) {
		resp, err := c.post(ctx, "/v1/event-types", map[string]any{"name": event, "active": true}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		resp, err = c.post(ctx, "/v1/consumers/"+consumer+"/webhooks", map[string]any{
			"events": []string{event}, "url": url + "/hook", "active": true,
		}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	}
	push := func(c *restClient) {
		resp, err := c.post(ctx, "/v1/consumers/"+consumer+"/events?event="+event,
			map[string]any{"payload": map[string]any{"n": 1}}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	slowClient, fastClient := sp.client(t, slowTenant), sp.client(t, fastTenant)
	setup(slowClient, slow.URL)
	setup(fastClient, fast.srv.URL)

	const backlog = 4
	for range backlog {
		push(slowClient)
	}
	push(fastClient)

	// The uncapped tenant is served while the capped one still waits.
	fast.waitFor(t, 1)
	require.Less(t, slowDone.Load(), int32(backlog), "the capped tenant's backlog finished before the other tenant was served")

	require.Eventually(t, func() bool { return slowDone.Load() == backlog }, 60*time.Second, 100*time.Millisecond)
	require.EqualValues(t, 1, maxInFlight.Load(), "the capped tenant had more than MaxInFlight requests in flight")

	// Its jobs ran on its own queue.
	var queues []string
	require.NoError(t, sp.db.Select(&queues, `
		SELECT DISTINCT queue FROM river_job WHERE args @> jsonb_build_object('tenant_id', $1::text)`, slowTenant.String()))
	require.Equal(t, []string{"small"}, queues)

	// Waiting at the cap used no delivery attempt.
	var attempts []int
	require.NoError(t, sp.db.Select(&attempts, `
		SELECT wd.attempt_count FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wr.id = wd.webhook_id
		WHERE wr.tenant_id = $1`, slowTenant))
	require.Len(t, attempts, backlog)
	for _, a := range attempts {
		require.Equal(t, 1, a, "a snooze at the cap was counted as a delivery attempt")
	}
}
