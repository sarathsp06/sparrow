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

type requiresTransformSub struct {
	SubscriptionID    string `json:"subscription_id"`
	EventName         string `json:"event_name"`
	TransformEnabled  bool   `json:"transform_enabled"`
	TransformTemplate string `json:"transform_template"`
	OnTransformError  string `json:"on_transform_error"`
}

func listWebhookSubs(t *testing.T, c *restClient, ctx context.Context, consumer, webhookID string) []requiresTransformSub {
	t.Helper()
	var out struct {
		Items []requiresTransformSub `json:"items"`
	}
	_, err := c.get(ctx, "/v1/consumers/"+consumer+"/subscriptions?webhook_id="+webhookID, &out)
	require.NoError(t, err)
	return out.Items
}

// A webhook registered with requires_transform gets its template on every
// created subscription, and the API then refuses anything that would leave a
// subscription without one.
func TestRequiresTransform_EnforcedOnEveryWrite(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()
	const consumer = "req-transform"
	registerEventType(t, c, ctx, "order.created")
	registerEventType(t, c, ctx, "order.refunded")
	srv, rec := startBodyRecorder(t)
	const tmpl = `{"text": {{printf "%s: %v" .event_name .payload.id | json}}}`

	// Without a template there is nothing to give the created subscriptions.
	resp, err := c.post(ctx, "/v1/consumers/"+consumer+"/webhooks", map[string]any{
		"url": srv.URL, "events": []string{"order.created"}, "requires_transform": true,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var hook struct {
		WebhookID         string `json:"webhook_id"`
		RequiresTransform bool   `json:"requires_transform"`
	}
	resp, err = c.post(ctx, "/v1/consumers/"+consumer+"/webhooks", map[string]any{
		"url": srv.URL, "events": []string{"order.created", "order.refunded"},
		"requires_transform": true, "transform_template": tmpl, "on_transform_error": "fallback",
		"template_source": "ai_draft", "template_notes": "drafted for Slack",
	}, &hook)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.True(t, hook.RequiresTransform)

	subs := listWebhookSubs(t, c, ctx, consumer, hook.WebhookID)
	require.Len(t, subs, 2)
	for _, s := range subs {
		assert.True(t, s.TransformEnabled, s.EventName)
		assert.Equal(t, tmpl, s.TransformTemplate, s.EventName)
		assert.Equal(t, "fallback", s.OnTransformError, s.EventName)
	}
	// The registration template is each subscription's first saved version.
	var versions struct {
		Items []struct {
			Source string `json:"source"`
			Notes  string `json:"notes"`
		} `json:"items"`
	}
	_, err = c.get(ctx, "/v1/consumers/"+consumer+"/subscriptions/"+subs[0].SubscriptionID+"/templateVersions", &versions)
	require.NoError(t, err)
	require.Len(t, versions.Items, 1)
	assert.Equal(t, "ai_draft", versions.Items[0].Source)
	assert.Equal(t, "drafted for Slack", versions.Items[0].Notes)

	// Turning the transform off, or adding a subscription without one, is refused.
	resp, err = c.do(ctx, http.MethodPatch, "/v1/consumers/"+consumer+"/subscriptions/"+subs[0].SubscriptionID,
		map[string]any{"transform_enabled": false}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp, err = c.post(ctx, "/v1/consumers/"+consumer+"/subscriptions", map[string]any{
		"webhook_id": hook.WebhookID, "event_name": "*",
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp, err = c.post(ctx, "/v1/consumers/"+consumer+"/subscriptions", map[string]any{
		"webhook_id": hook.WebhookID, "event_name": "*", "transform_enabled": true, "transform_template": tmpl,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Replacing events in bulk would recreate subscriptions without templates.
	resp, err = c.do(ctx, http.MethodPatch, "/v1/consumers/"+consumer+"/webhooks/"+hook.WebhookID,
		map[string]any{"events": []string{"order.created"}}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	// Deliveries go out transformed.
	pushPayload(t, c, ctx, consumer, "order.created", map[string]any{"id": "o-1"})
	require.Eventually(t, func() bool { return len(rec.all()) > 0 }, 30*time.Second, 300*time.Millisecond)
	assert.JSONEq(t, `{"text": "order.created: o-1"}`, string(rec.all()[0]))
}

// requires_transform can only be turned on once every subscription has a transform.
func TestRequiresTransform_TurnOnNeedsTransforms(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()
	const consumer = "req-transform-on"
	registerEventType(t, c, ctx, "ticket.opened")
	srv, _ := startBodyRecorder(t)
	webhookID := registerWebhookPipeline(t, c, ctx, consumer, "ticket.opened", srv.URL, 3)
	path := "/v1/consumers/" + consumer + "/webhooks/" + webhookID

	resp, err := c.do(ctx, http.MethodPatch, path, map[string]any{"requires_transform": true}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	sub := listWebhookSubs(t, c, ctx, consumer, webhookID)[0]
	resp, err = c.do(ctx, http.MethodPatch, "/v1/consumers/"+consumer+"/subscriptions/"+sub.SubscriptionID,
		map[string]any{"transform_enabled": true, "transform_template": `{"id": {{.payload.id | json}}}`}, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var hook struct {
		RequiresTransform bool `json:"requires_transform"`
	}
	resp, err = c.do(ctx, http.MethodPatch, path, map[string]any{"requires_transform": true}, &hook)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, hook.RequiresTransform)
}

// A subscription that has no transform despite the webhook requiring one (only
// possible by editing the database) fails the delivery instead of sending the
// default envelope.
func TestRequiresTransform_DeliveryWithoutTransformFails(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()
	const consumer, eventName = "req-transform-legacy", "lead.created"
	srv, rec := startBodyRecorder(t)
	webhookID, subID := setupTemplateSubscription(t, c, ctx, consumer, eventName, srv.URL, `{"ok": true}`,
		map[string]any{"on_transform_error": "fallback"})
	resp, err := c.do(ctx, http.MethodPatch, "/v1/consumers/"+consumer+"/webhooks/"+webhookID,
		map[string]any{"requires_transform": true}, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	_, err = env.sqlxDB.ExecContext(ctx, `UPDATE event_subscriptions SET transform_enabled = false WHERE id = $1`, subID)
	require.NoError(t, err)

	eventID := pushPayload(t, c, ctx, consumer, eventName, map[string]any{"id": "l-1"})
	d := waitForDelivery(t, c, ctx, consumer, eventID, func(d templateDelivery) bool { return d.Status == "failed" })
	assert.Equal(t, "template_error", d.ErrorCategory)
	assert.Contains(t, d.ErrorMessage, "requires a payload transform")
	time.Sleep(time.Second)
	assert.Empty(t, rec.all(), "the default envelope must never reach a receiver that requires a transform")
}
