//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// raw sends a request as c and returns the status and the body as text.
func raw(t *testing.T, c *restClient, method, path string, body any) (int, string) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, c.baseURL+path, reader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck
	text, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(text)
}

// TestAPITenantIsolationAcrossResources has tenant B create one of every
// resource Sparrow's API exposes, then has the default tenant (as every
// self-hosted request is) read, list, change, delete, retry and re-push each
// of them by id. Every by-id read must be a 404, no list or export may
// mention any of B's ids or names, every write must fail, and B's resources
// and receiver must be untouched afterwards.
func TestAPITenantIsolationAcrossResources(t *testing.T) {
	ctx := context.Background()
	sp := startEmbeddedSparrow(t)
	tenantB := sp.addTenant(t, "tenant-b")
	b := sp.client(t, tenantB)
	a := sp.client(t, uuid.Nil)

	const consumer = "secret-consumer"
	const eventType = "invoice.paid"
	rcv := newCountingReceiver(t)

	// ── Tenant B's resources ────────────────────────────────────────────
	resp, err := b.post(ctx, "/v1/event-types", map[string]any{"name": eventType, "description": "original", "active": true}, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var wh struct {
		WebhookID string `json:"webhook_id"`
	}
	resp, err = b.post(ctx, "/v1/consumers/"+consumer+"/webhooks", map[string]any{
		"events": []string{eventType}, "url": rcv.srv.URL + "/hook-b", "active": true, "description": "b-webhook",
	}, &wh)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var subs struct {
		Items []struct {
			SubscriptionID string `json:"subscription_id"`
		} `json:"items"`
	}
	_, err = b.get(ctx, "/v1/consumers/"+consumer+"/subscriptions", &subs)
	require.NoError(t, err)
	require.Len(t, subs.Items, 1)
	subID := subs.Items[0].SubscriptionID

	var alert struct {
		ID string `json:"id"`
	}
	resp, err = b.post(ctx, "/v1/consumers/"+consumer+"/alert-configs", map[string]any{
		"email": "b-ops@example.com", "event_types": []string{"sparrow.webhook.disabled"},
	}, &alert)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var ev struct {
		EventID string `json:"event_id"`
	}
	resp, err = b.post(ctx, "/v1/consumers/"+consumer+"/events?event="+eventType, map[string]any{"payload": map[string]any{"n": 1}}, &ev)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	rcv.waitFor(t, 1)

	var deliveries struct {
		Items []struct {
			DeliveryID string `json:"delivery_id"`
		} `json:"items"`
		RetryID string `json:"retry_id"`
	}
	require.Eventually(t, func() bool {
		_, err := b.get(ctx, "/v1/consumers/"+consumer+"/deliveries?prepare_retry=true", &deliveries)
		require.NoError(t, err)
		return len(deliveries.Items) == 1 && deliveries.RetryID != ""
	}, 15*time.Second, 200*time.Millisecond)
	deliveryID, retryID := deliveries.Items[0].DeliveryID, deliveries.RetryID

	var events struct {
		RepushID string `json:"repush_id"`
	}
	_, err = b.get(ctx, "/v1/consumers/"+consumer+"/events?prepare_repush=true", &events)
	require.NoError(t, err)
	require.NotEmpty(t, events.RepushID)
	repushID := events.RepushID

	// Nothing the other tenant sees may mention any of these.
	secrets := []string{wh.WebhookID, subID, alert.ID, ev.EventID, deliveryID, retryID, repushID, "/hook-b", "b-webhook", "b-ops@example.com"}
	globalSecrets := append([]string{consumer, eventType}, secrets...)
	// noLeak fails when body mentions one of tokens that the request itself
	// did not carry: echoing the id a caller sent tells it nothing new.
	noLeak := func(t *testing.T, request, body string, tokens []string) {
		t.Helper()
		for _, s := range tokens {
			if strings.Contains(request, s) {
				continue
			}
			require.NotContains(t, body, s, "%s leaked tenant B's %q", request, s)
		}
	}
	asText := func(path string, body any) string {
		b, _ := json.Marshal(body)
		return path + " " + string(b)
	}

	// ── Reads by id: all 404, nothing leaked ────────────────────────────
	c := "/v1/consumers/" + consumer
	for _, path := range []string{
		c + "/webhooks/" + wh.WebhookID,
		c + "/webhooks/" + wh.WebhookID + "/health",
		c + "/subscriptions/" + subID,
		c + "/subscriptions/" + subID + "/templateVersions",
		c + "/deliveries/" + deliveryID,
		c + "/deliveries/" + deliveryID + "/attempts",
		c + "/retry-jobs/" + retryID,
		c + "/repush-jobs/" + repushID,
		"/v1/deliveries/" + deliveryID,
		"/v1/events/" + ev.EventID,
		"/v1/event-types/" + eventType,
		"/v1/event-types/" + eventType + "/versions/1",
	} {
		t.Run("GET "+path, func(t *testing.T) {
			status, body := raw(t, a, http.MethodGet, path, nil)
			require.Equal(t, http.StatusNotFound, status, "body: %s", body)
			noLeak(t, asText(path, nil), body, secrets)
		})
	}

	// The global attempts endpoint answers an unknown delivery with an empty
	// list rather than 404. B's delivery must get exactly the answer an id
	// that does not exist gets, so it reveals nothing.
	t.Run("GET /v1/deliveries/{id}/attempts", func(t *testing.T) {
		status, body := raw(t, a, http.MethodGet, "/v1/deliveries/"+deliveryID+"/attempts", nil)
		unknownStatus, unknownBody := raw(t, a, http.MethodGet, "/v1/deliveries/"+uuid.NewString()+"/attempts", nil)
		require.Equal(t, unknownStatus, status)
		require.JSONEq(t, unknownBody, body)
	})

	// ── Lists and exports: nothing of B's in them ───────────────────────
	for _, tc := range []struct {
		method, path string
		body         any
		tokens       []string
	}{
		{http.MethodGet, "/v1/consumers", nil, globalSecrets},
		{http.MethodGet, "/v1/stats", nil, globalSecrets},
		{http.MethodGet, "/v1/webhooks", nil, globalSecrets},
		{http.MethodGet, "/v1/subscriptions", nil, globalSecrets},
		{http.MethodGet, "/v1/events", nil, globalSecrets},
		{http.MethodGet, "/v1/deliveries", nil, globalSecrets},
		{http.MethodGet, "/v1/health-summary", nil, globalSecrets},
		{http.MethodGet, "/v1/event-types", nil, globalSecrets},
		{http.MethodGet, "/v1/event-types/" + eventType + "/versions", nil, secrets},
		{http.MethodPost, "/v1/event-types:export", map[string]any{"all": true}, globalSecrets},
		{http.MethodGet, c + "/stats", nil, secrets},
		{http.MethodGet, c + "/webhooks", nil, secrets},
		{http.MethodGet, c + "/subscriptions", nil, secrets},
		{http.MethodGet, c + "/deliveries", nil, secrets},
		{http.MethodGet, c + "/events", nil, secrets},
		{http.MethodGet, c + "/alert-configs", nil, secrets},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			status, body := raw(t, a, tc.method, tc.path, tc.body)
			require.Less(t, status, 500, "body: %s", body)
			noLeak(t, asText(tc.path, tc.body), body, tc.tokens)
		})
	}

	// ── Writes: every one fails ─────────────────────────────────────────
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPatch, c + "/webhooks/" + wh.WebhookID, map[string]any{"description": "hacked"}},
		{http.MethodPost, c + "/webhooks/" + wh.WebhookID + ":pause", nil},
		{http.MethodPost, c + "/webhooks/" + wh.WebhookID + ":resume", nil},
		{http.MethodPost, c + "/webhooks/" + wh.WebhookID + ":rotateSecret", nil},
		{http.MethodPatch, c + "/subscriptions/" + subID, map[string]any{"transform_enabled": false}},
		{http.MethodPost, c + "/subscriptions/" + subID + ":pause", nil},
		{http.MethodPost, c + "/subscriptions/" + subID + ":resume", nil},
		{http.MethodPost, c + "/subscriptions/" + subID + ":sendTestEvent", nil},
		{http.MethodPost, c + "/deliveries/" + deliveryID + ":retry", nil},
		{http.MethodPost, "/v1/deliveries/" + deliveryID + ":retry", nil},
		{http.MethodPost, c + "/deliveries:retryBatch", map[string]any{"retry_id": retryID}},
		{http.MethodPost, c + "/retry-jobs/" + retryID + ":cancel", nil},
		{http.MethodPost, c + "/events:rePush", map[string]any{"repush_id": repushID}},
		{http.MethodPost, c + "/repush-jobs/" + repushID + ":cancel", nil},
		{http.MethodPost, "/v1/events/" + ev.EventID + ":repush", nil},
		{http.MethodPatch, "/v1/event-types/" + eventType, map[string]any{"description": "hacked"}},
		{http.MethodDelete, c + "/alert-configs/" + alert.ID, nil},
		{http.MethodDelete, c + "/subscriptions/" + subID, nil},
		{http.MethodDelete, c + "/webhooks/" + wh.WebhookID, nil},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			status, body := raw(t, a, tc.method, tc.path, tc.body)
			require.GreaterOrEqual(t, status, 400, "the other tenant's write succeeded: %s", body)
			require.Less(t, status, 500, "body: %s", body)
			noLeak(t, asText(tc.path, tc.body), body, secrets)
		})
	}

	// ── Tenant B is untouched ───────────────────────────────────────────
	status, body := raw(t, b, http.MethodGet, c+"/webhooks/"+wh.WebhookID, nil)
	require.Equal(t, http.StatusOK, status)
	require.Contains(t, body, `"description":"b-webhook"`)
	require.Contains(t, body, `"active":true`)

	status, _ = raw(t, b, http.MethodGet, c+"/subscriptions/"+subID, nil)
	require.Equal(t, http.StatusOK, status)

	status, body = raw(t, b, http.MethodGet, c+"/alert-configs", nil)
	require.Equal(t, http.StatusOK, status)
	require.Contains(t, body, alert.ID)

	status, body = raw(t, b, http.MethodGet, "/v1/event-types/"+eventType, nil)
	require.Equal(t, http.StatusOK, status)
	require.Contains(t, body, `"description":"original"`)

	for _, job := range []string{c + "/retry-jobs/" + retryID, c + "/repush-jobs/" + repushID} {
		status, body = raw(t, b, http.MethodGet, job, nil)
		require.Equal(t, http.StatusOK, status)
		require.True(t, strings.Contains(body, `"status":"pending"`), "%s was started or cancelled by the other tenant: %s", job, body)
	}

	time.Sleep(2 * time.Second) // let any wrongly queued retry, test event or re-push land
	require.EqualValues(t, 1, rcv.count.Load(), "the other tenant made B's receiver get more deliveries")
}
