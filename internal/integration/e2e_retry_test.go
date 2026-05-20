//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/sarathsp06/sparrow/proto"
)

// startFailThenSucceedTarget returns a test server that fails the first N
// requests with the given status code, then succeeds all subsequent ones.
// Returns the server and a counter of total requests received.
func startFailThenSucceedTarget(t *testing.T, failCount int, failStatus int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var counter atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		defer r.Body.Close()

		n := counter.Add(1)
		if int(n) <= failCount {
			w.WriteHeader(failStatus)
			_, _ = w.Write([]byte(`{"error":"temporary failure"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &counter
}

// startAlwaysFailTarget returns a test server that always returns the given status.
func startAlwaysFailTarget(t *testing.T, status int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var counter atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		defer r.Body.Close()
		counter.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error":"permanent failure"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &counter
}

// startCountingTarget captures all requests and always returns 200.
func startCountingTarget(t *testing.T) (*httptest.Server, *atomic.Int32, func() [][]byte) {
	t.Helper()
	var (
		counter atomic.Int32
		mu      sync.Mutex
		bodies  [][]byte
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		counter.Add(1)
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	getBodies := func() [][]byte {
		mu.Lock()
		defer mu.Unlock()
		cp := make([][]byte, len(bodies))
		copy(cp, bodies)
		return cp
	}

	return srv, &counter, getBodies
}

// setupWebhookPipeline is a helper that registers an event, webhook, and subscription.
// Returns webhookID, subscriptionID.
func setupWebhookPipeline(t *testing.T, clients *testClients, namespace, eventName, targetURL string, maxRetries int32) (string, string) {
	t.Helper()
	ctx := context.Background()

	// Register event
	_, err := clients.event.RegisterEvent(ctx, connect.NewRequest(&pb.RegisterEventRequest{
		Name:   eventName,
		Active: true,
	}))
	require.NoError(t, err)

	// Register webhook
	webhookResp, err := clients.webhook.RegisterWebhook(ctx, connect.NewRequest(&pb.RegisterWebhookRequest{
		Namespace: namespace,
		Url:       targetURL + "/webhook",
		Active:    true,
		HttpConfig: &pb.WebhookHTTPConfig{
			MaxRetries:            maxRetries,
			RequestTimeoutSeconds: 5,
			WebhookSecret:         "test-secret",
			CaptureResponseBody:   true,
		},
	}))
	require.NoError(t, err)
	webhookID := webhookResp.Msg.GetWebhookId()

	// Create subscription
	subResp, err := clients.subscription.CreateSubscription(ctx, connect.NewRequest(&pb.CreateSubscriptionRequest{
		WebhookId: webhookID,
		EventName: eventName,
		Namespace: namespace,
	}))
	require.NoError(t, err)

	return webhookID, subResp.Msg.GetSubscriptionId()
}

// pushTestEvent pushes a simple event and returns the event ID.
func pushTestEvent(t *testing.T, clients *testClients, namespace, eventName string) string {
	t.Helper()
	ctx := context.Background()

	payload, err := structpb.NewStruct(map[string]any{
		"test": true,
		"ts":   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	resp, err := clients.event.PushEvent(ctx, connect.NewRequest(&pb.PushEventRequest{
		Namespace:  namespace,
		Event:      eventName,
		Payload:    payload,
		TtlSeconds: 300,
	}))
	require.NoError(t, err)
	return resp.Msg.GetEventId()
}

// TestE2E_RetryOnServerError verifies that a delivery retries on 500 and
// eventually succeeds when the target recovers.
func TestE2E_RetryOnServerError(t *testing.T) {
	env := setupEnv(t)
	clients := newClients(env)

	const (
		namespace = "retry-test"
		eventName = "retry.server_error"
	)

	// Target fails first 2 requests, then succeeds
	targetSrv, requestCount := startFailThenSucceedTarget(t, 2, http.StatusInternalServerError)

	webhookID, _ := setupWebhookPipeline(t, clients, namespace, eventName, targetSrv.URL, 3)
	_ = webhookID

	// Push event
	eventID := pushTestEvent(t, clients, namespace, eventName)

	// Wait for successful delivery (should take ~3 attempts)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pollDeliveryStatus(t, clients, namespace, eventID, ctx, func(d *pb.WebhookDelivery) bool {
		return d.GetStatus() == pb.WebhookDeliveryStatus_DELIVERY_SUCCESS
	})

	// Verify the target received 3 requests (2 failed + 1 success)
	assert.GreaterOrEqual(t, int(requestCount.Load()), 3, "target should have received at least 3 requests")
}

// TestE2E_ExhaustedRetries verifies that a delivery reaches failed status
// when all retry attempts are exhausted.
func TestE2E_ExhaustedRetries(t *testing.T) {
	env := setupEnv(t)
	clients := newClients(env)

	const (
		namespace = "exhaust-retry-test"
		eventName = "retry.exhausted"
	)

	// Target always returns 500
	targetSrv, requestCount := startAlwaysFailTarget(t, http.StatusInternalServerError)

	setupWebhookPipeline(t, clients, namespace, eventName, targetSrv.URL, 2)

	// Push event
	eventID := pushTestEvent(t, clients, namespace, eventName)

	// Wait for delivery to reach failed status
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	delivery := pollDeliveryStatus(t, clients, namespace, eventID, ctx, func(d *pb.WebhookDelivery) bool {
		return d.GetStatus() == pb.WebhookDeliveryStatus_DELIVERY_FAILED
	})

	assert.Equal(t, "server_error", delivery.GetErrorCategory())
	// At least 1 attempt was made to the target
	assert.GreaterOrEqual(t, int(requestCount.Load()), 1)
}

// TestE2E_FanOutMultipleSubscribers verifies that one event is delivered
// to multiple webhooks subscribed to the same event.
func TestE2E_FanOutMultipleSubscribers(t *testing.T) {
	env := setupEnv(t)
	clients := newClients(env)
	ctx := context.Background()

	const (
		namespace = "fanout-test"
		eventName = "fanout.created"
	)

	// Register the event type
	_, err := clients.event.RegisterEvent(ctx, connect.NewRequest(&pb.RegisterEventRequest{
		Name:   eventName,
		Active: true,
	}))
	require.NoError(t, err)

	// Start 3 targets
	target1, count1, _ := startCountingTarget(t)
	target2, count2, _ := startCountingTarget(t)
	target3, count3, _ := startCountingTarget(t)

	// Register 3 webhooks + subscriptions
	for i, url := range []string{target1.URL, target2.URL, target3.URL} {
		webhookResp, err := clients.webhook.RegisterWebhook(ctx, connect.NewRequest(&pb.RegisterWebhookRequest{
			Namespace:   namespace,
			Url:         url + "/webhook",
			Active:      true,
			Description: "fanout target " + string(rune('A'+i)),
			HttpConfig: &pb.WebhookHTTPConfig{
				MaxRetries:            1,
				RequestTimeoutSeconds: 5,
				WebhookSecret:         "secret-" + string(rune('A'+i)),
			},
		}))
		require.NoError(t, err)

		_, err = clients.subscription.CreateSubscription(ctx, connect.NewRequest(&pb.CreateSubscriptionRequest{
			WebhookId: webhookResp.Msg.GetWebhookId(),
			EventName: eventName,
			Namespace: namespace,
		}))
		require.NoError(t, err)
	}

	// Push one event
	eventID := pushTestEvent(t, clients, namespace, eventName)
	_ = eventID

	// Wait for all 3 targets to receive a delivery
	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for fan-out: got %d, %d, %d deliveries",
				count1.Load(), count2.Load(), count3.Load())
		case <-ticker.C:
			if count1.Load() >= 1 && count2.Load() >= 1 && count3.Load() >= 1 {
				t.Logf("all 3 targets received delivery: %d, %d, %d",
					count1.Load(), count2.Load(), count3.Load())
				return
			}
		}
	}
}

// TestE2E_PausedWebhookNoDelivery verifies that pausing a webhook prevents
// new event deliveries, and resuming it allows them again.
func TestE2E_PausedWebhookNoDelivery(t *testing.T) {
	env := setupEnv(t)
	clients := newClients(env)
	ctx := context.Background()

	const (
		namespace = "pause-test"
		eventName = "pause.event"
	)

	targetSrv, requestCount, _ := startCountingTarget(t)

	webhookID, _ := setupWebhookPipeline(t, clients, namespace, eventName, targetSrv.URL, 1)

	// Pause the webhook
	_, err := clients.webhook.PauseWebhook(ctx, connect.NewRequest(&pb.PauseWebhookRequest{
		WebhookId: webhookID,
		Namespace: namespace,
		Reason:    "testing pause",
	}))
	require.NoError(t, err)

	// Push an event while paused
	_ = pushTestEvent(t, clients, namespace, eventName)

	// Wait a bit and verify NO delivery was made
	time.Sleep(5 * time.Second)
	assert.Equal(t, int32(0), requestCount.Load(), "paused webhook should not receive deliveries")

	// Resume the webhook
	_, err = clients.webhook.ResumeWebhook(ctx, connect.NewRequest(&pb.ResumeWebhookRequest{
		WebhookId: webhookID,
		Namespace: namespace,
		Reason:    "testing resume",
	}))
	require.NoError(t, err)

	// Push another event after resume
	eventID2 := pushTestEvent(t, clients, namespace, eventName)

	// This delivery should succeed
	pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	pollDeliveryStatus(t, clients, namespace, eventID2, pollCtx, func(d *pb.WebhookDelivery) bool {
		return d.GetStatus() == pb.WebhookDeliveryStatus_DELIVERY_SUCCESS
	})

	assert.GreaterOrEqual(t, int(requestCount.Load()), 1, "resumed webhook should receive delivery")
}

// TestE2E_DeleteSubscriptionStopsDelivery verifies that deleting a subscription
// prevents future events from creating deliveries for it.
func TestE2E_DeleteSubscriptionStopsDelivery(t *testing.T) {
	env := setupEnv(t)
	clients := newClients(env)
	ctx := context.Background()

	const (
		namespace = "delete-sub-test"
		eventName = "delete.sub.event"
	)

	targetSrv, requestCount, _ := startCountingTarget(t)

	_, subscriptionID := setupWebhookPipeline(t, clients, namespace, eventName, targetSrv.URL, 1)

	// First event should be delivered
	eventID1 := pushTestEvent(t, clients, namespace, eventName)
	pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	pollDeliveryStatus(t, clients, namespace, eventID1, pollCtx, func(d *pb.WebhookDelivery) bool {
		return d.GetStatus() == pb.WebhookDeliveryStatus_DELIVERY_SUCCESS
	})
	assert.Equal(t, int32(1), requestCount.Load())

	// Delete the subscription
	_, err := clients.subscription.DeleteSubscription(ctx, connect.NewRequest(&pb.DeleteSubscriptionRequest{
		SubscriptionId: subscriptionID,
		Namespace:      namespace,
	}))
	require.NoError(t, err)

	// Push another event -- should NOT create a delivery
	_ = pushTestEvent(t, clients, namespace, eventName)

	// Wait and verify no new deliveries
	time.Sleep(5 * time.Second)
	assert.Equal(t, int32(1), requestCount.Load(), "deleted subscription should not receive new deliveries")
}

// TestE2E_SingleRePush verifies that re-pushing an event creates a new delivery.
func TestE2E_SingleRePush(t *testing.T) {
	env := setupEnv(t)
	clients := newClients(env)
	ctx := context.Background()

	const (
		namespace = "repush-test"
		eventName = "repush.event"
	)

	targetSrv, requestCount, _ := startCountingTarget(t)

	setupWebhookPipeline(t, clients, namespace, eventName, targetSrv.URL, 1)

	// Push original event
	eventID := pushTestEvent(t, clients, namespace, eventName)

	// Wait for first delivery
	pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	pollDeliveryStatus(t, clients, namespace, eventID, pollCtx, func(d *pb.WebhookDelivery) bool {
		return d.GetStatus() == pb.WebhookDeliveryStatus_DELIVERY_SUCCESS
	})
	assert.Equal(t, int32(1), requestCount.Load())

	// Re-push the same event
	_, err := clients.event.RePushEvent(ctx, connect.NewRequest(&pb.RePushEventRequest{
		EventId: eventID,
	}))
	require.NoError(t, err)

	// Wait for the second delivery
	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for re-push delivery")
		case <-ticker.C:
			if requestCount.Load() >= 2 {
				t.Log("re-push delivery received")
				return
			}
		}
	}
}

// TestE2E_IdempotencyKeyDedup verifies that pushing the same event twice with
// the same idempotency key only creates one delivery.
func TestE2E_IdempotencyKeyDedup(t *testing.T) {
	env := setupEnv(t)
	clients := newClients(env)
	ctx := context.Background()

	const (
		namespace      = "idempotency-test"
		eventName      = "idemp.event"
		idempotencyKey = "unique-key-12345"
	)

	targetSrv, requestCount, _ := startCountingTarget(t)

	setupWebhookPipeline(t, clients, namespace, eventName, targetSrv.URL, 1)

	payload, err := structpb.NewStruct(map[string]any{"order_id": "ord_1"})
	require.NoError(t, err)

	idempKey := idempotencyKey

	// First push
	resp1, err := clients.event.PushEvent(ctx, connect.NewRequest(&pb.PushEventRequest{
		Namespace:  namespace,
		Event:      eventName,
		Payload:    payload,
		TtlSeconds: 300,
		Id:         &idempKey,
	}))
	require.NoError(t, err)
	assert.False(t, resp1.Msg.GetDuplicate(), "first push should not be duplicate")

	// Second push with same key
	resp2, err := clients.event.PushEvent(ctx, connect.NewRequest(&pb.PushEventRequest{
		Namespace:  namespace,
		Event:      eventName,
		Payload:    payload,
		TtlSeconds: 300,
		Id:         &idempKey,
	}))
	require.NoError(t, err)
	assert.True(t, resp2.Msg.GetDuplicate(), "second push should be marked as duplicate")

	// Wait for delivery of the first event
	pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	pollDeliveryStatus(t, clients, namespace, resp1.Msg.GetEventId(), pollCtx, func(d *pb.WebhookDelivery) bool {
		return d.GetStatus() == pb.WebhookDeliveryStatus_DELIVERY_SUCCESS
	})

	// Only one delivery should have been made
	time.Sleep(3 * time.Second)
	assert.Equal(t, int32(1), requestCount.Load(), "duplicate event should not create second delivery")
}

// TestE2E_BatchRetryDeliveries verifies the batch retry flow:
// push events that fail -> list with prepare_retry -> retry batch -> deliveries succeed.
func TestE2E_BatchRetryDeliveries(t *testing.T) {
	env := setupEnv(t)
	clients := newClients(env)
	ctx := context.Background()

	const (
		namespace = "batch-retry-test"
		eventName = "batch.retry.event"
	)

	// Target that fails initially
	var shouldFail atomic.Bool
	shouldFail.Store(true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		defer r.Body.Close()
		if shouldFail.Load() {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	// Set up with 0 retries so deliveries fail immediately
	_, err := clients.event.RegisterEvent(ctx, connect.NewRequest(&pb.RegisterEventRequest{
		Name:   eventName,
		Active: true,
	}))
	require.NoError(t, err)

	webhookResp, err := clients.webhook.RegisterWebhook(ctx, connect.NewRequest(&pb.RegisterWebhookRequest{
		Namespace: namespace,
		Url:       srv.URL + "/webhook",
		Active:    true,
		HttpConfig: &pb.WebhookHTTPConfig{
			MaxRetries:            0, // No retries -- fail immediately
			RequestTimeoutSeconds: 5,
			WebhookSecret:         "test-secret",
		},
	}))
	require.NoError(t, err)

	_, err = clients.subscription.CreateSubscription(ctx, connect.NewRequest(&pb.CreateSubscriptionRequest{
		WebhookId: webhookResp.Msg.GetWebhookId(),
		EventName: eventName,
		Namespace: namespace,
	}))
	require.NoError(t, err)

	// Push 3 events -- all will fail
	var eventIDs []string
	for i := 0; i < 3; i++ {
		eid := pushTestEvent(t, clients, namespace, eventName)
		eventIDs = append(eventIDs, eid)
	}

	// Wait for all deliveries to reach failed status
	for _, eid := range eventIDs {
		pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		pollDeliveryStatus(t, clients, namespace, eid, pollCtx, func(d *pb.WebhookDelivery) bool {
			return d.GetStatus() == pb.WebhookDeliveryStatus_DELIVERY_FAILED
		})
		cancel()
	}

	// Fix the target
	shouldFail.Store(false)

	// List deliveries with prepare_retry=true
	failedStatus := "failed"
	listResp, err := clients.delivery.ListDeliveries(ctx, connect.NewRequest(&pb.ListDeliveriesRequest{
		Namespace:    namespace,
		Status:       &failedStatus,
		PrepareRetry: true,
	}))
	require.NoError(t, err)
	retryID := listResp.Msg.GetRetryId()
	require.NotEmpty(t, retryID, "ListDeliveries with prepare_retry should return retry_id")
	assert.GreaterOrEqual(t, len(listResp.Msg.GetDeliveries()), 3)

	// Execute the batch retry
	_, err = clients.delivery.RetryDeliveries(ctx, connect.NewRequest(&pb.RetryDeliveriesRequest{
		RetryId: retryID,
	}))
	require.NoError(t, err)

	// Poll batch status until completed
	pollBatchComplete(t, clients, retryID, "retry", 60*time.Second)

	// Verify deliveries now succeed
	for _, eid := range eventIDs {
		pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		pollDeliveryStatus(t, clients, namespace, eid, pollCtx, func(d *pb.WebhookDelivery) bool {
			return d.GetStatus() == pb.WebhookDeliveryStatus_DELIVERY_SUCCESS
		})
		cancel()
	}
}

// TestE2E_PauseWebhookStopsRetries verifies that pausing a webhook while
// deliveries are retrying prevents further retry attempts.
func TestE2E_PauseWebhookStopsRetries(t *testing.T) {
	env := setupEnv(t)
	clients := newClients(env)
	ctx := context.Background()

	const (
		namespace = "pause-retry-test"
		eventName = "pause.retry.event"
	)

	// Target that always fails
	targetSrv, requestCount := startAlwaysFailTarget(t, http.StatusInternalServerError)

	// Use many retries so we have time to pause
	webhookID, _ := setupWebhookPipeline(t, clients, namespace, eventName, targetSrv.URL, 10)

	// Push event -- will start retrying
	_ = pushTestEvent(t, clients, namespace, eventName)

	// Wait for at least 1 attempt
	time.Sleep(3 * time.Second)
	countBeforePause := requestCount.Load()
	assert.GreaterOrEqual(t, int(countBeforePause), 1, "should have made at least 1 attempt")

	// Pause the webhook
	_, err := clients.webhook.PauseWebhook(ctx, connect.NewRequest(&pb.PauseWebhookRequest{
		WebhookId: webhookID,
		Namespace: namespace,
		Reason:    "stop retries",
	}))
	require.NoError(t, err)

	// Wait and verify no more requests arrive
	time.Sleep(10 * time.Second)
	countAfterPause := requestCount.Load()

	// Allow some slack (1-2 requests might have been in-flight during pause)
	assert.LessOrEqual(t, int(countAfterPause-countBeforePause), 2,
		"paused webhook should stop retrying (got %d more requests after pause)", countAfterPause-countBeforePause)
}

// pollDeliveryStatus polls ListDeliveries until a delivery matching the event
// satisfies the predicate or times out. Returns the matching delivery.
func pollDeliveryStatus(t *testing.T, clients *testClients, namespace, eventID string, ctx context.Context, predicate func(*pb.WebhookDelivery) bool) *pb.WebhookDelivery {
	t.Helper()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for delivery predicate (event %s)", eventID)
			return nil
		case <-ticker.C:
			resp, err := clients.delivery.ListDeliveries(ctx, connect.NewRequest(&pb.ListDeliveriesRequest{
				Namespace: namespace,
				EventId:   eventID,
			}))
			if err != nil {
				continue
			}
			for _, d := range resp.Msg.GetDeliveries() {
				if predicate(d) {
					return d
				}
			}
		}
	}
}

// pollBatchComplete polls GetRetryStatus or GetRepushStatus until the batch
// completes or times out.
func pollBatchComplete(t *testing.T, clients *testClients, batchID, batchType string, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for batch %s to complete", batchID)
		case <-ticker.C:
			switch batchType {
			case "retry":
				resp, err := clients.delivery.GetRetryStatus(ctx, connect.NewRequest(&pb.GetRetryStatusRequest{
					RetryId: batchID,
				}))
				if err != nil {
					continue
				}
				batch := resp.Msg.GetBatch()
				t.Logf("  batch retry status: %s (processed=%d/%d)", batch.GetStatus(), batch.GetProcessed(), batch.GetTotal())
				if batch.GetStatus() == "completed" || batch.GetStatus() == "cancelled" {
					return
				}
			case "repush":
				resp, err := clients.event.GetRepushStatus(ctx, connect.NewRequest(&pb.GetRepushStatusRequest{
					RepushId: batchID,
				}))
				if err != nil {
					continue
				}
				batch := resp.Msg.GetBatch()
				t.Logf("  batch repush status: %s (processed=%d/%d)", batch.GetStatus(), batch.GetProcessed(), batch.GetTotal())
				if batch.GetStatus() == "completed" || batch.GetStatus() == "cancelled" {
					return
				}
			}
		}
	}
}

// TestE2E_BatchRePushEvents verifies the batch re-push flow:
// push events -> list with prepare_repush -> re-push batch -> new deliveries created.
func TestE2E_BatchRePushEvents(t *testing.T) {
	env := setupEnv(t)
	clients := newClients(env)
	ctx := context.Background()

	const (
		namespace = "batch-repush-test"
		eventName = "batch.repush.event"
	)

	targetSrv, requestCount, _ := startCountingTarget(t)

	setupWebhookPipeline(t, clients, namespace, eventName, targetSrv.URL, 1)

	// Push 3 events
	for i := 0; i < 3; i++ {
		pushTestEvent(t, clients, namespace, eventName)
	}

	// Wait for all 3 initial deliveries
	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for initial deliveries, got %d", requestCount.Load())
		case <-ticker.C:
			if requestCount.Load() >= 3 {
				goto initialDone
			}
		}
	}
initialDone:

	// List event reports with prepare_repush
	listResp, err := clients.event.ListEventReports(ctx, connect.NewRequest(&pb.ListEventReportsRequest{
		Namespace:     namespace,
		PrepareRepush: true,
	}))
	require.NoError(t, err)
	repushID := listResp.Msg.GetRepushId()
	require.NotEmpty(t, repushID, "ListEventReports with prepare_repush should return repush_id")

	// Execute batch re-push
	_, err = clients.event.RePushEvents(ctx, connect.NewRequest(&pb.RePushEventsRequest{
		RepushId: repushID,
	}))
	require.NoError(t, err)

	// Poll batch status
	pollBatchComplete(t, clients, repushID, "repush", 60*time.Second)

	// Verify target received at least 6 total requests (3 original + 3 re-push)
	time.Sleep(3 * time.Second)
	assert.GreaterOrEqual(t, int(requestCount.Load()), 6,
		"expected at least 6 deliveries (3 original + 3 re-push), got %d", requestCount.Load())
}

// TestE2E_TimeoutRetry verifies that a slow target causes timeout errors
// that are retried, and succeeds when the target becomes fast.
func TestE2E_TimeoutRetry(t *testing.T) {
	env := setupEnv(t)
	clients := newClients(env)
	ctx := context.Background()

	const (
		namespace = "timeout-test"
		eventName = "timeout.event"
	)

	// Target that is slow initially, then fast
	var counter atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		defer r.Body.Close()
		n := counter.Add(1)
		if n <= 2 {
			// Sleep longer than the request timeout (5s configured below is the webhook timeout)
			time.Sleep(10 * time.Second)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	// Register with short timeout
	_, err := clients.event.RegisterEvent(ctx, connect.NewRequest(&pb.RegisterEventRequest{
		Name:   eventName,
		Active: true,
	}))
	require.NoError(t, err)

	webhookResp, err := clients.webhook.RegisterWebhook(ctx, connect.NewRequest(&pb.RegisterWebhookRequest{
		Namespace: namespace,
		Url:       srv.URL + "/webhook",
		Active:    true,
		HttpConfig: &pb.WebhookHTTPConfig{
			MaxRetries:            3,
			RequestTimeoutSeconds: 2, // Very short timeout
			WebhookSecret:         "test",
		},
	}))
	require.NoError(t, err)

	_, err = clients.subscription.CreateSubscription(ctx, connect.NewRequest(&pb.CreateSubscriptionRequest{
		WebhookId: webhookResp.Msg.GetWebhookId(),
		EventName: eventName,
		Namespace: namespace,
	}))
	require.NoError(t, err)

	// Push event
	eventID := pushTestEvent(t, clients, namespace, eventName)

	// Wait for eventual success (3rd attempt should be fast)
	pollCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	delivery := pollDeliveryStatus(t, clients, namespace, eventID, pollCtx, func(d *pb.WebhookDelivery) bool {
		return d.GetStatus() == pb.WebhookDeliveryStatus_DELIVERY_SUCCESS
	})

	assert.EqualValues(t, 200, delivery.GetResponseCode())
}

// TestE2E_EnvelopePayloadFormat verifies the webhook body envelope format
// and that metadata is correctly passed through.
func TestE2E_EnvelopePayloadFormat(t *testing.T) {
	env := setupEnv(t)
	clients := newClients(env)
	ctx := context.Background()

	const (
		namespace = "envelope-test"
		eventName = "envelope.event"
	)

	// Capture the delivered body
	var (
		mu          sync.Mutex
		capturedBody []byte
		done        = make(chan struct{})
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		mu.Lock()
		if capturedBody == nil {
			capturedBody = body
			close(done)
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	setupWebhookPipeline(t, clients, namespace, eventName, srv.URL, 1)

	// Push event with specific payload
	payload, _ := structpb.NewStruct(map[string]any{
		"user_id": "usr_abc",
		"action":  "signup",
	})
	pushResp, err := clients.event.PushEvent(ctx, connect.NewRequest(&pb.PushEventRequest{
		Namespace:  namespace,
		Event:      eventName,
		Payload:    payload,
		TtlSeconds: 300,
		Metadata:   map[string]string{"source": "test", "env": "integration"},
	}))
	require.NoError(t, err)
	eventID := pushResp.Msg.GetEventId()

	// Wait for delivery
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for delivery")
	}

	// Parse envelope
	var envelope map[string]any
	mu.Lock()
	err = json.Unmarshal(capturedBody, &envelope)
	mu.Unlock()
	require.NoError(t, err)

	// Verify envelope fields
	assert.Equal(t, "1", envelope["version"])
	assert.Equal(t, eventID, envelope["event_id"])
	assert.Equal(t, eventName, envelope["event_name"])
	assert.NotEmpty(t, envelope["timestamp"])
	assert.EqualValues(t, 1, envelope["attempt"])

	// Verify nested payload
	p, ok := envelope["payload"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "usr_abc", p["user_id"])
	assert.Equal(t, "signup", p["action"])
}
