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

// TestE2E_WebhookSoftDelete checks that deleting a webhook hides it and stops
// fan-out to it while keeping its delivery history, and that its URL can be
// registered again.
func TestE2E_WebhookSoftDelete(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		consumer  = "soft-delete"
		eventName = "soft.delete"
	)
	target, hits := startCountingTarget(t)
	registerEventType(t, c, ctx, eventName)
	oldID := registerWebhookPipeline(t, c, ctx, consumer, eventName, target.URL, 0)

	firstEvent := pushTestEvent(t, c, ctx, consumer, eventName)
	pollCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	delivered := pollDeliveryStatus(t, c, pollCtx, consumer, firstEvent, func(d deliveryItem) bool { return d.Status == "success" })

	resp, err := c.do(ctx, http.MethodDelete, "/v1/consumers/"+consumer+"/webhooks/"+oldID, nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Gone from webhook reads, and a second delete is a 404.
	resp, err = c.get(ctx, "/v1/consumers/"+consumer+"/webhooks/"+oldID, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp, err = c.do(ctx, http.MethodDelete, "/v1/consumers/"+consumer+"/webhooks/"+oldID, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	var list struct {
		TotalCount int `json:"total_count"`
	}
	_, err = c.get(ctx, "/v1/webhooks?consumer="+consumer, &list)
	require.NoError(t, err)
	assert.Equal(t, 0, list.TotalCount)

	// Its delivery history is kept.
	var deliveries struct {
		Items []deliveryItem `json:"items"`
	}
	_, err = c.get(ctx, "/v1/consumers/"+consumer+"/deliveries?event_id="+firstEvent, &deliveries)
	require.NoError(t, err)
	require.Len(t, deliveries.Items, 1)
	assert.Equal(t, delivered.DeliveryID, deliveries.Items[0].DeliveryID)

	// The same URL registers again as a new webhook, and only it receives
	// the next event.
	newID := registerWebhookPipeline(t, c, ctx, consumer, eventName, target.URL, 0)
	assert.NotEqual(t, oldID, newID)
	secondEvent := pushTestEvent(t, c, ctx, consumer, eventName)
	pollDeliveryStatus(t, c, pollCtx, consumer, secondEvent, func(d deliveryItem) bool { return d.Status == "success" })
	deliveries.Items = nil
	_, err = c.get(ctx, "/v1/consumers/"+consumer+"/deliveries?event_id="+secondEvent, &deliveries)
	require.NoError(t, err)
	assert.Len(t, deliveries.Items, 1, "the deleted webhook must not get a delivery")
	assert.Equal(t, int32(2), hits.Load())
}
