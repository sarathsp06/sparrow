package queue

import (
	"context"
	"encoding/json"
	"errors"
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
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	sparrowerrors "github.com/sarathsp06/sparrow/pkg/errors"

	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/crypto"
	"github.com/sarathsp06/sparrow/pkg/template"
)

// WebhookWorker handles webhook delivery jobs
type WebhookWorker struct {
	river.WorkerDefaults[WebhookArgs]
	webhookRepo      store.WebhookRepository
	eventRepo        systemEventRepo
	deliveryRepo     store.DeliveryRepository
	subscriptionRepo store.SubscriptionRepository
	healthRepo       store.HealthRepository
	rateLimitRepo    store.RateLimitRepository
	alertConfigRepo  store.AlertConfigRepository
	jobInserter      JobInserter
	cryptoSvc        *crypto.Service
	tracer           trace.Tracer
	logger           *slog.Logger
	client           *client.WebhookClient
	// listen delivers to listen sessions (sparrow-cli:// webhooks).
	listen *listenSender
	// captureLimit is the storage limit for capture_response_body webhooks.
	captureLimit int64
	// templateErrors counts payload transform failures, labelled by the
	// subscription's on_transform_error. Nil if the meter is unavailable.
	templateErrors metric.Int64Counter
	// autoDisable decides when a webhook whose receiver keeps failing is
	// paused automatically.
	autoDisable AutoDisablePolicy
	// metrics are the delivery-outcome instruments; nil fields are skipped.
	metrics workerMetrics
}

// AutoDisablePolicy pauses a webhook once its receiver has failed at least
// MinFailures attempts in a row with no success for After. A zero After (or
// MinFailures) turns automatic disabling off.
type AutoDisablePolicy struct {
	After       time.Duration
	MinFailures int
}

func (p AutoDisablePolicy) enabled() bool {
	return p.After > 0 && p.MinFailures > 0
}

// NewWebhookWorker creates a new webhook worker
func NewWebhookWorker(webhookRepo store.WebhookRepository, eventRepo systemEventRepo, deliveryRepo store.DeliveryRepository, subscriptionRepo store.SubscriptionRepository, healthRepo store.HealthRepository, rateLimitRepo store.RateLimitRepository, alertConfigRepo store.AlertConfigRepository, listenRepo store.ListenRepository, jobInserter JobInserter, cryptoSvc *crypto.Service, clientConfig *client.Config, autoDisable AutoDisablePolicy) *WebhookWorker {
	// Initialize the centralized webhook client
	webhookClient := client.NewWebhookClient(clientConfig)

	return &WebhookWorker{
		webhookRepo:      webhookRepo,
		eventRepo:        eventRepo,
		deliveryRepo:     deliveryRepo,
		subscriptionRepo: subscriptionRepo,
		healthRepo:       healthRepo,
		rateLimitRepo:    rateLimitRepo,
		alertConfigRepo:  alertConfigRepo,
		jobInserter:      jobInserter,
		cryptoSvc:        cryptoSvc,
		logger:           slog.Default().With("component", "webhook-worker"),
		tracer:           observability.GetTracer("sparrow.workers.webhook"),
		client:           webhookClient,
		listen:           newListenSender(listenRepo),
		captureLimit:     clientConfig.CapturedResponseLimit(),
		templateErrors:   newTemplateErrorCounter(),
		autoDisable:      autoDisable,
		metrics:          newWorkerMetrics(),
	}
}

func newTemplateErrorCounter() metric.Int64Counter {
	c, err := observability.GetMeter("sparrow").Int64Counter(
		"sparrow_template_errors_total",
		metric.WithDescription("Payload transform templates that failed to render, by on_transform_error"),
	)
	if err != nil {
		return nil
	}
	return c
}

// maxRetryDelay caps the exponential backoff regardless of configuration.
const maxRetryDelay = 24 * time.Hour

// NextRetry honors the webhook's configured retry_backoff_seconds: the delay
// after attempt N is base * 2^(N-1), capped at maxRetryDelay. A zero base
// (jobs enqueued before the field existed) returns the zero time, which tells
// River to fall back to its default retry policy.
func (w *WebhookWorker) NextRetry(job *river.Job[WebhookArgs]) time.Time {
	base := job.Args.RetryBackoffSeconds
	if base <= 0 {
		return time.Time{}
	}
	shift := job.Attempt - 1
	if shift < 0 {
		shift = 0
	}
	if shift > 20 {
		shift = 20 // past this the cap always wins; avoid overflow
	}
	delay := time.Duration(base) * time.Second << uint(shift)
	if delay > maxRetryDelay {
		delay = maxRetryDelay
	}
	return time.Now().Add(delay)
}

// holdReason returns why a delivery must be held instead of sent, or "" to
// send it. Fan-out and the delivery worker both record it on the held
// delivery, so the UI can say why it was not sent.
func holdReason(webhook *store.WebhookRegistration, subscription *store.EventSubscription) string {
	switch {
	case webhook.AutoDisabledAt != nil && !webhook.Active:
		return "Held: webhook was auto-disabled; retry after resuming it"
	case !webhook.Active:
		return "Held: webhook is paused; retry after resuming it"
	case subscription != nil && subscription.Paused():
		if subscription.PausedReason != "" {
			return "Held: subscription is paused (" + subscription.PausedReason + "); retry after resuming it"
		}
		return "Held: subscription is paused; retry after resuming it"
	}
	return ""
}

// statusForFailure resolves the delivery status to record for a failed
// attempt: StatusRetrying while River still has attempts left, StatusFailed
// once this was the last one. Without this distinction "failed" isn't
// terminal — a later successful retry flips it back to "success", making
// "failed" an unreliable signal for anyone polling delivery status.
func statusForFailure(attempt, maxAttempts int) store.WebhookDeliveryStatus {
	if attempt < maxAttempts {
		return store.StatusRetrying
	}
	return store.StatusFailed
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

	// Parse job-arg IDs up front. Malformed args are permanent — cancel
	// instead of burning retries.
	tenantID, err := uuid.Parse(args.TenantID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("invalid tenant ID %q in job args: %w", args.TenantID, err))
	}
	webhookID, err := uuid.Parse(args.WebhookID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("invalid webhook ID %q in job args: %w", args.WebhookID, err))
	}
	deliveryID, err := uuid.Parse(args.DeliveryID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("invalid delivery ID %q in job args: %w", args.DeliveryID, err))
	}
	eventID, err := uuid.Parse(args.EventID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("invalid event ID %q in job args: %w", args.EventID, err))
	}

	// Get webhook configuration from database
	webhook, err := w.webhookRepo.GetWebhookByID(ctx, tenantID, webhookID, args.Consumer)
	if err != nil {
		w.logger.ErrorContext(ctx, "Failed to get webhook configuration", "error", err, "webhook_id", args.WebhookID)
		_ = w.deliveryRepo.UpdateDeliveryStatus(ctx, deliveryID, store.StatusFailed, 0, "", fmt.Sprintf("Failed to get webhook configuration: %v", err), "unknown")
		return fmt.Errorf("failed to get webhook configuration: %w", err)
	}

	// Get event record from database. The repo returns (nil, nil) for
	// missing rows.
	eventRecord, err := w.eventRepo.GetEventByID(ctx, tenantID, eventID)
	if err != nil {
		w.logger.ErrorContext(ctx, "Failed to get event record", "error", err, "event_id", args.EventID)
		_ = w.deliveryRepo.UpdateDeliveryStatus(ctx, deliveryID, store.StatusFailed, 0, "", fmt.Sprintf("Failed to get event record: %v", err), "unknown")
		return fmt.Errorf("failed to get event record: %w", err)
	}
	if eventRecord == nil {
		// Missing row is a permanent condition — retrying won't create it.
		w.logger.ErrorContext(ctx, "Event record not found", "event_id", args.EventID, "delivery_id", args.DeliveryID)
		_ = w.deliveryRepo.UpdateDeliveryStatus(ctx, deliveryID, store.StatusFailed, 0, "", "Event record not found", "unknown")
		return river.JobCancel(fmt.Errorf("event record %s not found", args.EventID))
	}

	// Get subscription if available
	var subscription *store.EventSubscription
	if args.SubscriptionID != "" {
		subscriptionID, parseErr := uuid.Parse(args.SubscriptionID)
		if parseErr != nil {
			w.logger.WarnContext(ctx, "Invalid subscription ID in job args", "error", parseErr, "subscription_id", args.SubscriptionID)
		} else {
			subscription, err = w.subscriptionRepo.GetSubscription(ctx, tenantID, subscriptionID)
			if err != nil {
				// If subscription is missing, we might still want to proceed if it's a legacy delivery,
				// but for now let's assume strict consistency or log warning.
				// Given the refactor, we expect subscription to exist if ID is passed.
				w.logger.WarnContext(ctx, "Failed to get subscription", "error", err, "subscription_id", args.SubscriptionID)
				// Continue without subscription (will use default webhook config)
			}
		}
	}

	ctx, span := w.tracer.Start(ctx, "webhook.delivery",
		trace.WithAttributes(
			attribute.String("delivery_id", args.DeliveryID),
			attribute.String("webhook_id", args.WebhookID),
			attribute.String("event_id", args.EventID),
			attribute.String("url", client.RedactURL(webhook.URL)),
			attribute.String("consumer", args.Consumer),
			attribute.String("event", eventRecord.Event),
		),
	)
	defer span.End()

	log := w.logger.With("job_id", job.ID, "delivery_id", args.DeliveryID, "webhook_id", args.WebhookID)

	// Check if the delivery has expired
	if time.Now().After(args.ExpiresAt) {
		span.SetStatus(otelcodes.Error, "webhook delivery expired")
		log.WarnContext(ctx, "Webhook delivery expired", "expires_at", args.ExpiresAt)

		err := w.deliveryRepo.UpdateDeliveryStatus(ctx, deliveryID, store.StatusExpired, 0, "", "Delivery expired", "unknown")
		if err != nil {
			log.ErrorContext(ctx, "Failed to update delivery status to expired", "error", err)
		}
		return nil
	}

	// A paused webhook or subscription (manual or automatic) holds its
	// deliveries instead of sending or failing them: the delivery becomes
	// paused, exactly like one fanned out during the pause, and is sent when
	// it is retried. Holding is a choice on the sending side, so it never
	// touches the webhook's health.
	if reason := holdReason(webhook, subscription); reason != "" {
		log.InfoContext(ctx, "Holding delivery as paused", "reason", reason)
		span.SetAttributes(attribute.String("pause_action", "held"))
		if err := w.deliveryRepo.HoldDelivery(ctx, deliveryID, reason); err != nil {
			return fmt.Errorf("hold paused delivery: %w", err)
		}
		return nil
	}

	log.InfoContext(ctx, "Processing webhook delivery", "event_id", args.EventID, "url", client.RedactURL(webhook.URL))

	// Render the payload before taking a rate-limit slot, so a template that
	// cannot render never consumes one.
	payloadBytes, templateFailed, err := w.renderPayload(ctx, log, job, webhook.RequiresTransform, subscription, eventRecord, deliveryID)
	if err != nil {
		return err
	}
	if templateFailed {
		// The delivery was failed with template_error; do not retry.
		return nil
	}

	// Rate limiting: check leaky bucket before sending.
	// AcquireDeliverySlot atomically advances the bucket and returns the slot
	// assigned to this delivery. If the slot is in the future, snooze the job.
	if webhook.RateLimitRPS != nil && *webhook.RateLimitRPS > 0 {
		nextDeliveryAt, rateLimitRPS, err := w.rateLimitRepo.AcquireDeliverySlot(ctx, webhookID)
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

	// Prepare delivery request using centralized client logic. A decrypt
	// failure of a configured secret is fail-closed: return the error so River
	// retries rather than delivering an unsigned/secret-less request.
	deliveryReq, err := client.PrepareDeliveryRequest(webhook, subscription, eventRecord, args.DeliveryID, payloadBytes, w.cryptoSvc)
	if err != nil {
		log.ErrorContext(ctx, "Failed to prepare delivery request", "error", err, "webhook_id", args.WebhookID, "delivery_id", args.DeliveryID)
		return fmt.Errorf("prepare delivery request: %w", err)
	}

	// Store the request body in the delivery record
	if err := w.deliveryRepo.UpdateDeliveryRequestBody(ctx, deliveryID, string(payloadBytes)); err != nil {
		log.WarnContext(ctx, "Failed to store request body", "error", err, "delivery_id", args.DeliveryID)
	}

	// Send the request. A listen session is never dialed: the CLI that
	// polls for it answers instead, and everything below handles its
	// response like an HTTP one.
	var resp *http.Response
	var duration time.Duration
	if client.IsListenURL(webhook.URL) {
		resp, duration, err = w.listen.Send(ctx, tenantID, deliveryReq)
	} else {
		resp, duration, err = w.client.Send(ctx, deliveryReq)
	}

	if err != nil {
		// Job-context cancellation (worker shutdown, job timeout) is not a
		// delivery failure — return the raw error so River retries without
		// marking the delivery permanently failed.
		if errors.Is(err, context.Canceled) ||
			(errors.Is(err, context.DeadlineExceeded) && ctx.Err() != nil) {
			log.WarnContext(ctx, "Webhook send interrupted by context cancellation, leaving for retry", "error", err)
			return err
		}

		// Classify the network/transport error
		errorCategory := sparrowerrors.ClassifyError(err)

		log.ErrorContext(ctx, "Failed to send webhook",
			"error", err,
			"duration_ms", duration.Milliseconds(),
			"error_category", string(errorCategory),
		)

		// Non-retryable categories (DNS, TLS) are terminal regardless of
		// attempts remaining; everything else is only "failed" once River
		// has exhausted retries.
		terminal := errorCategory != sparrowerrors.CategoryUnknown && !sparrowerrors.IsRetryableCategory(errorCategory)
		status := store.StatusFailed
		if !terminal {
			status = statusForFailure(job.Attempt, args.MaxAttempts)
		}
		_ = w.deliveryRepo.UpdateDeliveryStatus(ctx, deliveryID, status, 0, "", fmt.Sprintf("Request failed: %v", err), string(errorCategory))

		// Record health event and update health state
		w.recordHealthOutcome(ctx, log, tenantID, args.Consumer, webhookID, deliveryID, webhook.URL, false, int(duration.Milliseconds()), 0, err.Error(), string(errorCategory))
		if status == store.StatusFailed {
			w.emitDeliveryFailedEvent(ctx, log, tenantID, args.Consumer, webhookID, deliveryID, eventID, webhook.URL, job.Attempt, string(errorCategory), fmt.Sprintf("Request failed: %v", err))
		}

		// For non-retryable error categories (DNS, TLS), cancel River retries
		// by returning nil instead of an error. The delivery is already marked failed.
		// Unclassified transport errors (CategoryUnknown) are retried: an
		// unrecognized network failure is more likely transient than permanent.
		if terminal {
			log.WarnContext(ctx, "Non-retryable error category, cancelling retries",
				"error_category", string(errorCategory),
			)
			return nil
		}

		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body up to a storage limit. ReadBody drains a bounded
	// remainder afterwards so the keep-alive connection can be reused.
	// CaptureResponseBody controls the storage size limit:
	//   false (default) -> store up to 1 KB (useful for error diagnostics)
	//   true            -> store up to SPARROW_MAX_CAPTURED_RESPONSE_BYTES
	//                      (1 MiB by default; full response capture)
	var body []byte
	var bodyErr error
	if webhook.CaptureResponseBody {
		body, bodyErr = client.ReadBody(resp, w.captureLimit)
	} else {
		body, bodyErr = client.ReadBody(resp, 1024) // 1 KB — enough for error messages
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

		err := w.deliveryRepo.UpdateDeliveryStatus(ctx, deliveryID,
			store.StatusSuccess, resp.StatusCode, string(body), "", string(sparrowerrors.CategorySuccess))
		if err != nil {
			log.ErrorContext(ctx, "Failed to update delivery status to success", "error", err)
		}

		w.recordHealthOutcome(ctx, log, tenantID, args.Consumer, webhookID, deliveryID, webhook.URL, true, int(duration.Milliseconds()), resp.StatusCode, "", string(sparrowerrors.CategorySuccess))

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
		w.recordHealthOutcome(ctx, log, tenantID, args.Consumer, webhookID, deliveryID, webhook.URL, false,
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

	// Client errors (4xx) are terminal regardless of attempts remaining;
	// everything else is only "failed" once River has exhausted retries.
	terminal := !sparrowerrors.IsRetryableCategory(errorCategory)
	status := store.StatusFailed
	if !terminal {
		status = statusForFailure(job.Attempt, args.MaxAttempts)
	}

	err = w.deliveryRepo.UpdateDeliveryStatus(ctx, deliveryID,
		status, resp.StatusCode, string(body), errorMessage, string(errorCategory))
	if err != nil {
		log.ErrorContext(ctx, "Failed to update delivery status to failed", "error", err)
	}

	w.recordHealthOutcome(ctx, log, tenantID, args.Consumer, webhookID, deliveryID, webhook.URL, false, int(duration.Milliseconds()), resp.StatusCode, errorMessage, string(errorCategory))
	if status == store.StatusFailed {
		w.emitDeliveryFailedEvent(ctx, log, tenantID, args.Consumer, webhookID, deliveryID, eventID, webhook.URL, job.Attempt, string(errorCategory), errorMessage)
	}

	log.WarnContext(ctx, "Webhook delivery failed",
		"status_code", resp.StatusCode,
		"error_category", string(errorCategory),
		"duration_ms", duration.Milliseconds(),
	)

	// For client errors (4xx), do not retry - return nil to cancel River retries.
	// The delivery is already marked as failed with the appropriate error category.
	if terminal {
		log.WarnContext(ctx, "Non-retryable HTTP status, cancelling retries",
			"status_code", resp.StatusCode,
			"error_category", string(errorCategory),
		)
		return nil
	}

	return fmt.Errorf("webhook delivery failed: %s", errorMessage)
}

// renderPayload builds the body to send. Without a transform it is the
// default envelope. With one, a render error is recorded on the delivery and
// then handled per the subscription's on_transform_error:
//   - fail (default): the delivery is marked failed with error category
//     template_error and terminal is true, so River does not retry it. It is
//     retryable by hand once the template is fixed.
//   - fallback: the default envelope is sent instead.
//
// A webhook that requires a transform never receives the default envelope:
// a subscription without one (the API refuses that, so only legacy or
// hand-edited rows) fails the same way as a template error, and fallback
// fails instead of sending the envelope.
//
// A template error is a fault in the sender's configuration, not in the
// receiver, so it is never recorded as a health event: it does not affect the
// webhook's health or raise health or delivery-failed alerts.
func (w *WebhookWorker) renderPayload(ctx context.Context, log *slog.Logger, job *river.Job[WebhookArgs], requiresTransform bool, subscription *store.EventSubscription, eventRecord *store.EventRecord, deliveryID uuid.UUID) (payload []byte, terminal bool, err error) {
	args := job.Args
	envelope := func() ([]byte, bool, error) {
		body, err := client.BuildEnvelopePayload(args.EventID, eventRecord.Event, job.Attempt, eventRecord.Payload)
		if err != nil {
			log.ErrorContext(ctx, "Failed to marshal webhook payload", "error", err)
			return nil, false, err
		}
		return body, false, nil
	}

	fail := func(msg string) ([]byte, bool, error) {
		if err := w.deliveryRepo.UpdateDeliveryStatus(ctx, deliveryID, store.StatusFailed, 0, "",
			msg, string(sparrowerrors.CategoryTemplateError)); err != nil {
			log.ErrorContext(ctx, "Failed to mark delivery failed after template error", "error", err)
			return nil, false, fmt.Errorf("mark delivery failed: %w", err)
		}
		return nil, true, nil
	}

	if subscription == nil || !subscription.HasTransform() {
		if requiresTransform {
			msg := "This webhook requires a payload transform, but the subscription has none. Add a transform template, then retry the delivery."
			if err := w.deliveryRepo.UpdateDeliveryTemplateError(ctx, deliveryID, msg); err != nil {
				log.ErrorContext(ctx, "Failed to record template error", "error", err)
			}
			log.WarnContext(ctx, "Webhook requires a transform but the subscription has none, failing delivery",
				"subscription_id", args.SubscriptionID)
			return fail(msg)
		}
		return envelope()
	}

	body, renderErr := w.client.TransformPayloadWith(subscription.TransformTemplate, template.NewWebhookTemplateContext(
		args.EventID,
		eventRecord.Event,
		time.Now().UTC().Format(time.RFC3339),
		job.Attempt,
		eventRecord.Payload,
	), template.ExecOptions{StrictMissingKeys: subscription.StrictTemplate()})
	if renderErr == nil {
		return body, false, nil
	}

	msg := renderErr.Error()
	if err := w.deliveryRepo.UpdateDeliveryTemplateError(ctx, deliveryID, msg); err != nil {
		log.ErrorContext(ctx, "Failed to record template error", "error", err)
	}
	// A webhook that requires a transform never falls back to the envelope,
	// so its failures count as fail whatever the subscription says.
	mode := store.OnTransformErrorFail
	if subscription.FallbackOnTransformError() && !requiresTransform {
		mode = store.OnTransformErrorFallback
	}
	if w.templateErrors != nil {
		w.templateErrors.Add(ctx, 1, metric.WithAttributes(attribute.String("on_transform_error", mode)))
	}

	if mode == store.OnTransformErrorFallback {
		log.WarnContext(ctx, "Template transformation failed, sending envelope payload (on_transform_error=fallback)",
			"error", renderErr, "subscription_id", args.SubscriptionID)
		return envelope()
	}

	log.WarnContext(ctx, "Template transformation failed, failing delivery",
		"error", renderErr, "subscription_id", args.SubscriptionID, "requires_transform", requiresTransform)
	return fail("Template transformation failed: " + msg)
}

// recordHealthOutcome records a webhook health event and updates the health state.
// This is the shared implementation for all delivery outcome paths (success, client error, server error).
//
// Listen sessions are skipped entirely: a developer's local app failing says
// nothing about a production receiver, so it must not move health, metrics,
// alerts or auto-disable.
func (w *WebhookWorker) recordHealthOutcome(ctx context.Context, log *slog.Logger, tenantID uuid.UUID, consumer string, webhookID, deliveryID uuid.UUID, url string, success bool, durationMs int, statusCode int, errorMessage string, errorCategory string) {
	if client.IsListenURL(url) {
		return
	}
	w.metrics.recordAttempt(ctx, success, durationMs, errorCategory)
	if err := w.healthRepo.RecordWebhookHealthEvent(ctx, webhookID, deliveryID, success, durationMs, statusCode, errorMessage, errorCategory); err != nil {
		log.ErrorContext(ctx, "Failed to record health event", "error", err)
	}
	oldHealth, newHealth, err := w.healthRepo.UpdateWebhookHealthState(ctx, webhookID, success, time.Now())
	if err != nil {
		log.ErrorContext(ctx, "Failed to update webhook health state", "error", err)
		return
	}
	w.emitHealthChangedEvent(ctx, log, tenantID, consumer, webhookID, url, oldHealth, newHealth)
	if !success {
		w.maybeAutoDisable(ctx, log, tenantID, consumer, webhookID, url)
	}
}

// maybeAutoDisable pauses webhookID if its receiver has been failing long
// enough under w.autoDisable, then announces it with a
// sparrow.webhook.disabled system event. _sparrow's own webhooks are never
// auto-disabled: they carry the alerts that would report it.
func (w *WebhookWorker) maybeAutoDisable(ctx context.Context, log *slog.Logger, tenantID uuid.UUID, consumer string, webhookID uuid.UUID, url string) {
	if consumer == SystemEventConsumer || !w.autoDisable.enabled() {
		return
	}
	disabled, err := w.healthRepo.AutoDisableWebhook(ctx, webhookID, w.autoDisable.MinFailures, w.autoDisable.After)
	if err != nil {
		log.ErrorContext(ctx, "Failed to check webhook for auto-disable", "error", err)
		return
	}
	if disabled == nil {
		return
	}
	log.WarnContext(ctx, "Webhook auto-disabled after repeated failures",
		"reason", disabled.Reason,
		"consecutive_failures", disabled.ConsecutiveFailures,
		"failing_since", disabled.FailingSince)
	if w.metrics.autoDisabled != nil {
		w.metrics.autoDisabled.Add(ctx, 1)
	}
	w.emitWebhookDisabledEvent(ctx, log, tenantID, consumer, webhookID, url, disabled)
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
