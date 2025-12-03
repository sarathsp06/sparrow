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
		w.logger.Error("Failed to unmarshal job metadata", "error", unmarshallErr, "event_id", args.EventID)
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	// Get webhook configuration from database
	webhook, err := w.webhookRepo.GetWebhookByID(ctx, uuid.MustParse(args.WebhookID), args.Namespace)
	if err != nil {
		w.logger.Error("Failed to get webhook configuration", "error", err, "webhook_id", args.WebhookID)
		_ = w.webhookRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID), store.StatusFailed, 0, "", fmt.Sprintf("Failed to get webhook configuration: %v", err))
		return fmt.Errorf("failed to get webhook configuration: %w", err)
	}

	// Get event record from database
	eventRecord, err := w.webhookRepo.GetEventByID(ctx, uuid.MustParse(args.EventID))
	if err != nil {
		w.logger.Error("Failed to get event record", "error", err, "event_id", args.EventID)
		_ = w.webhookRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID), store.StatusFailed, 0, "", fmt.Sprintf("Failed to get event record: %v", err))
		return fmt.Errorf("failed to get event record: %w", err)
	}

	// Get subscription if available
	var subscription *store.EventSubscription
	if args.SubscriptionID != "" {
		subscription, err = w.webhookRepo.GetSubscription(ctx, uuid.MustParse(args.SubscriptionID))
		if err != nil {
			// If subscription is missing, we might still want to proceed if it's a legacy delivery,
			// but for now let's assume strict consistency or log warning.
			// Given the refactor, we expect subscription to exist if ID is passed.
			w.logger.Warn("Failed to get subscription", "error", err, "subscription_id", args.SubscriptionID)
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
		log.Warn("Webhook delivery expired", "expires_at", args.ExpiresAt)

		err := w.webhookRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID), store.StatusExpired, 0, "", "Delivery expired")
		if err != nil {
			log.Error("Failed to update delivery status to expired", "error", err)
		}
		return nil
	}

	log.Info("Processing webhook delivery", "event_id", args.EventID, "url", webhook.URL)

	// Prepare payload
	// Default payload structure
	defaultPayload := struct {
		EventID string         `json:"event_id"`
		Event   string         `json:"event"`
		Payload map[string]any `json:"payload"`
	}{
		EventID: args.EventID,
		Event:   eventRecord.Event,
		Payload: eventRecord.Payload,
	}

	var payloadBytes []byte

	// Check for transformation
	if subscription != nil && subscription.TransformEnabled && subscription.TransformTemplate != "" {
		payloadBytes, err = w.client.TransformPayload(subscription.TransformTemplate, client.WebhookTemplateContext{
			EventID:   args.EventID,
			EventName: eventRecord.Event,
			Payload:   eventRecord.Payload,
		})
		if err != nil {
			log.Error("Failed to transform payload", "error", err)
			_ = w.webhookRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID), store.StatusFailed, 0, "", fmt.Sprintf("Template transformation failed: %v", err))
			return fmt.Errorf("template transformation failed: %w", err)
		}
	} else {
		// Use default JSON payload
		payloadBytes, err = json.Marshal(defaultPayload)
		if err != nil {
			log.Error("Failed to marshal webhook payload", "error", err)
			return err
		}
	}

	// Prepare delivery request using centralized client logic
	deliveryReq := client.PrepareDeliveryRequest(webhook, subscription, eventRecord, args.DeliveryID, payloadBytes)

	// Store the request body in the delivery record
	if err := w.webhookRepo.UpdateDeliveryRequestBody(ctx, uuid.MustParse(args.DeliveryID), string(payloadBytes)); err != nil {
		log.Warn("Failed to store request body", "error", err, "delivery_id", args.DeliveryID)
	}

	// Send the request
	resp, duration, err := w.client.Send(ctx, deliveryReq)

	if err != nil {
		log.Error("Failed to send webhook",
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)

		_ = w.webhookRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID), store.StatusFailed, 0, "", fmt.Sprintf("Request failed: %v", err))

		// Record health event
		_ = w.webhookRepo.RecordWebhookHealthEvent(ctx, uuid.MustParse(args.WebhookID), uuid.MustParse(args.DeliveryID), false, int(duration.Milliseconds()), 0, err.Error())

		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	// We can use the client helper, but we need to respect the CaptureResponseBody flag
	var body []byte
	var bodyErr error
	if webhook.CaptureResponseBody {
		body, bodyErr = client.ReadBody(resp, 0) // No limit
	} else {
		body, bodyErr = client.ReadBody(resp, 1000) // Limit to 1000 chars
	}

	if bodyErr != nil {
		log.Warn("Failed to read response body", "error", bodyErr)
		body = []byte("Failed to read response body")
	}

	log.Info("Webhook response received",
		"status_code", resp.StatusCode,
		"duration_ms", duration.Milliseconds(),
	)

	// Check success status
	// We can use a helper in the client or keep logic here.
	// The logic depends on webhook.ExpectedStatusCodes which is in the webhook struct.
	// Let's keep the isSuccessStatusCode logic or move it to a shared utility.
	// For now, I'll inline a simplified check or re-implement the helper if I removed it.
	// Wait, I removed the helper method `isSuccessStatusCode` in this replacement.
	// I should probably keep it or move it to the client package.
	// Moving it to client package is cleaner.

	isSuccess := isSuccessStatusCode(resp.StatusCode, webhook.ExpectedStatusCodes)

	if isSuccess {
		span.SetStatus(otelcodes.Ok, "webhook delivered successfully")

		// Metrics are already recorded by the client!
		// But we might want to record worker-specific metrics if any.

		err := w.webhookRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID),
			store.StatusSuccess, resp.StatusCode, string(body), "")
		if err != nil {
			log.Error("Failed to update delivery status to success", "error", err)
		}

		_ = w.webhookRepo.RecordWebhookHealthEvent(ctx, uuid.MustParse(args.WebhookID), uuid.MustParse(args.DeliveryID), true, int(duration.Milliseconds()), resp.StatusCode, "")

		return nil
	}

	// Failure case
	errorMessage := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)
	span.SetStatus(otelcodes.Error, "webhook delivery failed")

	err = w.webhookRepo.UpdateDeliveryStatus(ctx, uuid.MustParse(args.DeliveryID),
		store.StatusFailed, resp.StatusCode, string(body), errorMessage)
	if err != nil {
		log.Error("Failed to update delivery status to failed", "error", err)
	}

	_ = w.webhookRepo.RecordWebhookHealthEvent(ctx, uuid.MustParse(args.WebhookID), uuid.MustParse(args.DeliveryID), false, int(duration.Milliseconds()), resp.StatusCode, errorMessage)

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
