//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type consumerStats struct {
	TotalWebhooks        int     `json:"total_webhooks"`
	TotalDeliveries      int     `json:"total_deliveries"`
	SuccessfulDeliveries int     `json:"successful_deliveries"`
	FailedDeliveries     int     `json:"failed_deliveries"`
	PendingDeliveries    int     `json:"pending_deliveries"`
	SuccessRate          float64 `json:"success_rate"`
}

// TestE2E_ConsumerStatsCountAttempts checks that consumer and global stats
// come from the health evaluator's running totals: a delivery that fails
// twice and then succeeds counts as three attempts, two of them failed.
func TestE2E_ConsumerStatsCountAttempts(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		consumer  = "stats-attempts"
		eventName = "stats.attempts"
	)
	target, _ := startFailThenSucceedTarget(t, 2, http.StatusInternalServerError)
	registerEventType(t, c, ctx, eventName)
	registerWebhookPipeline(t, c, ctx, consumer, eventName, target.URL, 3)
	eventID := pushTestEvent(t, c, ctx, consumer, eventName)

	pollCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	pollDeliveryStatus(t, c, pollCtx, consumer, eventID, func(d deliveryItem) bool { return d.Status == "success" })

	// Totals appear after the next evaluator pass (2s in the test env).
	var stats consumerStats
	require.Eventually(t, func() bool {
		resp, err := c.get(ctx, "/v1/consumers/"+consumer+"/stats", &stats)
		return err == nil && resp.StatusCode == http.StatusOK && stats.TotalDeliveries == 3
	}, 60*time.Second, 500*time.Millisecond, "stats never reached 3 attempts: %+v", stats)
	assert.Equal(t, 1, stats.TotalWebhooks)
	assert.Equal(t, 1, stats.SuccessfulDeliveries)
	assert.Equal(t, 2, stats.FailedDeliveries)
	assert.Equal(t, 0, stats.PendingDeliveries)
	assert.InDelta(t, 1.0/3.0, stats.SuccessRate, 0.0001)

	var global consumerStats
	resp, err := c.get(ctx, "/v1/stats", &global)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.GreaterOrEqual(t, global.TotalDeliveries, 3)
	assert.GreaterOrEqual(t, global.FailedDeliveries, 2)
}

// TestE2E_DeliveryStatusFilter checks the status filter compares the enum:
// a known status filters, an unknown one is a validation error instead of a
// database error.
func TestE2E_DeliveryStatusFilter(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		consumer  = "status-filter"
		eventName = "status.filter"
	)
	target, _ := startCountingTarget(t)
	registerEventType(t, c, ctx, eventName)
	registerWebhookPipeline(t, c, ctx, consumer, eventName, target.URL, 0)
	eventID := pushTestEvent(t, c, ctx, consumer, eventName)

	pollCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	pollDeliveryStatus(t, c, pollCtx, consumer, eventID, func(d deliveryItem) bool { return d.Status == "success" })

	var out struct {
		Items []deliveryItem `json:"items"`
	}
	resp, err := c.get(ctx, "/v1/consumers/"+consumer+"/deliveries?status=success", &out)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, out.Items, 1)

	out.Items = nil
	resp, err = c.get(ctx, "/v1/consumers/"+consumer+"/deliveries?status=failed", &out)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, out.Items)

	resp, err = c.get(ctx, "/v1/consumers/"+consumer+"/deliveries?status=bogus", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

// TestE2E_ConsumerDeliveryPaging pages a consumer's deliveries across several
// webhooks (the per-webhook merge used when only a consumer is given): every
// delivery appears once, newest first, across page boundaries.
func TestE2E_ConsumerDeliveryPaging(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		consumer  = "consumer-paging"
		eventName = "consumer.paging"
	)
	registerEventType(t, c, ctx, eventName)
	for i := 0; i < 3; i++ {
		target, _ := startCountingTarget(t)
		registerWebhookPipeline(t, c, ctx, consumer, eventName, target.URL, 0)
	}
	var last string
	for i := 0; i < 3; i++ {
		last = pushTestEvent(t, c, ctx, consumer, eventName)
	}
	pollCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	pollDeliveryStatus(t, c, pollCtx, consumer, last, func(d deliveryItem) bool { return d.Status == "success" })

	type item struct {
		DeliveryID string    `json:"delivery_id"`
		CreatedAt  time.Time `json:"created_at"`
	}
	seen := map[string]bool{}
	var prev time.Time
	cursor := ""
	for pages := 0; pages < 10; pages++ {
		var out struct {
			Items      []item `json:"items"`
			Pagination struct {
				HasMore    bool   `json:"has_more"`
				NextCursor string `json:"next_cursor"`
			} `json:"pagination"`
		}
		path := "/v1/deliveries?consumer=" + consumer + "&limit=4"
		if cursor != "" {
			path += "&cursor=" + cursor
		}
		resp, err := c.get(ctx, path, &out)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		for _, it := range out.Items {
			assert.False(t, seen[it.DeliveryID], "delivery %s listed twice", it.DeliveryID)
			seen[it.DeliveryID] = true
			if !prev.IsZero() {
				assert.False(t, it.CreatedAt.After(prev), "not newest first")
			}
			prev = it.CreatedAt
		}
		if !out.Pagination.HasMore {
			break
		}
		cursor = out.Pagination.NextCursor
	}
	assert.Len(t, seen, 9, "3 events x 3 webhooks")
}
