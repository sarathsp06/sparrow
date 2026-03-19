package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	sparrowerrors "github.com/sarathsp06/sparrow/pkg/errors"

	"github.com/sarathsp06/sparrow/internal/logger"
	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// WebhookWorker handles webhook delivery jobs
type WebhookWorker struct {
	river.WorkerDefaults[WebhookArgs]
	webhookRepo store.RepositoryInterface
	tracer      trace.Tracer
	logger      *slog.Logger
	metrics     *observability.SparrowMetrics
	client      *client.WebhookClient
}

// NewWebhookWorker creates a new webhook worker
func NewWebhookWorker(webhookRepo store.RepositoryInterface) *WebhookWorker {
	metrics, err := observability.NewSparrowMetrics()
	if err != nil {
		// Log error but continue without metrics
		log := logger.NewLogger("webhook-worker")
		log.Error("Failed to initialize metrics", "error", err)
	}

	// Initialize the centralized webhook client
	// In a real app, we might pass config here
	webhookClient := client.NewWebhookClient(nil)

	return &WebhookWorker{
		webhookRepo: webhookRepo,
		logger:      logger.NewLogger("webhook-worker"),
		tracer:      observability.GetTracer("sparrow.workers.webhook"),
		metrics:     metrics,
		client:      webhookClient,
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
		_ = w.webhookRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID), store.StatusFailed, 0, "", fmt.Sprintf("Failed to get webhook configuration: %v", err), "unknown")
		return fmt.Errorf("failed to get webhook configuration: %w", err)
	}

	// Get event record from database
	eventRecord, err := w.webhookRepo.GetEventByID(ctx, tenantID, uuid.MustParse(args.EventID))
	if err != nil {
		w.logger.ErrorContext(ctx, "Failed to get event record", "error", err, "event_id", args.EventID)
		_ = w.webhookRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID), store.StatusFailed, 0, "", fmt.Sprintf("Failed to get event record: %v", err), "unknown")
		return fmt.Errorf("failed to get event record: %w", err)
	}

	// Get subscription if available
	var subscription *store.EventSubscription
	if args.SubscriptionID != "" {
		subscription, err = w.webhookRepo.GetSubscription(ctx, tenantID, uuid.MustParse(args.SubscriptionID))
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

		err := w.webhookRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID), store.StatusExpired, 0, "", "Delivery expired", "unknown")
		if err != nil {
			log.ErrorContext(ctx, "Failed to update delivery status to expired", "error", err)
		}
		return nil
	}

	log.InfoContext(ctx, "Processing webhook delivery", "event_id", args.EventID, "url", webhook.URL)

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
			log.ErrorContext(ctx, "Failed to transform payload", "error", err)
			_ = w.webhookRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID), store.StatusFailed, 0, "", fmt.Sprintf("Template transformation failed: %v", err), "unknown")
			return fmt.Errorf("template transformation failed: %w", err)
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

	// Prepare delivery request using centralized client logic
	deliveryReq := client.PrepareDeliveryRequest(webhook, subscription, eventRecord, args.DeliveryID, payloadBytes)

	// Store the request body in the delivery record
	if err := w.webhookRepo.UpdateDeliveryRequestBody(ctx, uuid.MustParse(args.DeliveryID), string(payloadBytes)); err != nil {
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

		_ = w.webhookRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID), store.StatusFailed, 0, "", fmt.Sprintf("Request failed: %v", err), string(errorCategory))

		// Record health event and update health state
		webhookUUID := uuid.MustParse(args.WebhookID)
		if healthErr := w.webhookRepo.RecordWebhookHealthEvent(ctx, webhookUUID, uuid.MustParse(args.DeliveryID), false, int(duration.Milliseconds()), 0, err.Error(), string(errorCategory)); healthErr != nil {
			log.ErrorContext(ctx, "Failed to record health event", "error", healthErr)
		}
		if healthErr := w.webhookRepo.UpdateWebhookHealthState(ctx, webhookUUID, false, time.Now()); healthErr != nil {
			log.ErrorContext(ctx, "Failed to update webhook health state", "error", healthErr)
		}

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

	// Read response body
	// We can use the client helper, but we need to respect the CaptureResponseBody flag
	const maxResponseBodyBytes = 1024 * 1024 // 1 MB cap to prevent OOM from malicious endpoints
	var body []byte
	var bodyErr error
	if webhook.CaptureResponseBody {
		body, bodyErr = client.ReadBody(resp, maxResponseBodyBytes)
	} else {
		body, bodyErr = client.ReadBody(resp, 1000) // Limit to 1000 chars
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

		err := w.webhookRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID),
			store.StatusSuccess, resp.StatusCode, string(body), "", string(sparrowerrors.CategorySuccess))
		if err != nil {
			log.ErrorContext(ctx, "Failed to update delivery status to success", "error", err)
		}

		webhookUUID := uuid.MustParse(args.WebhookID)
		if healthErr := w.webhookRepo.RecordWebhookHealthEvent(ctx, webhookUUID, uuid.MustParse(args.DeliveryID), true, int(duration.Milliseconds()), resp.StatusCode, "", string(sparrowerrors.CategorySuccess)); healthErr != nil {
			log.ErrorContext(ctx, "Failed to record health event", "error", healthErr)
		}
		if healthErr := w.webhookRepo.UpdateWebhookHealthState(ctx, webhookUUID, true, time.Now()); healthErr != nil {
			log.ErrorContext(ctx, "Failed to update webhook health state", "error", healthErr)
		}

		return nil
	}

	// Failure case - classify the HTTP status code error
	errorCategory := sparrowerrors.ClassifyHTTPStatus(resp.StatusCode)
	errorMessage := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)
	span.SetStatus(otelcodes.Error, "webhook delivery failed")
	span.SetAttributes(attribute.String("error_category", string(errorCategory)))

	err = w.webhookRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID),
		store.StatusFailed, resp.StatusCode, string(body), errorMessage, string(errorCategory))
	if err != nil {
		log.ErrorContext(ctx, "Failed to update delivery status to failed", "error", err)
	}

	webhookUUID := uuid.MustParse(args.WebhookID)
	if healthErr := w.webhookRepo.RecordWebhookHealthEvent(ctx, webhookUUID, uuid.MustParse(args.DeliveryID), false, int(duration.Milliseconds()), resp.StatusCode, errorMessage, string(errorCategory)); healthErr != nil {
		log.ErrorContext(ctx, "Failed to record health event", "error", healthErr)
	}
	if healthErr := w.webhookRepo.UpdateWebhookHealthState(ctx, webhookUUID, false, time.Now()); healthErr != nil {
		log.ErrorContext(ctx, "Failed to update webhook health state", "error", healthErr)
	}

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
