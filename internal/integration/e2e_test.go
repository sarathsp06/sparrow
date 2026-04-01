//go:build integration

package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/sarathsp06/sparrow/proto"
	pbconnect "github.com/sarathsp06/sparrow/proto/protoconnect"
)

// capturedWebhook holds the data received by the test webhook target server.
type capturedWebhook struct {
	Headers http.Header
	Body    []byte
}

// startWebhookTarget starts an httptest.Server that captures exactly one webhook
// delivery. It returns the server and a function that blocks until a delivery is
// received (or ctx expires).
func startWebhookTarget(t *testing.T) (*httptest.Server, func(ctx context.Context) capturedWebhook) {
	t.Helper()

	var (
		mu       sync.Mutex
		captured *capturedWebhook
		done     = make(chan struct{})
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Logf("webhook target: failed to read body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		mu.Lock()
		defer mu.Unlock()
		if captured == nil {
			captured = &capturedWebhook{
				Headers: r.Header.Clone(),
				Body:    body,
			}
			close(done)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	wait := func(ctx context.Context) capturedWebhook {
		t.Helper()
		select {
		case <-done:
			mu.Lock()
			defer mu.Unlock()
			return *captured
		case <-ctx.Done():
			t.Fatal("timed out waiting for webhook delivery")
			return capturedWebhook{} // unreachable
		}
	}

	return srv, wait
}

// newClients creates Connect-RPC Go clients pointed at the test environment's
// HTTP server, with the root API key set as the auth header.
type testClients struct {
	webhook      pbconnect.WebhookServiceClient
	event        pbconnect.EventServiceClient
	subscription pbconnect.SubscriptionServiceClient
	delivery     pbconnect.DeliveryServiceClient
}

func newClients(env *testEnv) *testClients {
	return &testClients{
		webhook:      pbconnect.NewWebhookServiceClient(http.DefaultClient, env.baseURL),
		event:        pbconnect.NewEventServiceClient(http.DefaultClient, env.baseURL),
		subscription: pbconnect.NewSubscriptionServiceClient(http.DefaultClient, env.baseURL),
		delivery:     pbconnect.NewDeliveryServiceClient(http.DefaultClient, env.baseURL),
	}
}

// TestE2E_HappyPath exercises the full webhook delivery pipeline:
//
//	Create namespace -> Register event -> Register webhook -> Create subscription
//	-> Push event -> River processes event -> Fans out delivery -> HTTP POST
//	-> Verify delivery status, headers, body, HMAC.
func TestE2E_HappyPath(t *testing.T) {
	env := setupEnv(t)
	clients := newClients(env)
	ctx := context.Background()

	const (
		namespaceName = "integration-test"
		eventName     = "order.created"
		webhookSecret = "test-secret-key-for-hmac"
	)

	// Namespace is now a free-form string — no need to create it via API.
	// ── Step 2: Register event type ──────────────────────────────────────
	t.Log("Step 2: Registering event type")
	eventResp, err := clients.event.RegisterEvent(ctx, connect.NewRequest(&pb.RegisterEventRequest{
		Name:        eventName,
		Description: "Order created event for integration test",
		Active:      true,
	}))
	require.NoError(t, err, "RegisterEvent failed")
	t.Logf("  event registered: %s", eventResp.Msg.GetEventId())

	// ── Step 3: Start webhook target server ──────────────────────────────
	t.Log("Step 3: Starting webhook target server")
	targetSrv, waitForDelivery := startWebhookTarget(t)
	t.Logf("  webhook target listening at: %s", targetSrv.URL)

	// ── Step 4: Register webhook ─────────────────────────────────────────
	t.Log("Step 4: Registering webhook")
	webhookResp, err := clients.webhook.RegisterWebhook(ctx, connect.NewRequest(&pb.RegisterWebhookRequest{
		Namespace:   namespaceName,
		Url:         targetSrv.URL + "/webhook",
		Active:      true,
		Description: "Integration test webhook",
		HttpConfig: &pb.WebhookHTTPConfig{
			MaxRetries:            3,
			RequestTimeoutSeconds: 10,
			WebhookSecret:         webhookSecret,
			CaptureResponseBody:   true,
		},
	}))
	require.NoError(t, err, "RegisterWebhook failed")
	webhookID := webhookResp.Msg.GetWebhookId()
	require.NotEmpty(t, webhookID)
	t.Logf("  webhook registered: %s", webhookID)

	// ── Step 5: Create subscription ──────────────────────────────────────
	t.Log("Step 5: Creating subscription")
	subResp, err := clients.subscription.CreateSubscription(ctx, connect.NewRequest(&pb.CreateSubscriptionRequest{
		WebhookId: webhookID,
		EventName: eventName,
		Namespace: namespaceName,
	}))
	require.NoError(t, err, "CreateSubscription failed")
	subscriptionID := subResp.Msg.GetSubscriptionId()
	require.NotEmpty(t, subscriptionID)
	t.Logf("  subscription created: %s", subscriptionID)

	// ── Step 6: Push event ───────────────────────────────────────────────
	t.Log("Step 6: Pushing event")
	eventPayload, err := structpb.NewStruct(map[string]any{
		"order_id":    "ord_12345",
		"customer_id": "cust_67890",
		"amount":      99.99,
		"currency":    "USD",
		"items": []any{
			map[string]any{"sku": "SKU-001", "qty": float64(2)},
		},
	})
	require.NoError(t, err, "failed to create event payload struct")

	pushResp, err := clients.event.PushEvent(ctx, connect.NewRequest(&pb.PushEventRequest{
		Namespace:  namespaceName,
		Event:      eventName,
		Payload:    eventPayload,
		TtlSeconds: 300,
		Metadata: map[string]string{
			"source": "integration_test",
		},
	}))
	require.NoError(t, err, "PushEvent failed")
	eventID := pushResp.Msg.GetEventId()
	require.NotEmpty(t, eventID)
	t.Logf("  event pushed: %s", eventID)

	// ── Step 7: Wait for webhook delivery ────────────────────────────────
	t.Log("Step 7: Waiting for webhook delivery...")
	deliveryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	delivery := waitForDelivery(deliveryCtx)
	t.Log("  webhook delivery received!")

	// ── Step 8: Validate webhook body ────────────────────────────────────
	t.Log("Step 8: Validating webhook body")

	var envelope map[string]any
	err = json.Unmarshal(delivery.Body, &envelope)
	require.NoError(t, err, "failed to parse webhook body as JSON")

	// Check envelope fields (snake_case)
	assert.Equal(t, "1", envelope["version"], "envelope version should be '1'")
	assert.Equal(t, eventID, envelope["event_id"], "envelope event_id should match pushed event")
	assert.Equal(t, eventName, envelope["event_name"], "envelope event_name should match")
	assert.NotEmpty(t, envelope["timestamp"], "envelope timestamp should be set")
	assert.EqualValues(t, 1, envelope["attempt"], "first delivery attempt should be 1")

	// Check payload is nested under "payload" key
	payloadMap, ok := envelope["payload"].(map[string]any)
	require.True(t, ok, "envelope should contain 'payload' as object")
	assert.Equal(t, "ord_12345", payloadMap["order_id"])
	assert.Equal(t, "cust_67890", payloadMap["customer_id"])
	assert.InDelta(t, 99.99, payloadMap["amount"], 0.001)
	assert.Equal(t, "USD", payloadMap["currency"])

	// Verify body does NOT contain namespace, webhook_id, or delivery_id at top level
	_, hasNamespace := envelope["namespace"]
	_, hasWebhookID := envelope["webhook_id"]
	_, hasDeliveryID := envelope["delivery_id"]
	assert.False(t, hasNamespace, "body should NOT contain namespace (sent via headers)")
	assert.False(t, hasWebhookID, "body should NOT contain webhook_id (sent via headers)")
	assert.False(t, hasDeliveryID, "body should NOT contain delivery_id (sent via headers)")

	// ── Step 9: Validate webhook headers ─────────────────────────────────
	t.Log("Step 9: Validating webhook headers")

	// X-Sparrow-Event-ID
	sparrowEventID := delivery.Headers.Get("X-Sparrow-Event-ID")
	assert.Equal(t, eventID, sparrowEventID, "X-Sparrow-Event-ID header should match event ID")

	// X-Sparrow-Webhook-ID
	sparrowWebhookID := delivery.Headers.Get("X-Sparrow-Webhook-ID")
	assert.Equal(t, webhookID, sparrowWebhookID, "X-Sparrow-Webhook-ID header should match webhook ID")

	// X-Sparrow-Delivery-ID
	sparrowDeliveryID := delivery.Headers.Get("X-Sparrow-Delivery-ID")
	assert.NotEmpty(t, sparrowDeliveryID, "X-Sparrow-Delivery-ID header should be set")

	// Content-Type
	contentType := delivery.Headers.Get("Content-Type")
	assert.Equal(t, "application/json", contentType, "Content-Type should be application/json")

	// User-Agent should contain Sparrow
	userAgent := delivery.Headers.Get("User-Agent")
	assert.True(t, strings.HasPrefix(userAgent, "Sparrow-Webhook/"), "User-Agent should start with 'Sparrow-Webhook/'")

	// HMAC signature (webhook has a secret)
	signature := delivery.Headers.Get("X-Sparrow-Signature-256")
	assert.NotEmpty(t, signature, "X-Sparrow-Signature-256 should be set (webhook has secret)")
	assert.True(t, strings.HasPrefix(signature, "sha256="), "signature should start with 'sha256='")

	timestamp := delivery.Headers.Get("X-Sparrow-Timestamp")
	assert.NotEmpty(t, timestamp, "X-Sparrow-Timestamp should be set")

	// ── Step 10: Validate HMAC signature ─────────────────────────────────
	t.Log("Step 10: Validating HMAC signature")
	validateHMAC(t, delivery.Body, webhookSecret, timestamp, signature)

	// ── Step 11: Poll for delivery status via API ────────────────────────
	t.Log("Step 11: Polling for delivery status via API")
	pollDeliverySuccess(t, clients, namespaceName, eventID, sparrowDeliveryID, 30*time.Second)

	t.Log("E2E happy path test passed!")
}

// validateHMAC verifies the HMAC-SHA256 signature matches the expected value.
func validateHMAC(t *testing.T, body []byte, secret, timestamp, signatureHeader string) {
	t.Helper()

	// Signature format: "sha256=<hex>"
	hexSig := strings.TrimPrefix(signatureHeader, "sha256=")
	require.NotEqual(t, signatureHeader, hexSig, "signature should have 'sha256=' prefix")

	message := timestamp + "." + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	expected := hex.EncodeToString(mac.Sum(nil))

	assert.Equal(t, expected, hexSig, "HMAC signature mismatch")
}

// pollDeliverySuccess polls the ListDeliveries API until the given delivery
// reaches "success" status or the timeout expires.
func pollDeliverySuccess(t *testing.T, clients *testClients, namespace, eventID, deliveryID string, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for delivery %s to reach success status", deliveryID)
		case <-ticker.C:
			resp, err := clients.delivery.ListDeliveries(ctx, connect.NewRequest(&pb.ListDeliveriesRequest{
				Namespace: namespace,
				EventId:   eventID,
			}))
			if err != nil {
				t.Logf("  poll: ListDeliveries error (retrying): %v", err)
				continue
			}

			for _, d := range resp.Msg.GetDeliveries() {
				if d.GetDeliveryId() == deliveryID {
					t.Logf("  poll: delivery %s status=%s attempts=%d responseCode=%d",
						deliveryID, d.GetStatus(), d.GetAttemptCount(), d.GetResponseCode())

					if d.GetStatus() == pb.WebhookDeliveryStatus_DELIVERY_SUCCESS {
						assert.EqualValues(t, 200, d.GetResponseCode(), "response code should be 200")
						assert.EqualValues(t, 1, d.GetAttemptCount(), "should succeed on first attempt")
						return
					}
					if d.GetStatus() == pb.WebhookDeliveryStatus_DELIVERY_FAILED {
						t.Fatalf("delivery %s failed: %s (category: %s)",
							deliveryID, d.GetErrorMessage(), d.GetErrorCategory())
					}
				}
			}
		}
	}
}
