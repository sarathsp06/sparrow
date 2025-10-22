package workers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sarathsp06/sparrow/internal/jobs"
	"github.com/sarathsp06/sparrow/internal/logger"
	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/webhooks"
)

// WebhookWorker handles webhook delivery jobs
type WebhookWorker struct {
	river.WorkerDefaults[jobs.WebhookArgs]
	webhookRepo *webhooks.Repository
	tracer      trace.Tracer
	metrics     *observability.SparrowMetrics
}

// NewWebhookWorker creates a new webhook worker
func NewWebhookWorker(webhookRepo *webhooks.Repository) *WebhookWorker {
	metrics, err := observability.NewSparrowMetrics()
	if err != nil {
		// Log error but continue without metrics
		log := logger.NewLogger("webhook-worker")
		log.Error("Failed to initialize metrics", "error", err)
	}

	return &WebhookWorker{
		webhookRepo: webhookRepo,
		tracer:      observability.GetTracer("sparrow.workers.webhook"),
		metrics:     metrics,
	}
}

// Work processes the webhook delivery job
func (w *WebhookWorker) Work(ctx context.Context, job *river.Job[jobs.WebhookArgs]) error {
	args := job.Args

	ctx, span := w.tracer.Start(ctx, "webhook.delivery",
		trace.WithAttributes(
			attribute.String("delivery_id", args.DeliveryID),
			attribute.String("webhook_id", args.WebhookID),
			attribute.String("event_id", args.EventID),
			attribute.String("url", args.URL),
			attribute.String("namespace", args.Namespace),
			attribute.String("event", args.Event),
		),
	)
	defer span.End()

	log := logger.NewLogger("webhook-worker")

	// Check if the delivery has expired
	if time.Now().After(args.ExpiresAt) {
		span.SetStatus(otelcodes.Error, "webhook delivery expired")
		log.Warn("Webhook delivery expired",
			"job_id", job.ID,
			"delivery_id", args.DeliveryID,
			"webhook_id", args.WebhookID,
			"expires_at", args.ExpiresAt,
		)

		err := w.webhookRepo.UpdateDeliveryStatus(ctx, args.DeliveryID,
			webhooks.StatusExpired, 0, "", "Delivery expired")
		if err != nil {
			log.Error("Failed to update delivery status to expired", "error", err)
		}
		return fmt.Errorf("webhook delivery expired")
	}

	log.Info("Processing webhook delivery",
		"job_id", job.ID,
		"delivery_id", args.DeliveryID,
		"webhook_id", args.WebhookID,
		"event_id", args.EventID,
		"url", args.URL,
		"method", "POST",
		"namespace", args.Namespace,
		"event", args.Event,
	)

	// Update delivery status to sending
	if err := w.webhookRepo.UpdateDeliveryStatus(ctx, args.DeliveryID,
		webhooks.StatusSending, 0, "", ""); err != nil {
		log.Error("Failed to update delivery status to sending", "error", err)
	}

	// Create HTTP request (always POST for webhooks)
	req, err := http.NewRequestWithContext(ctx, "POST", args.URL, bytes.NewBuffer([]byte(args.Payload)))
	if err != nil {
		log.Error("Failed to create request",
			"job_id", job.ID,
			"delivery_id", args.DeliveryID,
			"url", args.URL,
			"method", "POST",
			"error", err,
		)

		w.webhookRepo.UpdateDeliveryStatus(ctx, args.DeliveryID,
			webhooks.StatusFailed, 0, "", fmt.Sprintf("Failed to create request: %v", err))
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set default Content-Type
	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range args.Headers {
		req.Header.Set(key, value)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(args.Timeout) * time.Second,
	}

	// Send the request
	startTime := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		log.Error("Failed to send webhook",
			"job_id", job.ID,
			"delivery_id", args.DeliveryID,
			"url", args.URL,
			"method", "POST",
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)

		w.webhookRepo.UpdateDeliveryStatus(ctx, args.DeliveryID,
			webhooks.StatusFailed, 0, "", fmt.Sprintf("Request failed: %v", err))
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Read response body (limit to first 1000 chars for logging)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1000))
	if err != nil {
		log.Warn("Failed to read response body", "error", err)
		body = []byte("Failed to read response body")
	}

	log.Info("Webhook response received",
		"job_id", job.ID,
		"delivery_id", args.DeliveryID,
		"url", args.URL,
		"method", "POST",
		"status_code", resp.StatusCode,
		"status", resp.Status,
		"duration_ms", duration.Milliseconds(),
	)

	// Consider 2xx status codes as success
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		span.SetAttributes(
			attribute.Int("status_code", resp.StatusCode),
			attribute.Float64("duration_seconds", duration.Seconds()),
		)
		span.SetStatus(otelcodes.Ok, "webhook delivered successfully")

		// Record metrics
		if w.metrics != nil {
			w.metrics.WebhookDeliveries.Add(ctx, 1)
			w.metrics.DeliveryDuration.Record(ctx, duration.Seconds())
		}

		log.Info("Webhook delivered successfully",
			"job_id", job.ID,
			"delivery_id", args.DeliveryID,
			"url", args.URL,
			"status_code", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
		)

		err := w.webhookRepo.UpdateDeliveryStatus(ctx, args.DeliveryID,
			webhooks.StatusSuccess, resp.StatusCode, string(body), "")
		if err != nil {
			log.Error("Failed to update delivery status to success", "error", err)
		}
		return nil
	}

	// For non-2xx responses, update status and return error for retry
	errorMessage := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)

	span.SetAttributes(
		attribute.Int("status_code", resp.StatusCode),
		attribute.Float64("duration_seconds", duration.Seconds()),
	)
	span.RecordError(fmt.Errorf("webhook delivery failed: %s", errorMessage))
	span.SetStatus(otelcodes.Error, "webhook delivery failed")

	// Record metrics
	if w.metrics != nil {
		w.metrics.WebhookDeliveries.Add(ctx, 1)
		w.metrics.DeliveryDuration.Record(ctx, duration.Seconds())
	}

	log.Warn("Webhook delivery failed",
		"job_id", job.ID,
		"delivery_id", args.DeliveryID,
		"url", args.URL,
		"status_code", resp.StatusCode,
		"status", resp.Status,
		"duration_ms", duration.Milliseconds(),
	)

	err = w.webhookRepo.UpdateDeliveryStatus(ctx, args.DeliveryID,
		webhooks.StatusFailed, resp.StatusCode, string(body), errorMessage)
	if err != nil {
		log.Error("Failed to update delivery status to failed", "error", err)
	}

	return fmt.Errorf("webhook delivery failed: %s", errorMessage)
}
