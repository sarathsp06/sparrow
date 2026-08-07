package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	sparrowerrors "github.com/sarathsp06/sparrow/pkg/errors"

	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/crypto"
)

// WebhookWorker handles webhook delivery jobs
type WebhookWorker struct {
	river.WorkerDefaults[WebhookArgs]
	webhookRepo      store.WebhookRepository
	eventRepo        store.EventRepository
	subscriptionRepo store.SubscriptionRepository
	healthRepo       store.HealthRepository
	rateLimitRepo    store.RateLimitRepository
	cryptoSvc        *crypto.Service
	tracer           trace.Tracer
	logger           *slog.Logger
	metrics          *observability.SparrowMetrics
	client           *client.WebhookClient
}

// NewWebhookWorker creates a new webhook worker
func NewWebhookWorker(webhookRepo store.WebhookRepository, eventRepo store.EventRepository, subscriptionRepo store.SubscriptionRepository, healthRepo store.HealthRepository, rateLimitRepo store.RateLimitRepository, cryptoSvc *crypto.Service, clientConfig *client.Config) *WebhookWorker {
	metrics, err := observability.NewSparrowMetrics()
	if err != nil {
		// Log error but continue without metrics
		slog.Default().With("component", "webhook-worker").Error("Failed to initialize metrics", "error", err)
	}

	// Initialize the centralized webhook client
	webhookClient := client.NewWebhookClient(clientConfig)

	return &WebhookWorker{
		webhookRepo:      webhookRepo,
		eventRepo:        eventRepo,
		subscriptionRepo: subscriptionRepo,
		healthRepo:       healthRepo,
		rateLimitRepo:    rateLimitRepo,
		cryptoSvc:        cryptoSvc,
		logger:           slog.Default().With("component", "webhook-worker"),
		tracer:           observability.GetTracer("sparrow.workers.webhook"),
		metrics:          metrics,
		client:           webhookClient,
	}
}

// Work processes the webhook delivery job
func (w *WebhookWorker) Work(ctx context.Context, job *river.Job[WebhookArgs]) error {
	args := job.Args

	// get trace id and set that as metadata
	carrier := make(propagation.MapCarrier)
	if unmarshallErr := json.Unmarshal(job.Metadata, &carrier); unmarshallErr != nil {
		w.logger.ErrorContext(ctx, "Failed to unmarshal job metadata", "error", unmarshallErr, "event_id", args.EventID)
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	// Parse tenant ID from job args
	tenantID := uuid.MustParse(args.TenantID)

	// Get webhook configuration from database
	webhook, err := w.webhookRepo.GetWebhookByID(ctx, tenantID, uuid.MustParse(args.WebhookID), args.Namespace)
	if err != nil {
		w.logger.ErrorContext(ctx, "Failed to get webhook configuration", "error", err, "webhook_id", args.WebhookID)
		_ = w.eventRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID), store.StatusFailed, 0, "", fmt.Sprintf("Failed to get webhook configuration: %v", err), "unknown")
		return fmt.Errorf("failed to get webhook configuration: %w", err)
	}

	// Get event record from database
	eventRecord, err := w.eventRepo.GetEventByID(ctx, tenantID, uuid.MustParse(args.EventID))
	if err != nil {
		w.logger.ErrorContext(ctx, "Failed to get event record", "error", err, "event_id", args.EventID)
		_ = w.eventRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID), store.StatusFailed, 0, "", fmt.Sprintf("Failed to get event record: %v", err), "unknown")
		return fmt.Errorf("failed to get event record: %w", err)
	}

	// Get subscription if available
	var subscription *store.EventSubscription
	if args.SubscriptionID != "" {
		subscription, err = w.subscriptionRepo.GetSubscription(ctx, tenantID, uuid.MustParse(args.SubscriptionID))
		if err != nil {
			// If subscription is missing, we might still want to proceed if it's a legacy delivery,
			// but for now let's assume strict consistency or log warning.
			// Given the refactor, we expect subscription to exist if ID is passed.
			w.logger.WarnContext(ctx, "Failed to get subscription", "error", err, "subscription_id", args.SubscriptionID)
			// Continue without subscription (will use default webhook config)
		}
	}

	ctx, span := w.tracer.Start(ctx, "webhook.delivery",
		trace.WithAttributes(
			attribute.String("delivery_id", args.DeliveryID),
			attribute.String("webhook_id", args.WebhookID),
			attribute.String("event_id", args.EventID),
			attribute.String("url", webhook.URL),
			attribute.String("namespace", args.Namespace),
			attribute.String("event", eventRecord.Event),
		),
	)
	defer span.End()

	log := w.logger.With("job_id", job.ID, "delivery_id", args.DeliveryID, "webhook_id", args.WebhookID)

	// Check if the delivery has expired
	if time.Now().After(args.ExpiresAt) {
		span.SetStatus(otelcodes.Error, "webhook delivery expired")
		log.WarnContext(ctx, "Webhook delivery expired", "expires_at", args.ExpiresAt)

		err := w.eventRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID), store.StatusExpired, 0, "", "Delivery expired", "unknown")
		if err != nil {
			log.ErrorContext(ctx, "Failed to update delivery status to expired", "error", err)
		}
		return nil
	}

	log.InfoContext(ctx, "Processing webhook delivery", "event_id", args.EventID, "url", webhook.URL)

	// Rate limiting: check leaky bucket before sending.
	// AcquireDeliverySlot atomically advances the bucket and returns the slot
	// assigned to this delivery. If the slot is in the future, snooze the job.
	if webhook.RateLimitRPS != nil && *webhook.RateLimitRPS > 0 {
		nextDeliveryAt, rateLimitRPS, err := w.rateLimitRepo.AcquireDeliverySlot(ctx, uuid.MustParse(args.WebhookID))
		if err != nil {
			log.ErrorContext(ctx, "Failed to acquire delivery slot", "error", err, "webhook_id", args.WebhookID)
			// Non-fatal: proceed without rate limiting rather than failing delivery
		} else if rateLimitRPS > 0 {
			// Our slot = nextDeliveryAt - (1/rateLimitRPS)
			interval := time.Duration(float64(time.Second) / rateLimitRPS)
			mySlot := nextDeliveryAt.Add(-interval)
			delay := time.Until(mySlot)
			if delay > 0 {
				log.InfoContext(ctx, "Rate limited, snoozing delivery",
					"webhook_id", args.WebhookID,
					"delivery_id", args.DeliveryID,
					"snooze_until", mySlot,
					"delay", delay,
					"rate_limit_rps", rateLimitRPS,
				)
				span.SetAttributes(attribute.Float64("rate_limit_rps", rateLimitRPS))
				span.SetAttributes(attribute.String("rate_limit_action", "snoozed"))
				return river.JobSnooze(delay)
			}
		}
	}

	defaultPayload := eventRecord.Payload

	var payloadBytes []byte

	// Check for transformation
	if subscription != nil && subscription.TransformEnabled && subscription.TransformTemplate != "" {
		payloadBytes, err = w.client.TransformPayload(subscription.TransformTemplate, client.NewWebhookTemplateContext(
			args.EventID,
			eventRecord.Event,
			time.Now().UTC().Format(time.RFC3339),
			job.Attempt,
			eventRecord.Payload,
		))
		if err != nil {
			// Graceful degradation: template transform failed, fall back to
			// envelope payload instead of failing the delivery. Template errors
			// are not transient (retrying won't fix a bad template), so we log
			// the warning and continue with the default payload format.
			log.WarnContext(ctx, "Template transformation failed, falling back to envelope payload",
				"error", err,
				"subscription_id", args.SubscriptionID,
				"delivery_id", args.DeliveryID,
			)
			payloadBytes, err = client.BuildEnvelopePayload(
				args.EventID,
				eventRecord.Event,
				job.Attempt,
				defaultPayload,
			)
			if err != nil {
				log.ErrorContext(ctx, "Failed to marshal fallback envelope payload", "error", err)
				return err
			}
		}
	} else {
		// Use default JSON envelope
		payloadBytes, err = client.BuildEnvelopePayload(
			args.EventID,
			eventRecord.Event,
			job.Attempt,
			defaultPayload,
		)
		if err != nil {
			log.ErrorContext(ctx, "Failed to marshal webhook payload", "error", err)
			return err
		}
	}

	// Prepare delivery request using centralized client logic. A decrypt
	// failure of a configured secret is fail-closed: return the error so River
	// retries rather than delivering an unsigned/secret-less request.
	deliveryReq, err := client.PrepareDeliveryRequest(webhook, subscription, eventRecord, args.DeliveryID, payloadBytes, w.cryptoSvc)
	if err != nil {
		log.ErrorContext(ctx, "Failed to prepare delivery request", "error", err, "webhook_id", args.WebhookID, "delivery_id", args.DeliveryID)
		return fmt.Errorf("prepare delivery request: %w", err)
	}

	// Store the request body in the delivery record
	if err := w.eventRepo.UpdateDeliveryRequestBody(ctx, uuid.MustParse(args.DeliveryID), string(payloadBytes)); err != nil {
		log.WarnContext(ctx, "Failed to store request body", "error", err, "delivery_id", args.DeliveryID)
	}

	// Send the request
	resp, duration, err := w.client.Send(ctx, deliveryReq)

	if err != nil {
		// Classify the network/transport error
		errorCategory := sparrowerrors.ClassifyError(err)

		log.ErrorContext(ctx, "Failed to send webhook",
			"error", err,
			"duration_ms", duration.Milliseconds(),
			"error_category", string(errorCategory),
		)

		_ = w.eventRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID), store.StatusFailed, 0, "", fmt.Sprintf("Request failed: %v", err), string(errorCategory))

		// Record health event and update health state
		w.recordHealthOutcome(ctx, log, args.WebhookID, args.DeliveryID, false, int(duration.Milliseconds()), 0, err.Error(), string(errorCategory))

		// For non-retryable error categories (DNS, TLS), cancel River retries
		// by returning nil instead of an error. The delivery is already marked failed.
		if !sparrowerrors.IsRetryableCategory(errorCategory) {
			log.WarnContext(ctx, "Non-retryable error category, cancelling retries",
				"error_category", string(errorCategory),
			)
			return nil
		}

		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body. The body is always consumed to allow HTTP connection reuse.
	// CaptureResponseBody controls the storage size limit:
	//   false (default) -> store up to 1 KB (useful for error diagnostics)
	//   true            -> store up to 1 MB (full response capture)
	const maxResponseBodyBytes = 1024 * 1024 // 1 MB
	var body []byte
	var bodyErr error
	if webhook.CaptureResponseBody {
		body, bodyErr = client.ReadBody(resp, maxResponseBodyBytes)
	} else {
		body, bodyErr = client.ReadBody(resp, 1000) // 1 KB — enough for error messages
	}

	if bodyErr != nil {
		log.WarnContext(ctx, "Failed to read response body", "error", bodyErr)
		body = []byte("Failed to read response body")
	}

	log.InfoContext(ctx, "Webhook response received",
		"status_code", resp.StatusCode,
		"duration_ms", duration.Milliseconds(),
	)

	isSuccess := isSuccessStatusCode(resp.StatusCode, webhook.ExpectedStatusCodes)

	if isSuccess {
		span.SetStatus(otelcodes.Ok, "webhook delivered successfully")

		// Metrics are already recorded by the client!
		// But we might want to record worker-specific metrics if any.

		err := w.eventRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID),
			store.StatusSuccess, resp.StatusCode, string(body), "", string(sparrowerrors.CategorySuccess))
		if err != nil {
			log.ErrorContext(ctx, "Failed to update delivery status to success", "error", err)
		}

		w.recordHealthOutcome(ctx, log, args.WebhookID, args.DeliveryID, true, int(duration.Milliseconds()), resp.StatusCode, "", string(sparrowerrors.CategorySuccess))

		return nil
	}

	// Handle 429 Too Many Requests: snooze the job based on Retry-After header.
	// This doesn't count as a retry attempt — the target is explicitly asking us to slow down.
	if resp.StatusCode == http.StatusTooManyRequests {
		snoozeDuration := parseRetryAfter(resp.Header.Get("Retry-After"))

		log.WarnContext(ctx, "Target returned 429 Too Many Requests, snoozing delivery",
			"delivery_id", args.DeliveryID,
			"webhook_id", args.WebhookID,
			"snooze_duration", snoozeDuration,
			"retry_after_header", resp.Header.Get("Retry-After"),
		)
		span.SetAttributes(
			attribute.String("rate_limit_action", "snoozed_429"),
			attribute.Int64("snooze_seconds", int64(snoozeDuration.Seconds())),
		)

		// Record the 429 as a health event (the endpoint is overloaded)
		w.recordHealthOutcome(ctx, log, args.WebhookID, args.DeliveryID, false,
			int(duration.Milliseconds()), resp.StatusCode,
			"HTTP 429: Too Many Requests", string(sparrowerrors.CategoryRateLimited))

		// Don't update delivery status to failed — we're going to retry via snooze.
		// The delivery remains in its current status (pending/retrying).
		return river.JobSnooze(snoozeDuration)
	}

	// Failure case - classify the error.
	// If the status code is in a standard error range (4xx, 5xx), classify by HTTP range.
	// If it's a 2xx/3xx that simply didn't match expected_status_codes, use unexpected_status.
	var errorCategory sparrowerrors.ErrorCategory
	httpCategory := sparrowerrors.ClassifyHTTPStatus(resp.StatusCode)
	if httpCategory == sparrowerrors.CategorySuccess || httpCategory == sparrowerrors.CategoryUnknown {
		// The HTTP status itself is OK (2xx) or ambiguous (1xx/3xx), but it wasn't
		// in the webhook's expected_status_codes list. This is a configuration/contract
		// mismatch, not a server error.
		errorCategory = sparrowerrors.CategoryUnexpectedStatus
	} else {
		errorCategory = httpCategory
	}
	errorMessage := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)
	span.SetStatus(otelcodes.Error, "webhook delivery failed")
	span.SetAttributes(attribute.String("error_category", string(errorCategory)))

	err = w.eventRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID),
		store.StatusFailed, resp.StatusCode, string(body), errorMessage, string(errorCategory))
	if err != nil {
		log.ErrorContext(ctx, "Failed to update delivery status to failed", "error", err)
	}

	w.recordHealthOutcome(ctx, log, args.WebhookID, args.DeliveryID, false, int(duration.Milliseconds()), resp.StatusCode, errorMessage, string(errorCategory))

	log.WarnContext(ctx, "Webhook delivery failed",
		"status_code", resp.StatusCode,
		"error_category", string(errorCategory),
		"duration_ms", duration.Milliseconds(),
	)

	// For client errors (4xx), do not retry - return nil to cancel River retries.
	// The delivery is already marked as failed with the appropriate error category.
	if !sparrowerrors.IsRetryableCategory(errorCategory) {
		log.WarnContext(ctx, "Non-retryable HTTP status, cancelling retries",
			"status_code", resp.StatusCode,
			"error_category", string(errorCategory),
		)
		return nil
	}

	return fmt.Errorf("webhook delivery failed: %s", errorMessage)
}

// recordHealthOutcome records a webhook health event and updates the health state.
// This is the shared implementation for all delivery outcome paths (success, client error, server error).
func (w *WebhookWorker) recordHealthOutcome(ctx context.Context, log *slog.Logger, webhookID, deliveryID string, success bool, durationMs int, statusCode int, errorMessage string, errorCategory string) {
	webhookUUID := uuid.MustParse(webhookID)
	deliveryUUID := uuid.MustParse(deliveryID)

	if err := w.healthRepo.RecordWebhookHealthEvent(ctx, webhookUUID, deliveryUUID, success, durationMs, statusCode, errorMessage, errorCategory); err != nil {
		log.ErrorContext(ctx, "Failed to record health event", "error", err)
	}
	if err := w.healthRepo.UpdateWebhookHealthState(ctx, webhookUUID, success, time.Now()); err != nil {
		log.ErrorContext(ctx, "Failed to update webhook health state", "error", err)
	}
}

// Helper function for status code checking (re-implemented as standalone or private method)
func isSuccessStatusCode(statusCode int, expectedStatusCodes []int64) bool {
	if len(expectedStatusCodes) == 0 {
		return statusCode >= 200 && statusCode < 300
	}
	for _, expected := range expectedStatusCodes {
		if statusCode == int(expected) {
			return true
		}
		// Simple range check support (e.g. 20 for 200-209) could be added here if needed
		// For now, exact match or simple 2xx default
	}
	return false
}

// defaultRetryAfter is the default snooze duration when a 429 response
// has no Retry-After header or the header can't be parsed.
const defaultRetryAfter = 60 * time.Second

// maxRetryAfter caps the snooze duration to prevent a misbehaving server
// from parking our jobs for unreasonable durations.
const maxRetryAfter = 15 * time.Minute

// parseRetryAfter extracts a delay duration from an HTTP Retry-After header.
// The header can be either a number of seconds (e.g. "120") or an HTTP-date
// (e.g. "Thu, 01 Dec 2025 16:00:00 GMT"). Returns defaultRetryAfter if the
// header is empty or unparseable. Clamps to maxRetryAfter.
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return defaultRetryAfter
	}

	// Try parsing as seconds (most common for rate limiting)
	if seconds, err := strconv.Atoi(header); err == nil {
		d := time.Duration(seconds) * time.Second
		if d <= 0 {
			return defaultRetryAfter
		}
		if d > maxRetryAfter {
			return maxRetryAfter
		}
		return d
	}

	// Try parsing as HTTP-date (RFC 7231 §7.1.1.1)
	if t, err := time.Parse(time.RFC1123, header); err == nil {
		d := time.Until(t)
		if d <= 0 {
			return defaultRetryAfter
		}
		if d > maxRetryAfter {
			return maxRetryAfter
		}
		return d
	}

	return defaultRetryAfter
}
