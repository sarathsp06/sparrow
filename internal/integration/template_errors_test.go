//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bodyRecorder is a webhook target that returns 200 and keeps every body.
type bodyRecorder struct {
	mu     sync.Mutex
	bodies [][]byte
}

func startBodyRecorder(t *testing.T) (*httptest.Server, *bodyRecorder) {
	t.Helper()
	rec := &bodyRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		rec.mu.Lock()
		rec.bodies = append(rec.bodies, b)
		rec.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

func (r *bodyRecorder) all() [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([][]byte(nil), r.bodies...)
}

type templateDelivery struct {
	DeliveryID    string `json:"delivery_id"`
	Status        string `json:"status"`
	ErrorCategory string `json:"error_category"`
	ErrorMessage  string `json:"error_message"`
	TemplateError string `json:"template_error"`
}

// setupTemplateSubscription registers eventName, a webhook to target, and
// sets the auto-created subscription's transform and settings.
func setupTemplateSubscription(t *testing.T, c *restClient, ctx context.Context, consumer, eventName, targetURL, tmpl string, settings map[string]any) (webhookID, subscriptionID string) {
	t.Helper()
	registerEventType(t, c, ctx, eventName)
	webhookID = registerWebhookPipeline(t, c, ctx, consumer, eventName, targetURL, 3)

	var subs struct {
		Items []struct {
			SubscriptionID string `json:"subscription_id"`
		} `json:"items"`
	}
	_, err := c.get(ctx, "/v1/consumers/"+consumer+"/subscriptions?webhook_id="+webhookID, &subs)
	require.NoError(t, err)
	require.Len(t, subs.Items, 1)
	subscriptionID = subs.Items[0].SubscriptionID

	patch := map[string]any{"transform_enabled": true, "transform_template": tmpl}
	for k, v := range settings {
		patch[k] = v
	}
	resp, err := c.do(ctx, http.MethodPatch, "/v1/consumers/"+consumer+"/subscriptions/"+subscriptionID, patch, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	return webhookID, subscriptionID
}

func pushPayload(t *testing.T, c *restClient, ctx context.Context, consumer, eventName string, payload map[string]any) string {
	t.Helper()
	var out struct {
		EventID string `json:"event_id"`
	}
	resp, err := c.post(ctx, "/v1/consumers/"+consumer+"/events?event="+eventName, map[string]any{"payload": payload}, &out)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	return out.EventID
}

func waitForDelivery(t *testing.T, c *restClient, ctx context.Context, consumer, eventID string, done func(templateDelivery) bool) templateDelivery {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		var out struct {
			Items []templateDelivery `json:"items"`
		}
		_, err := c.get(ctx, "/v1/consumers/"+consumer+"/deliveries?event_id="+eventID, &out)
		if err == nil {
			for _, d := range out.Items {
				if done(d) {
					return d
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for delivery of event %s", eventID)
	return templateDelivery{}
}

func TestTemplateErrors_FailByDefault_ThenRetryAfterFix(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()
	const consumer, eventName = "tmpl-fail", "invoice.paid"

	srv, rec := startBodyRecorder(t)
	webhookID, subID := setupTemplateSubscription(t, c, ctx, consumer, eventName, srv.URL,
		`{"amount": {{.payload.amount}}}`, nil)

	// The payload has no "amount": with the defaults (template_missing_key
	// error, on_transform_error fail) the render fails and nothing is sent.
	eventID := pushPayload(t, c, ctx, consumer, eventName, map[string]any{"invoice_id": "inv-1"})
	d := waitForDelivery(t, c, ctx, consumer, eventID, func(d templateDelivery) bool { return d.Status == "failed" })
	assert.Equal(t, "template_error", d.ErrorCategory)
	assert.Contains(t, d.TemplateError, "amount")
	assert.Contains(t, d.ErrorMessage, "Template transformation failed")

	// Give River a moment: a template error must not be retried automatically.
	time.Sleep(2 * time.Second)
	assert.Empty(t, rec.all(), "a failed transform must not send anything")

	// It is a sender-side configuration fault: no attempt is recorded against
	// the receiver, and the webhook's health is untouched.
	var attempts struct {
		Items []json.RawMessage `json:"items"`
	}
	_, err := c.get(ctx, "/v1/consumers/"+consumer+"/deliveries/"+d.DeliveryID+"/attempts", &attempts)
	require.NoError(t, err)
	assert.Empty(t, attempts.Items)
	var health struct {
		Health           string `json:"health"`
		FailedDeliveries int    `json:"failed_deliveries"`
	}
	_, err = c.get(ctx, "/v1/consumers/"+consumer+"/webhooks/"+webhookID+"/health", &health)
	require.NoError(t, err)
	assert.NotEqual(t, "unhealthy", health.Health)
	assert.NotEqual(t, "degraded", health.Health)
	assert.Zero(t, health.FailedDeliveries)

	// Fix the template to read the field optionally, then retry: the retry
	// renders the fixed template and clears the recorded error.
	resp, err := c.do(ctx, http.MethodPatch, "/v1/consumers/"+consumer+"/subscriptions/"+subID,
		map[string]any{"transform_template": `{"amount": {{ dig "amount" 0 .payload }}}`}, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp, err = c.post(ctx, "/v1/consumers/"+consumer+"/deliveries/"+d.DeliveryID+":retry", nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	d = waitForDelivery(t, c, ctx, consumer, eventID, func(d templateDelivery) bool { return d.Status == "success" })
	assert.Empty(t, d.TemplateError)
	bodies := rec.all()
	require.Len(t, bodies, 1)
	assert.JSONEq(t, `{"amount": 0}`, string(bodies[0]))
}

func TestTemplateErrors_FallbackSendsEnvelope(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()
	const consumer, eventName = "tmpl-fallback", "invoice.voided"

	srv, rec := startBodyRecorder(t)
	setupTemplateSubscription(t, c, ctx, consumer, eventName, srv.URL,
		`{"amount": {{.payload.amount}}}`, map[string]any{"on_transform_error": "fallback"})

	eventID := pushPayload(t, c, ctx, consumer, eventName, map[string]any{"invoice_id": "inv-2"})
	d := waitForDelivery(t, c, ctx, consumer, eventID, func(d templateDelivery) bool { return d.Status == "success" })
	assert.NotEmpty(t, d.TemplateError, "the error is recorded even though the envelope was sent")

	bodies := rec.all()
	require.Len(t, bodies, 1)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(bodies[0], &envelope))
	assert.Equal(t, eventName, envelope["event_name"])
	assert.Equal(t, "inv-2", envelope["payload"].(map[string]any)["invoice_id"])
}

func TestTemplateErrors_MissingKeyZeroRendersNoValue(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()
	const consumer, eventName = "tmpl-zero", "invoice.sent"

	srv, rec := startBodyRecorder(t)
	setupTemplateSubscription(t, c, ctx, consumer, eventName, srv.URL,
		`amount={{.payload.amount}}`, map[string]any{"template_missing_key": "zero"})

	eventID := pushPayload(t, c, ctx, consumer, eventName, map[string]any{"invoice_id": "inv-3"})
	waitForDelivery(t, c, ctx, consumer, eventID, func(d templateDelivery) bool { return d.Status == "success" })
	bodies := rec.all()
	require.Len(t, bodies, 1)
	assert.Equal(t, "amount=<no value>", string(bodies[0]))
}

func TestTemplateErrors_InvalidSettingsRejected(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	srv, _ := startBodyRecorder(t)
	_, subID := setupTemplateSubscription(t, c, ctx, "tmpl-invalid", "invoice.created", srv.URL, `{}`, nil)
	resp, err := c.do(ctx, http.MethodPatch, "/v1/consumers/tmpl-invalid/subscriptions/"+subID,
		map[string]any{"on_transform_error": "ignore"}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

	var sub struct {
		OnTransformError   string `json:"on_transform_error"`
		TemplateMissingKey string `json:"template_missing_key"`
	}
	_, err = c.get(ctx, "/v1/consumers/tmpl-invalid/subscriptions/"+subID, &sub)
	require.NoError(t, err)
	assert.Equal(t, "fail", sub.OnTransformError)
	assert.Equal(t, "error", sub.TemplateMissingKey)
}
