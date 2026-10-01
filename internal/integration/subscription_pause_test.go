//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type subscriptionResp struct {
	SubscriptionID string  `json:"subscription_id"`
	Paused         bool    `json:"paused"`
	PausedAt       *string `json:"paused_at"`
	PausedReason   string  `json:"paused_reason"`
}

func TestSubscriptionPause_HoldsDeliveriesAsPausedRows(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()
	const consumer = "pause-test"

	srv, rec := startBodyRecorder(t)
	registerEventType(t, c, ctx, "ticket.opened")
	registerEventType(t, c, ctx, "ticket.closed")
	webhookID := registerWebhookPipeline(t, c, ctx, consumer, "ticket.opened", srv.URL, 0)

	// A second subscription on the same webhook must be unaffected.
	var other subscriptionResp
	resp, err := c.post(ctx, "/v1/consumers/"+consumer+"/subscriptions", map[string]any{"webhook_id": webhookID, "event_name": "ticket.closed"}, &other)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var subs struct {
		Items []subscriptionResp `json:"items"`
	}
	_, err = c.get(ctx, "/v1/consumers/"+consumer+"/subscriptions?event_name=ticket.opened", &subs)
	require.NoError(t, err)
	require.Len(t, subs.Items, 1)
	subID := subs.Items[0].SubscriptionID

	var paused subscriptionResp
	resp, err = c.post(ctx, "/v1/consumers/"+consumer+"/subscriptions/"+subID+":pause", map[string]any{"reason": "receiver maintenance"}, &paused)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, paused.Paused)
	assert.Equal(t, "receiver maintenance", paused.PausedReason)
	require.NotNil(t, paused.PausedAt)

	// While paused: deliveries are recorded as paused and nothing is sent;
	// the other subscription still delivers.
	held1 := pushPayload(t, c, ctx, consumer, "ticket.opened", map[string]any{"n": 1})
	held2 := pushPayload(t, c, ctx, consumer, "ticket.opened", map[string]any{"n": 2})
	closed := pushPayload(t, c, ctx, consumer, "ticket.closed", map[string]any{"n": 3})
	waitForDelivery(t, c, ctx, consumer, held1, func(d templateDelivery) bool { return d.Status == "paused" })
	waitForDelivery(t, c, ctx, consumer, held2, func(d templateDelivery) bool { return d.Status == "paused" })
	waitForDelivery(t, c, ctx, consumer, closed, func(d templateDelivery) bool { return d.Status == "success" })
	time.Sleep(time.Second)
	require.Len(t, rec.all(), 1, "only the unpaused subscription delivered")

	// A pause is not a receiver failure.
	var health struct {
		FailedDeliveries int `json:"failed_deliveries"`
	}
	_, err = c.get(ctx, "/v1/consumers/"+consumer+"/webhooks/"+webhookID+"/health", &health)
	require.NoError(t, err)
	assert.Zero(t, health.FailedDeliveries)

	// Resume: new events flow again; held deliveries stay paused until retried.
	var resumed struct {
		subscriptionResp
		PausedSince      *string `json:"paused_since"`
		PausedDeliveries int     `json:"paused_deliveries"`
	}
	resp, err = c.post(ctx, "/v1/consumers/"+consumer+"/subscriptions/"+subID+":resume", nil, &resumed)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.False(t, resumed.Paused)
	assert.Equal(t, 2, resumed.PausedDeliveries)
	require.NotNil(t, resumed.PausedSince)
	assert.Equal(t, *paused.PausedAt, *resumed.PausedSince)

	after := pushPayload(t, c, ctx, consumer, "ticket.opened", map[string]any{"n": 4})
	waitForDelivery(t, c, ctx, consumer, after, func(d templateDelivery) bool { return d.Status == "success" })
	d := waitForDelivery(t, c, ctx, consumer, held1, func(d templateDelivery) bool { return true })
	assert.Equal(t, "paused", d.Status, "held deliveries are never sent automatically")

	// Retry what was held since the pause began, with the existing batch retry.
	var list struct {
		Items   []templateDelivery `json:"items"`
		RetryID string             `json:"retry_id"`
	}
	q := url.Values{
		"status":          {"paused"},
		"subscription_id": {subID},
		"created_after":   {*resumed.PausedSince},
		"prepare_retry":   {"true"},
	}
	_, err = c.get(ctx, "/v1/consumers/"+consumer+"/deliveries?"+q.Encode(), &list)
	require.NoError(t, err)
	require.Len(t, list.Items, 2)
	require.NotEmpty(t, list.RetryID)
	var job struct {
		ID string `json:"id"`
	}
	resp, err = c.post(ctx, "/v1/consumers/"+consumer+"/deliveries:retryBatch", map[string]any{"repush_id": list.RetryID}, &job)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	waitForDelivery(t, c, ctx, consumer, held1, func(d templateDelivery) bool { return d.Status == "success" })
	waitForDelivery(t, c, ctx, consumer, held2, func(d templateDelivery) bool { return d.Status == "success" })
	assert.Len(t, rec.all(), 4)
}

func TestSubscriptionPause_ImportCanPauseFailingSubscriptions(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	_, err := c.post(ctx, "/v1/event-types", map[string]any{"name": "invoice.paid", "event_schema": totalSchema("number")}, nil)
	require.NoError(t, err)
	srv, _ := startBodyRecorder(t)
	_, subID := setupTemplateSubscription(t, c, ctx, "billing", "invoice.paid", srv.URL, `{"amount": {{.payload.total}}}`, nil)

	breaking := map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}}
	b := bundle{APIVersion: "sparrow/v1", Kind: "EventTypeList", Items: []map[string]any{{"name": "invoice.paid", "event_schema": breaking}}}

	// A dry run with the policy pauses nothing.
	_, res := importBundle(t, c, ctx, b, map[string]any{"allow_breaking": true, "subscription_policy": "pause", "dry_run": true})
	assert.False(t, res.Applied)
	var sub subscriptionResp
	_, err = c.get(ctx, "/v1/consumers/billing/subscriptions/"+subID, &sub)
	require.NoError(t, err)
	assert.False(t, sub.Paused)

	_, res = importBundle(t, c, ctx, b, map[string]any{"allow_breaking": true, "subscription_policy": "pause"})
	require.True(t, res.Applied)
	_, err = c.get(ctx, "/v1/consumers/billing/subscriptions/"+subID, &sub)
	require.NoError(t, err)
	assert.True(t, sub.Paused)
	assert.Contains(t, sub.PausedReason, "invoice.paid moved to v2 by import")
}
