//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// TestE2E_SystemConsumerScope checks that Sparrow's own _sparrow consumer
// (the alert mailer) is listed and counted like any other consumer, can be
// scoped to on its own, and that alert delivery is reported as configured once
// a _sparrow webhook subscribes to the system events.
func TestE2E_SystemConsumerScope(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	repo := store.NewRepository(env.sqlxDB)
	for _, reg := range queue.SystemEventRegistrations() {
		reg := reg
		require.NoError(t, repo.RegisterEvent(ctx, tenant.DefaultTenantID, &reg))
	}

	type capabilities struct {
		AlertDelivery struct {
			Configured bool `json:"configured"`
		} `json:"alert_delivery"`
	}
	var caps capabilities
	_, err := c.get(ctx, "/v1/capabilities", &caps)
	require.NoError(t, err)
	assert.False(t, caps.AlertDelivery.Configured, "no _sparrow webhook yet")

	register := func(consumer string, events []string) {
		t.Helper()
		resp, err := c.post(ctx, "/v1/consumers/"+consumer+"/webhooks", map[string]any{
			"url":    "http://127.0.0.1:1/hook",
			"events": events,
		}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode, "register under %s", consumer)
	}
	register("tenant-a", []string{"order.created"})
	register(tenant.SystemConsumer, []string{"sparrow.webhook.delivery_failed", "*"})

	_, err = c.get(ctx, "/v1/capabilities", &caps)
	require.NoError(t, err)
	assert.True(t, caps.AlertDelivery.Configured)

	type page struct {
		Items []struct {
			Consumer string `json:"consumer"`
		} `json:"items"`
		Pagination struct {
			Total int `json:"total_count"`
		} `json:"pagination"`
	}
	total := func(path string) int {
		t.Helper()
		var p page
		resp, err := c.get(ctx, path, &p)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, path)
		return p.Pagination.Total
	}

	// Deliveries and event occurrences have no total_count (cursor paged):
	// these sets are small, so count the items on the first page.
	items := func(path string) int {
		t.Helper()
		var p page
		resp, err := c.get(ctx, path, &p)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, path)
		return len(p.Items)
	}

	type consumerList struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	names := func(path string) []string {
		t.Helper()
		var cs consumerList
		resp, err := c.get(ctx, path, &cs)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, path)
		out := []string{}
		for _, it := range cs.Items {
			out = append(out, it.Name)
		}
		return out
	}
	assert.Equal(t, []string{tenant.SystemConsumer, "tenant-a"}, names("/v1/consumers"))
	assert.Equal(t, []string{"tenant-a"}, names("/v1/consumers?q=TENANT"))
	assert.Equal(t, []string{tenant.SystemConsumer}, names("/v1/consumers?limit=1"))

	assert.Equal(t, 2, total("/v1/webhooks"))
	assert.Equal(t, 1, total("/v1/webhooks?consumer=_sparrow"))

	var stats struct {
		TotalWebhooks int `json:"total_webhooks"`
	}
	_, err = c.get(ctx, "/v1/stats", &stats)
	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalWebhooks)

	type summary struct {
		UnknownCount int `json:"unknown_count"`
		Rules        struct {
			WindowHours                  int     `json:"window_hours"`
			UnhealthyConsecutiveFailures int     `json:"unhealthy_consecutive_failures"`
			DegradedSuccessRate          float64 `json:"degraded_success_rate"`
		} `json:"rules"`
	}
	var sum summary
	_, err = c.get(ctx, "/v1/health-summary", &sum)
	require.NoError(t, err)
	assert.Equal(t, 2, sum.UnknownCount)
	assert.Equal(t, store.DefaultHealthRules.WindowHours, sum.Rules.WindowHours)
	assert.Equal(t, store.DefaultHealthRules.UnhealthyConsecutiveFailures, sum.Rules.UnhealthyConsecutiveFailures)
	assert.Equal(t, store.DefaultHealthRules.DegradedSuccessRate, sum.Rules.DegradedSuccessRate)
	_, err = c.get(ctx, "/v1/health-summary?consumer=tenant-a", &sum)
	require.NoError(t, err)
	assert.Equal(t, 1, sum.UnknownCount)

	// One occurrence (and delivery) per consumer.
	for _, consumer := range []string{"tenant-a", tenant.SystemConsumer} {
		resp, err := c.post(ctx, "/v1/consumers/"+consumer+"/events?event=order.created",
			map[string]any{"payload": map[string]any{"id": 1}}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode, "push under %s", consumer)
	}
	require.Eventually(t, func() bool {
		return items("/v1/deliveries") == 2
	}, 15*time.Second, 200*time.Millisecond)

	assert.Equal(t, 2, items("/v1/events"))
	assert.Equal(t, 1, items("/v1/deliveries?consumer=_sparrow"))

	var dl page
	_, err = c.get(ctx, "/v1/deliveries", &dl)
	require.NoError(t, err)
	consumers := map[string]bool{}
	for _, it := range dl.Items {
		consumers[it.Consumer] = true
	}
	assert.Equal(t, map[string]bool{"tenant-a": true, tenant.SystemConsumer: true}, consumers, "deliveries carry their webhook's consumer")
}
