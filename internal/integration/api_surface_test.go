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

// status calls the API and returns the response status, failing the test on
// a transport error.
func (c *restClient) status(ctx context.Context, method, path string, body, out any) int {
	c.t.Helper()
	resp, err := c.do(ctx, method, path, body, out)
	require.NoError(c.t, err, "%s %s", method, path)
	return resp.StatusCode
}

// TestAPI_Surface walks the management API against the real stack: each
// resource's read, update and lifecycle endpoints, plus the not-found and
// bad-input answers clients rely on.
func TestAPI_Surface(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		consumer  = "surface"
		eventName = "surface.order_created"
		missingID = "00000000-0000-0000-0000-000000000001"
	)
	base := "/v1/consumers/" + consumer

	// Event types: register with a schema, read it back, list, versions,
	// validate payloads against it.
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{"total": map[string]any{"type": "number"}},
		"required":   []string{"total"},
	}
	require.Equal(t, http.StatusCreated, c.status(ctx, http.MethodPost, "/v1/event-types",
		map[string]any{"name": eventName, "active": true, "description": "orders", "event_schema": schema}, nil))
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, "/v1/event-types/"+eventName, nil, nil))
	require.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodGet, "/v1/event-types/surface.nope", nil, nil))

	var types struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, "/v1/event-types?consumer="+consumer+"&active_only=true", nil, &types))
	names := map[string]bool{}
	for _, it := range types.Items {
		names[it.Name] = true
	}
	assert.True(t, names[eventName], "listEventTypes includes %s", eventName)

	var versions struct {
		Items []map[string]any `json:"items"`
	}
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, "/v1/event-types/"+eventName+"/versions", nil, &versions))
	require.NotEmpty(t, versions.Items)
	assert.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, "/v1/event-types/"+eventName+"/versions/1", nil, nil))
	assert.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodGet, "/v1/event-types/"+eventName+"/versions/99", nil, nil))

	var valid struct {
		Valid    bool     `json:"valid"`
		Warnings []string `json:"warnings"`
	}
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodPost, "/v1/event-types/"+eventName+":validate",
		map[string]any{"payload": map[string]any{"total": 12.5}}, &valid))
	assert.True(t, valid.Valid)
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodPost, "/v1/event-types/"+eventName+":validate",
		map[string]any{"payload": map[string]any{"total": "twelve"}}, &valid))
	assert.False(t, valid.Valid)
	assert.NotEmpty(t, valid.Warnings)

	assert.Equal(t, http.StatusOK, c.status(ctx, http.MethodPatch, "/v1/event-types/"+eventName,
		map[string]any{"description": "orders, updated"}, nil))

	// Webhooks: register, read, update, rotate the secret, pause and resume.
	target, requests := startCountingTarget(t)
	webhookID := registerWebhookPipeline(t, c, ctx, consumer, eventName, target.URL, 2)
	wh := base + "/webhooks/" + webhookID

	var got struct {
		WebhookID string `json:"webhook_id"`
		URL       string `json:"url"`
	}
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, wh, nil, &got))
	assert.Equal(t, webhookID, got.WebhookID)
	assert.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodGet, base+"/webhooks/"+missingID, nil, nil))
	assert.Equal(t, http.StatusBadRequest, c.status(ctx, http.MethodGet, base+"/webhooks/not-a-uuid", nil, nil))

	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodPatch, wh, map[string]any{
		"description": "surface target",
		"http_config": map[string]any{"max_retries": 3, "retry_backoff_seconds": 1, "request_timeout_seconds": 5},
		"headers":     map[string]string{"X-Team": "payments"},
	}, nil))
	assert.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodPatch, base+"/webhooks/"+missingID,
		map[string]any{"description": "x"}, nil))

	var rotated struct {
		WebhookSecret string `json:"webhook_secret"`
	}
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodPost, wh+":rotateSecret", nil, &rotated))
	assert.NotEmpty(t, rotated.WebhookSecret)
	assert.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodPost, base+"/webhooks/"+missingID+":rotateSecret", nil, nil))

	require.Equal(t, http.StatusNoContent, c.status(ctx, http.MethodPost, wh+":pause", nil, nil))
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodPost, wh+":resume", nil, nil))
	assert.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodPost, base+"/webhooks/"+missingID+":pause", nil, nil))

	assert.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, wh+"/health", nil, nil))
	assert.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, base+"/webhooks?limit=5", nil, nil))
	assert.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, "/v1/webhooks?health=unknown", nil, nil))

	// Subscriptions: list, read, update the template, history, pause, resume.
	var subs struct {
		Items []struct {
			SubscriptionID string `json:"subscription_id"`
		} `json:"items"`
	}
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, base+"/subscriptions?webhook_id="+webhookID, nil, &subs))
	require.Len(t, subs.Items, 1)
	sub := base + "/subscriptions/" + subs.Items[0].SubscriptionID
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, sub, nil, nil))
	assert.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodGet, base+"/subscriptions/"+missingID, nil, nil))
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodPatch, sub, map[string]any{
		"transform_enabled":  true,
		"transform_template": `{"total": {{ .payload.total }}}`,
		"template_notes":     "first template",
	}, nil))
	assert.Equal(t, http.StatusBadRequest, c.status(ctx, http.MethodPatch, sub, map[string]any{
		"transform_enabled": true, "transform_template": "{{ .payload.total ",
	}, nil))
	var history struct {
		Items []map[string]any `json:"items"`
	}
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, sub+"/templateVersions", nil, &history))
	assert.NotEmpty(t, history.Items)
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodPost, sub+":pause", map[string]any{"reason": "template review"}, nil))
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodPost, sub+":resume", nil, nil))
	assert.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, "/v1/subscriptions?event_name="+eventName, nil, nil))

	// Events and deliveries: push, read the occurrence, read and retry the
	// delivery under the consumer and globally, re-push.
	var pushed struct {
		EventID string `json:"event_id"`
	}
	require.Equal(t, http.StatusCreated, c.status(ctx, http.MethodPost, base+"/events?event="+eventName, map[string]any{
		"payload": map[string]any{"total": 42}, "ttl_seconds": 300, "labels": map[string]string{"region": "eu"},
	}, &pushed))
	eventID := pushed.EventID

	pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	delivered := pollDeliveryStatus(t, c, pollCtx, consumer, eventID, func(d deliveryItem) bool { return d.Status == "success" })
	cancel()

	var occ struct {
		EventID string         `json:"event_id"`
		Payload map[string]any `json:"payload"`
	}
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, "/v1/events/"+eventID, nil, &occ))
	assert.Equal(t, eventID, occ.EventID)
	assert.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodGet, "/v1/events/"+missingID, nil, nil))
	assert.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, "/v1/events?event="+eventName, nil, nil))
	assert.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, base+"/events?event="+eventName, nil, nil))

	var dl struct {
		DeliveryID string `json:"delivery_id"`
		Status     string `json:"status"`
	}
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, base+"/deliveries/"+delivered.DeliveryID, nil, &dl))
	assert.Equal(t, "success", dl.Status)
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, "/v1/deliveries/"+delivered.DeliveryID, nil, nil))
	assert.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodGet, base+"/deliveries/"+missingID, nil, nil))
	assert.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodGet, "/v1/deliveries/"+missingID, nil, nil))
	assert.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodGet, "/v1/consumers/other/deliveries/"+delivered.DeliveryID, nil, nil),
		"a delivery is not visible under another consumer")
	assert.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, base+"/deliveries/"+delivered.DeliveryID+"/attempts", nil, nil))
	assert.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, "/v1/deliveries/"+delivered.DeliveryID+"/attempts", nil, nil))

	before := requests.Load()
	// A delivery that already succeeded is not retried; re-push resends it.
	assert.Equal(t, http.StatusConflict, c.status(ctx, http.MethodPost, base+"/deliveries/"+delivered.DeliveryID+":retry", nil, nil))
	assert.Equal(t, http.StatusConflict, c.status(ctx, http.MethodPost, "/v1/deliveries/"+delivered.DeliveryID+":retry", nil, nil))
	assert.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodPost, base+"/deliveries/"+missingID+":retry", nil, nil))
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodPost, "/v1/events/"+eventID+":repush", nil, nil))
	assert.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodPost, "/v1/events/"+missingID+":repush", nil, nil))
	require.Eventually(t, func() bool { return requests.Load() > before }, 30*time.Second, 200*time.Millisecond,
		"the re-pushed delivery reaches the receiver")

	// Read-only views and catalogs.
	for _, path := range []string{
		base + "/stats", "/v1/stats", "/v1/health-summary", "/v1/consumers?q=surf",
		"/v1/template-functions", "/v1/capabilities", "/v1/recipes", "/v1/whoami",
	} {
		assert.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, path, nil, nil), path)
	}

	// Alert configs: create, list, delete.
	var alert struct {
		ID string `json:"id"`
	}
	require.Equal(t, http.StatusCreated, c.status(ctx, http.MethodPost, base+"/alert-configs", map[string]any{
		"email": "oncall@example.com", "event_types": []string{"sparrow.webhook.disabled"}, "webhook_id": webhookID,
	}, &alert))
	var alerts struct {
		Items []map[string]any `json:"items"`
	}
	require.Equal(t, http.StatusOK, c.status(ctx, http.MethodGet, base+"/alert-configs", nil, &alerts))
	assert.Len(t, alerts.Items, 1)
	assert.Equal(t, http.StatusNoContent, c.status(ctx, http.MethodDelete, base+"/alert-configs/"+alert.ID, nil, nil))

	// Teardown: delete the subscription and the webhook; both are gone after.
	assert.Equal(t, http.StatusNoContent, c.status(ctx, http.MethodDelete, sub, nil, nil))
	assert.Equal(t, http.StatusNoContent, c.status(ctx, http.MethodDelete, wh, nil, nil))
	assert.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodGet, wh, nil, nil))
	assert.Equal(t, http.StatusNotFound, c.status(ctx, http.MethodDelete, wh, nil, nil))
}
