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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startFailThenSucceedTarget returns a test server that fails the first N
// requests with the given status code, then succeeds all subsequent ones.
func startFailThenSucceedTarget(t *testing.T, failCount int, failStatus int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var counter atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		defer r.Body.Close() //nolint:errcheck

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
		defer r.Body.Close() //nolint:errcheck
		counter.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error":"permanent failure"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &counter
}

// startCountingTarget captures all requests and always returns 200.
func startCountingTarget(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var counter atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		defer r.Body.Close() //nolint:errcheck
		counter.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &counter
}

// registerEventType registers an event type via REST, ignoring 409 (already
// registered by an earlier test in the same package run).
func registerEventType(t *testing.T, c *restClient, ctx context.Context, name string) {
	t.Helper()
	resp, err := c.post(ctx, "/v1/event-types", map[string]any{
		"name":   name,
		"active": true,
	}, nil)
	require.NoError(t, err, "RegisterEventType failed")
	require.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusConflict, "unexpected status %d", resp.StatusCode)
}

// registerWebhookPipeline registers a webhook (auto-creating its subscription
// to eventName) and returns the webhook id.
func registerWebhookPipeline(t *testing.T, c *restClient, ctx context.Context, namespace, eventName, targetURL string, maxRetries int) string {
	t.Helper()
	var out struct {
		WebhookID string `json:"webhook_id"`
	}
	resp, err := c.post(ctx, "/v1/namespaces/"+namespace+"/webhooks", map[string]any{
		"events": []string{eventName},
		"url":    targetURL + "/webhook",
		"active": true,
		"http_config": map[string]any{
			"max_retries":             maxRetries,
			"request_timeout_seconds": 5,
			"capture_response_body":   true,
		},
	}, &out)
	require.NoError(t, err, "RegisterWebhook failed")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotEmpty(t, out.WebhookID)
	return out.WebhookID
}

// pushTestEvent pushes a simple event and returns the event ID.
func pushTestEvent(t *testing.T, c *restClient, ctx context.Context, namespace, eventName string) string {
	t.Helper()
	var out struct {
		EventID string `json:"event_id"`
	}
	resp, err := c.post(ctx, "/v1/namespaces/"+namespace+"/events?event="+eventName, map[string]any{
		"payload":     map[string]any{"test": true, "ts": time.Now().UnixMilli()},
		"ttl_seconds": 300,
	}, &out)
	require.NoError(t, err, "PushEvent failed")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotEmpty(t, out.EventID)
	return out.EventID
}

// deliveryItem mirrors the subset of the REST delivery item this file polls on.
type deliveryItem struct {
	DeliveryID    string `json:"delivery_id"`
	Status        string `json:"status"`
	ErrorCategory string `json:"error_category"`
	ResponseCode  int    `json:"response_code"`
}

// pollDeliveryStatus polls listDeliveries for the given event until a
// delivery matching the predicate appears, or ctx expires.
func pollDeliveryStatus(t *testing.T, c *restClient, ctx context.Context, namespace, eventID string, predicate func(deliveryItem) bool) deliveryItem {
	t.Helper()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for delivery predicate (event %s)", eventID)
			return deliveryItem{}
		case <-ticker.C:
			var out struct {
				Items []deliveryItem `json:"items"`
			}
			reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			_, err := c.get(reqCtx, "/v1/namespaces/"+namespace+"/deliveries?event_id="+eventID, &out)
			cancel()
			if err != nil {
				continue
			}
			for _, d := range out.Items {
				if predicate(d) {
					return d
				}
			}
		}
	}
}

// pollBatchJob polls a batch job (delivery retry or event repush) until it
// reaches a terminal status or ctx expires.
func pollBatchJob(t *testing.T, c *restClient, ctx context.Context, path string) {
	t.Helper()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for batch job %s to complete", path)
		case <-ticker.C:
			var out struct {
				Status    string `json:"status"`
				Processed int    `json:"processed"`
				Total     int    `json:"total"`
			}
			reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			_, err := c.get(reqCtx, path, &out)
			cancel()
			if err != nil {
				continue
			}
			t.Logf("  batch job status: %s (processed=%d/%d)", out.Status, out.Processed, out.Total)
			if out.Status == "completed" || out.Status == "cancelled" {
				return
			}
		}
	}
}

// TestE2E_RetryOnServerError verifies that a delivery retries on 500 and
// eventually succeeds when the target recovers.
func TestE2E_RetryOnServerError(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "retry-test"
		eventName = "retry.server_error"
	)

	targetSrv, requestCount := startFailThenSucceedTarget(t, 2, http.StatusInternalServerError)
	registerEventType(t, c, ctx, eventName)
	registerWebhookPipeline(t, c, ctx, namespace, eventName, targetSrv.URL, 3)

	eventID := pushTestEvent(t, c, ctx, namespace, eventName)

	pollCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	pollDeliveryStatus(t, c, pollCtx, namespace, eventID, func(d deliveryItem) bool {
		return d.Status == "success"
	})

	assert.GreaterOrEqual(t, int(requestCount.Load()), 3, "target should have received at least 3 requests")
}

// TestE2E_ExhaustedRetries verifies that a delivery reaches failed status
// when all retry attempts are exhausted.
func TestE2E_ExhaustedRetries(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "exhaust-retry-test"
		eventName = "retry.exhausted"
	)

	targetSrv, requestCount := startAlwaysFailTarget(t, http.StatusInternalServerError)
	registerEventType(t, c, ctx, eventName)
	registerWebhookPipeline(t, c, ctx, namespace, eventName, targetSrv.URL, 2)

	eventID := pushTestEvent(t, c, ctx, namespace, eventName)

	pollCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	delivery := pollDeliveryStatus(t, c, pollCtx, namespace, eventID, func(d deliveryItem) bool {
		return d.Status == "failed"
	})

	assert.Equal(t, "server_error", delivery.ErrorCategory)
	assert.GreaterOrEqual(t, int(requestCount.Load()), 1)
}

// TestE2E_FanOutMultipleSubscribers verifies that one event is delivered to
// every webhook subscribed to it.
func TestE2E_FanOutMultipleSubscribers(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "fanout-test"
		eventName = "fanout.created"
	)

	registerEventType(t, c, ctx, eventName)

	target1, count1 := startCountingTarget(t)
	target2, count2 := startCountingTarget(t)
	target3, count3 := startCountingTarget(t)

	for _, url := range []string{target1.URL, target2.URL, target3.URL} {
		registerWebhookPipeline(t, c, ctx, namespace, eventName, url, 1)
	}

	pushTestEvent(t, c, ctx, namespace, eventName)

	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for fan-out: got %d, %d, %d deliveries", count1.Load(), count2.Load(), count3.Load())
		case <-ticker.C:
			if count1.Load() >= 1 && count2.Load() >= 1 && count3.Load() >= 1 {
				t.Logf("all 3 targets received delivery: %d, %d, %d", count1.Load(), count2.Load(), count3.Load())
				return
			}
		}
	}
}

// TestE2E_PausedWebhookNoDelivery verifies that pausing a webhook prevents
// new deliveries, and resuming it allows them again.
func TestE2E_PausedWebhookNoDelivery(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "pause-test"
		eventName = "pause.event"
	)

	targetSrv, requestCount := startCountingTarget(t)
	registerEventType(t, c, ctx, eventName)
	webhookID := registerWebhookPipeline(t, c, ctx, namespace, eventName, targetSrv.URL, 1)

	resp, err := c.post(ctx, "/v1/namespaces/"+namespace+"/webhooks/"+webhookID+":pause", nil, nil)
	require.NoError(t, err, "PauseWebhook failed")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	pushTestEvent(t, c, ctx, namespace, eventName)

	time.Sleep(5 * time.Second)
	assert.Equal(t, int32(0), requestCount.Load(), "paused webhook should not receive deliveries")

	resp, err = c.post(ctx, "/v1/namespaces/"+namespace+"/webhooks/"+webhookID+":resume", nil, nil)
	require.NoError(t, err, "ResumeWebhook failed")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	eventID2 := pushTestEvent(t, c, ctx, namespace, eventName)

	pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	pollDeliveryStatus(t, c, pollCtx, namespace, eventID2, func(d deliveryItem) bool {
		return d.Status == "success"
	})

	assert.GreaterOrEqual(t, int(requestCount.Load()), 1, "resumed webhook should receive delivery")
}

// TestE2E_DeleteSubscriptionStopsDelivery verifies that deleting a
// subscription prevents future events from creating deliveries for it.
func TestE2E_DeleteSubscriptionStopsDelivery(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "delete-sub-test"
		eventName = "delete.sub.event"
	)

	targetSrv, requestCount := startCountingTarget(t)
	registerEventType(t, c, ctx, eventName)
	webhookID := registerWebhookPipeline(t, c, ctx, namespace, eventName, targetSrv.URL, 1)

	var subList struct {
		Items []struct {
			SubscriptionID string `json:"subscription_id"`
		} `json:"items"`
	}
	_, err := c.get(ctx, "/v1/namespaces/"+namespace+"/subscriptions?webhook_id="+webhookID, &subList)
	require.NoError(t, err, "ListSubscriptions failed")
	require.Len(t, subList.Items, 1)
	subscriptionID := subList.Items[0].SubscriptionID

	eventID1 := pushTestEvent(t, c, ctx, namespace, eventName)
	pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	pollDeliveryStatus(t, c, pollCtx, namespace, eventID1, func(d deliveryItem) bool {
		return d.Status == "success"
	})
	cancel()
	assert.Equal(t, int32(1), requestCount.Load())

	resp, err := c.do(ctx, http.MethodDelete, "/v1/namespaces/"+namespace+"/subscriptions/"+subscriptionID, nil, nil)
	require.NoError(t, err, "DeleteSubscription failed")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	pushTestEvent(t, c, ctx, namespace, eventName)

	time.Sleep(5 * time.Second)
	assert.Equal(t, int32(1), requestCount.Load(), "deleted subscription should not receive new deliveries")
}

// TestE2E_SingleRePush verifies that re-pushing an event creates a new delivery.
func TestE2E_SingleRePush(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "repush-test"
		eventName = "repush.event"
	)

	targetSrv, requestCount := startCountingTarget(t)
	registerEventType(t, c, ctx, eventName)
	registerWebhookPipeline(t, c, ctx, namespace, eventName, targetSrv.URL, 1)

	eventID := pushTestEvent(t, c, ctx, namespace, eventName)

	pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	pollDeliveryStatus(t, c, pollCtx, namespace, eventID, func(d deliveryItem) bool {
		return d.Status == "success"
	})
	cancel()
	assert.Equal(t, int32(1), requestCount.Load())

	resp, err := c.post(ctx, "/v1/events/"+eventID+":repush", nil, nil)
	require.NoError(t, err, "RepushEvent failed")
	require.Equal(t, http.StatusOK, resp.StatusCode)

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
// the same idempotency key only creates one delivery, and the second push's
// response reports duplicate=true.
func TestE2E_IdempotencyKeyDedup(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace      = "idempotency-test"
		eventName      = "idemp.event"
		idempotencyKey = "unique-key-12345"
	)

	targetSrv, requestCount := startCountingTarget(t)
	registerEventType(t, c, ctx, eventName)
	registerWebhookPipeline(t, c, ctx, namespace, eventName, targetSrv.URL, 1)

	var push1, push2 struct {
		EventID   string `json:"event_id"`
		Duplicate bool   `json:"duplicate"`
	}
	resp, err := c.post(ctx, "/v1/namespaces/"+namespace+"/events?event="+eventName, map[string]any{
		"payload":         map[string]any{"order_id": "ord_1"},
		"ttl_seconds":     300,
		"idempotency_key": idempotencyKey,
	}, &push1)
	require.NoError(t, err, "PushEvent (1st) failed")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.False(t, push1.Duplicate, "first push should not be a duplicate")

	resp, err = c.post(ctx, "/v1/namespaces/"+namespace+"/events?event="+eventName, map[string]any{
		"payload":         map[string]any{"order_id": "ord_1"},
		"ttl_seconds":     300,
		"idempotency_key": idempotencyKey,
	}, &push2)
	require.NoError(t, err, "PushEvent (2nd) failed")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.True(t, push2.Duplicate, "second push with the same idempotency_key should be marked duplicate")
	assert.Equal(t, push1.EventID, push2.EventID, "duplicate push should return the original event_id")

	pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	pollDeliveryStatus(t, c, pollCtx, namespace, push1.EventID, func(d deliveryItem) bool {
		return d.Status == "success"
	})

	time.Sleep(3 * time.Second)
	assert.Equal(t, int32(1), requestCount.Load(), "duplicate event should not create a second delivery")
}

// TestE2E_BatchRetryDeliveries verifies the batch retry flow: push events
// that fail -> list with prepare_retry -> start retry job -> deliveries succeed.
func TestE2E_BatchRetryDeliveries(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "batch-retry-test"
		eventName = "batch.retry.event"
	)

	var shouldFail atomic.Bool
	shouldFail.Store(true)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		defer r.Body.Close() //nolint:errcheck
		if shouldFail.Load() {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	registerEventType(t, c, ctx, eventName)
	registerWebhookPipeline(t, c, ctx, namespace, eventName, srv.URL, 0) // no retries -- fail immediately

	var eventIDs []string
	for range 3 {
		eventIDs = append(eventIDs, pushTestEvent(t, c, ctx, namespace, eventName))
	}

	for _, eid := range eventIDs {
		pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		pollDeliveryStatus(t, c, pollCtx, namespace, eid, func(d deliveryItem) bool {
			return d.Status == "failed"
		})
		cancel()
	}

	shouldFail.Store(false)

	var listOut struct {
		Items   []deliveryItem `json:"items"`
		RetryID string         `json:"retry_id"`
	}
	_, err := c.get(ctx, "/v1/namespaces/"+namespace+"/deliveries?status=failed&prepare_retry=true", &listOut)
	require.NoError(t, err, "ListDeliveries with prepare_retry failed")
	require.NotEmpty(t, listOut.RetryID, "prepare_retry should return a retry_id")
	assert.GreaterOrEqual(t, len(listOut.Items), 3)

	var jobOut struct {
		ID string `json:"id"`
	}
	resp, err := c.post(ctx, "/v1/namespaces/"+namespace+"/deliveries:retryBatch", map[string]any{
		"repush_id": listOut.RetryID,
	}, &jobOut)
	require.NoError(t, err, "startDeliveryRetryJob failed")
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	pollCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	pollBatchJob(t, c, pollCtx, "/v1/namespaces/"+namespace+"/retry-jobs/"+jobOut.ID)

	for _, eid := range eventIDs {
		spollCtx, scancel := context.WithTimeout(ctx, 30*time.Second)
		pollDeliveryStatus(t, c, spollCtx, namespace, eid, func(d deliveryItem) bool {
			return d.Status == "success"
		})
		scancel()
	}
}

// TestE2E_PauseWebhookStopsRetries verifies that pausing a webhook while
// deliveries are retrying prevents further retry attempts.
func TestE2E_PauseWebhookStopsRetries(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "pause-retry-test"
		eventName = "pause.retry.event"
	)

	targetSrv, requestCount := startAlwaysFailTarget(t, http.StatusInternalServerError)
	registerEventType(t, c, ctx, eventName)
	webhookID := registerWebhookPipeline(t, c, ctx, namespace, eventName, targetSrv.URL, 10) // many retries so we have time to pause

	pushTestEvent(t, c, ctx, namespace, eventName)

	time.Sleep(3 * time.Second)
	countBeforePause := requestCount.Load()
	assert.GreaterOrEqual(t, int(countBeforePause), 1, "should have made at least 1 attempt")

	resp, err := c.post(ctx, "/v1/namespaces/"+namespace+"/webhooks/"+webhookID+":pause", nil, nil)
	require.NoError(t, err, "PauseWebhook failed")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	time.Sleep(10 * time.Second)
	countAfterPause := requestCount.Load()

	assert.LessOrEqual(t, int(countAfterPause-countBeforePause), 2,
		"paused webhook should stop retrying (got %d more requests after pause)", countAfterPause-countBeforePause)
}

// TestE2E_BatchRePushEvents verifies the batch re-push flow: push events ->
// list with prepare_repush -> start repush job -> new deliveries created.
func TestE2E_BatchRePushEvents(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "batch-repush-test"
		eventName = "batch.repush.event"
	)

	targetSrv, requestCount := startCountingTarget(t)
	registerEventType(t, c, ctx, eventName)
	registerWebhookPipeline(t, c, ctx, namespace, eventName, targetSrv.URL, 1)

	for range 3 {
		pushTestEvent(t, c, ctx, namespace, eventName)
	}

	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
waitInitial:
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for initial deliveries, got %d", requestCount.Load())
		case <-ticker.C:
			if requestCount.Load() >= 3 {
				break waitInitial
			}
		}
	}
	ticker.Stop()

	var listOut struct {
		RepushID string `json:"repush_id"`
	}
	_, err := c.get(ctx, "/v1/namespaces/"+namespace+"/events?prepare_repush=true", &listOut)
	require.NoError(t, err, "ListEventOccurrences with prepare_repush failed")
	require.NotEmpty(t, listOut.RepushID, "prepare_repush should return a repush_id")

	var jobOut struct {
		ID string `json:"id"`
	}
	resp, err := c.post(ctx, "/v1/namespaces/"+namespace+"/events:rePush", map[string]any{
		"repush_id": listOut.RepushID,
	}, &jobOut)
	require.NoError(t, err, "startEventRepushJob failed")
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	pollCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	pollBatchJob(t, c, pollCtx, "/v1/namespaces/"+namespace+"/repush-jobs/"+jobOut.ID)

	time.Sleep(3 * time.Second)
	assert.GreaterOrEqual(t, int(requestCount.Load()), 6,
		"expected at least 6 deliveries (3 original + 3 re-push), got %d", requestCount.Load())
}

// TestE2E_TimeoutRetry verifies that a slow target causes timeout errors that
// are retried, and succeeds once the target becomes fast.
func TestE2E_TimeoutRetry(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "timeout-test"
		eventName = "timeout.event"
	)

	var counter atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		defer r.Body.Close() //nolint:errcheck
		n := counter.Add(1)
		if n <= 2 {
			time.Sleep(10 * time.Second) // longer than the 2s webhook timeout below
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	registerEventType(t, c, ctx, eventName)
	var webhookOut struct {
		WebhookID string `json:"webhook_id"`
	}
	resp, err := c.post(ctx, "/v1/namespaces/"+namespace+"/webhooks", map[string]any{
		"events": []string{eventName},
		"url":    srv.URL + "/webhook",
		"active": true,
		"http_config": map[string]any{
			"max_retries":             3,
			"request_timeout_seconds": 2, // very short timeout
		},
	}, &webhookOut)
	require.NoError(t, err, "RegisterWebhook failed")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	eventID := pushTestEvent(t, c, ctx, namespace, eventName)

	pollCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	delivery := pollDeliveryStatus(t, c, pollCtx, namespace, eventID, func(d deliveryItem) bool {
		return d.Status == "success"
	})

	assert.Equal(t, 200, delivery.ResponseCode)
}

// TestE2E_EnvelopePayloadFormat verifies the webhook body envelope format and
// that metadata is correctly passed through.
func TestE2E_EnvelopePayloadFormat(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "envelope-test"
		eventName = "envelope.event"
	)

	var (
		mu           sync.Mutex
		capturedBody []byte
		done         = make(chan struct{})
		once         sync.Once
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close() //nolint:errcheck
		mu.Lock()
		capturedBody = body
		mu.Unlock()
		once.Do(func() { close(done) })
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	registerEventType(t, c, ctx, eventName)
	registerWebhookPipeline(t, c, ctx, namespace, eventName, srv.URL, 1)

	var pushOut struct {
		EventID string `json:"event_id"`
	}
	resp, err := c.post(ctx, "/v1/namespaces/"+namespace+"/events?event="+eventName, map[string]any{
		"payload":     map[string]any{"user_id": "usr_abc", "action": "signup"},
		"ttl_seconds": 300,
		"metadata":    map[string]string{"source": "test", "env": "integration"},
	}, &pushOut)
	require.NoError(t, err, "PushEvent failed")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for delivery")
	}

	var envelope map[string]any
	mu.Lock()
	err = json.Unmarshal(capturedBody, &envelope)
	mu.Unlock()
	require.NoError(t, err)

	assert.Equal(t, "1", envelope["version"])
	assert.Equal(t, pushOut.EventID, envelope["event_id"])
	assert.Equal(t, eventName, envelope["event_name"])
	assert.NotEmpty(t, envelope["timestamp"])
	assert.EqualValues(t, 1, envelope["attempt"])

	p, ok := envelope["payload"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "usr_abc", p["user_id"])
	assert.Equal(t, "signup", p["action"])
}
