//go:build integration

package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturedWebhook holds the data received by the test webhook target server.
type capturedWebhook struct {
	Body    []byte
	Headers http.Header
}

// startWebhookTarget starts an httptest.Server that captures exactly one webhook
// delivery. It returns the server and a function that blocks until a delivery is
// received (or ctx expires).
func startWebhookTarget(t *testing.T) (*httptest.Server, func(ctx context.Context) capturedWebhook) {
	t.Helper()
	var mu sync.Mutex
	var captured *capturedWebhook
	received := make(chan struct{})
	var once sync.Once

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		captured = &capturedWebhook{Body: body, Headers: r.Header.Clone()}
		mu.Unlock()
		once.Do(func() { close(received) })
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	wait := func(ctx context.Context) capturedWebhook {
		select {
		case <-received:
		case <-ctx.Done():
			t.Fatal("timed out waiting for webhook delivery")
		}
		mu.Lock()
		defer mu.Unlock()
		return *captured
	}
	return srv, wait
}

// TestE2E_HappyPath exercises the full webhook delivery pipeline over REST:
//
//	Register event type -> Register webhook -> Create subscription
//	-> Push event -> River processes event -> Fans out delivery -> HTTP POST
//	-> Verify delivery status, headers, body, HMAC.
func TestE2E_HappyPath(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespaceName = "integration-test"
		eventName     = "order.created"
	)

	// ── Step 1: Register event type ──────────────────────────────────────
	t.Log("Step 1: Registering event type")
	var eventTypeOut struct {
		Name string `json:"name"`
	}
	resp, err := c.post(ctx, "/v1/event-types", map[string]any{
		"name":        eventName,
		"description": "Order created event for integration test",
		"active":      true,
	}, &eventTypeOut)
	require.NoError(t, err, "RegisterEvent failed")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	t.Logf("  event type registered: %s", eventTypeOut.Name)

	// ── Step 2: Start webhook target server ──────────────────────────────
	t.Log("Step 2: Starting webhook target server")
	targetSrv, waitForDelivery := startWebhookTarget(t)
	t.Logf("  webhook target listening at: %s", targetSrv.URL)

	// ── Step 3: Register webhook ─────────────────────────────────────────
	t.Log("Step 3: Registering webhook")
	var webhookOut struct {
		WebhookID  string `json:"webhook_id"`
		HTTPConfig struct {
			WebhookSecret string `json:"webhook_secret"`
		} `json:"http_config"`
	}
	resp, err = c.post(ctx, "/v1/namespaces/"+namespaceName+"/webhooks", map[string]any{
		"events":      []string{eventName},
		"url":         targetSrv.URL + "/webhook",
		"active":      true,
		"description": "Integration test webhook",
		"http_config": map[string]any{
			"max_retries":             3,
			"request_timeout_seconds": 10,
			"capture_response_body":   true,
		},
	}, &webhookOut)
	require.NoError(t, err, "RegisterWebhook failed")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	webhookID := webhookOut.WebhookID
	require.NotEmpty(t, webhookID)
	webhookSecret := webhookOut.HTTPConfig.WebhookSecret
	require.NotEmpty(t, webhookSecret, "registration response should return the generated webhook secret")
	t.Logf("  webhook registered: %s", webhookID)

	// ── Step 4: Verify webhook registration auto-created the subscription ─
	t.Log("Step 4: Verifying auto-created subscription")
	var subListOut struct {
		Items []struct {
			SubscriptionID string `json:"subscription_id"`
		} `json:"items"`
	}
	resp, err = c.get(ctx, "/v1/namespaces/"+namespaceName+"/subscriptions?webhook_id="+webhookID+"&event_name="+eventName, &subListOut)
	require.NoError(t, err, "ListSubscriptions failed")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, subListOut.Items, 1, "registering a webhook should auto-create exactly one subscription per listed event")
	subscriptionID := subListOut.Items[0].SubscriptionID
	require.NotEmpty(t, subscriptionID)
	t.Logf("  subscription auto-created: %s", subscriptionID)

	// ── Step 5: Push event ───────────────────────────────────────────────
	t.Log("Step 5: Pushing event")
	var pushOut struct {
		EventID string `json:"event_id"`
	}
	resp, err = c.post(ctx, "/v1/namespaces/"+namespaceName+"/events?event="+eventName, map[string]any{
		"payload": map[string]any{
			"order_id":    "ord_12345",
			"customer_id": "cust_67890",
			"amount":      99.99,
			"currency":    "USD",
			"items": []any{
				map[string]any{"sku": "SKU-001", "qty": 2},
			},
		},
		"ttl_seconds": 300,
		"metadata": map[string]string{
			"source": "integration_test",
		},
	}, &pushOut)
	require.NoError(t, err, "PushEvent failed")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	eventID := pushOut.EventID
	require.NotEmpty(t, eventID)
	t.Logf("  event pushed: %s", eventID)

	// ── Step 6: Wait for webhook delivery ────────────────────────────────
	t.Log("Step 6: Waiting for webhook delivery...")
	deliveryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	delivery := waitForDelivery(deliveryCtx)
	t.Log("  webhook delivery received!")

	// ── Step 7: Validate webhook body ────────────────────────────────────
	t.Log("Step 7: Validating webhook body")

	var envelope map[string]any
	err = json.Unmarshal(delivery.Body, &envelope)
	require.NoError(t, err, "failed to parse webhook body as JSON")

	assert.Equal(t, "1", envelope["version"], "envelope version should be '1'")
	assert.Equal(t, eventID, envelope["event_id"], "envelope event_id should match pushed event")
	assert.Equal(t, eventName, envelope["event_name"], "envelope event_name should match")
	assert.NotEmpty(t, envelope["timestamp"], "envelope timestamp should be set")
	assert.EqualValues(t, 1, envelope["attempt"], "first delivery attempt should be 1")

	payloadMap, ok := envelope["payload"].(map[string]any)
	require.True(t, ok, "envelope should contain 'payload' as object")
	assert.Equal(t, "ord_12345", payloadMap["order_id"])
	assert.Equal(t, "cust_67890", payloadMap["customer_id"])
	assert.InDelta(t, 99.99, payloadMap["amount"], 0.001)
	assert.Equal(t, "USD", payloadMap["currency"])

	_, hasNamespace := envelope["namespace"]
	_, hasWebhookID := envelope["webhook_id"]
	_, hasDeliveryID := envelope["delivery_id"]
	assert.False(t, hasNamespace, "body should NOT contain namespace (sent via headers)")
	assert.False(t, hasWebhookID, "body should NOT contain webhook_id (sent via headers)")
	assert.False(t, hasDeliveryID, "body should NOT contain delivery_id (sent via headers)")

	// ── Step 8: Validate webhook headers ─────────────────────────────────
	t.Log("Step 8: Validating webhook headers")

	sparrowEventID := delivery.Headers.Get("X-Sparrow-Event-ID")
	assert.Equal(t, eventID, sparrowEventID, "X-Sparrow-Event-ID header should match event ID")

	sparrowWebhookID := delivery.Headers.Get("X-Sparrow-Webhook-ID")
	assert.Equal(t, webhookID, sparrowWebhookID, "X-Sparrow-Webhook-ID header should match webhook ID")

	sparrowDeliveryID := delivery.Headers.Get("X-Sparrow-Delivery-ID")
	assert.NotEmpty(t, sparrowDeliveryID, "X-Sparrow-Delivery-ID header should be set")

	contentType := delivery.Headers.Get("Content-Type")
	assert.Equal(t, "application/json", contentType, "Content-Type should be application/json")

	userAgent := delivery.Headers.Get("User-Agent")
	assert.True(t, strings.HasPrefix(userAgent, "Sparrow-Webhook/"), "User-Agent should start with 'Sparrow-Webhook/'")

	webhookSig := delivery.Headers.Get("webhook-signature")
	assert.NotEmpty(t, webhookSig, "webhook-signature should be set (webhook has secret)")
	assert.True(t, strings.HasPrefix(webhookSig, "v1,"), "signature should start with 'v1,'")

	webhookTimestamp := delivery.Headers.Get("webhook-timestamp")
	assert.NotEmpty(t, webhookTimestamp, "webhook-timestamp should be set")

	webhookMsgID := delivery.Headers.Get("webhook-id")
	assert.NotEmpty(t, webhookMsgID, "webhook-id should be set")

	// ── Step 9: Validate HMAC signature ──────────────────────────────────
	t.Log("Step 9: Validating HMAC signature (Standard Webhooks)")
	validateHMAC(t, delivery.Body, webhookSecret, webhookMsgID, webhookTimestamp, webhookSig)

	// ── Step 10: Poll for delivery status via API ────────────────────────
	t.Log("Step 10: Polling for delivery status via API")
	pollDeliverySuccess(t, c, namespaceName, eventID, sparrowDeliveryID, 30*time.Second)

	t.Log("E2E happy path test passed!")
	_ = subscriptionID
}

// validateHMAC verifies the HMAC-SHA256 signature per Standard Webhooks spec.
// Signature format: "v1,<base64>" (may have additional space-delimited signatures).
// Message to sign: "{msgID}.{timestamp}.{body}"
func validateHMAC(t *testing.T, body []byte, secret, msgID, timestamp, signatureHeader string) {
	t.Helper()

	var b64Sig string
	for _, s := range strings.Split(signatureHeader, " ") {
		if strings.HasPrefix(s, "v1,") {
			b64Sig = strings.TrimPrefix(s, "v1,")
			break
		}
	}
	require.NotEmpty(t, b64Sig, "should find a v1 signature in webhook-signature header")

	rawSecret := []byte(secret)
	if strings.HasPrefix(secret, "whsec_") {
		decoded, decErr := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
		require.NoError(t, decErr, "failed to decode whsec_ secret")
		rawSecret = decoded
	}
	message := msgID + "." + timestamp + "." + string(body)
	mac := hmac.New(sha256.New, rawSecret)
	mac.Write([]byte(message))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	assert.Equal(t, expected, b64Sig, "HMAC signature mismatch")
}

// pollDeliverySuccess polls the ListDeliveries API until the given delivery
// reaches "success" status or the timeout expires.
func pollDeliverySuccess(t *testing.T, c *restClient, namespace, eventID, deliveryID string, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	type deliveryOut struct {
		DeliveryID    string `json:"delivery_id"`
		Status        string `json:"status"`
		AttemptCount  int    `json:"attempt_count"`
		ResponseCode  int    `json:"response_code"`
		ErrorMessage  string `json:"error_message"`
		ErrorCategory string `json:"error_category"`
	}
	type listOut struct {
		Items []deliveryOut `json:"items"`
	}

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for delivery %s to reach success status", deliveryID)
		case <-ticker.C:
			reqCtx, reqCancel := context.WithTimeout(ctx, 10*time.Second)
			var out listOut
			_, err := c.get(reqCtx, "/v1/namespaces/"+namespace+"/deliveries?event_id="+eventID, &out)
			reqCancel()
			if err != nil {
				t.Logf("  poll: ListDeliveries error (retrying): %v", err)
				continue
			}

			for _, d := range out.Items {
				if d.DeliveryID == deliveryID {
					t.Logf("  poll: delivery %s status=%s attempts=%d responseCode=%d",
						deliveryID, d.Status, d.AttemptCount, d.ResponseCode)

					if d.Status == "success" {
						assert.Equal(t, 200, d.ResponseCode, "response code should be 200")
						assert.Equal(t, 1, d.AttemptCount, "should succeed on first attempt")
						return
					}
					if d.Status == "failed" {
						t.Fatalf("delivery %s failed: %s (category: %s)",
							deliveryID, d.ErrorMessage, d.ErrorCategory)
					}
				}
			}
		}
	}
}
