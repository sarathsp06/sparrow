package queue

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/riverqueue/river"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/sarathsp06/sparrow/internal/logger"
	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// WebhookWorker handles webhook delivery jobs
type WebhookWorker struct {
	river.WorkerDefaults[WebhookArgs]
	webhookRepo store.RepositoryInterface
	tracer      trace.Tracer
	logger      *slog.Logger
	metrics     *observability.SparrowMetrics
}

// NewWebhookWorker creates a new webhook worker
func NewWebhookWorker(webhookRepo store.RepositoryInterface) *WebhookWorker {
	metrics, err := observability.NewSparrowMetrics()
	if err != nil {
		// Log error but continue without metrics
		log := logger.NewLogger("webhook-worker")
		log.Error("Failed to initialize metrics", "error", err)
	}

	return &WebhookWorker{
		webhookRepo: webhookRepo,
		logger:      logger.NewLogger("event-processing-worker"),

		tracer:  observability.GetTracer("sparrow.workers.webhook"),
		metrics: metrics,
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
	webhook, err := w.webhookRepo.GetWebhookByID(ctx, args.WebhookID, args.Namespace)
	if err != nil {
		w.logger.Error("Failed to get webhook configuration", "error", err, "webhook_id", args.WebhookID)
		_ = w.webhookRepo.UpdateDeliveryStatus(ctx, args.DeliveryID, store.StatusFailed, 0, "", fmt.Sprintf("Failed to get webhook configuration: %v", err))
		return fmt.Errorf("failed to get webhook configuration: %w", err)
	}

	// Get event record from database
	eventRecord, err := w.webhookRepo.GetEventByID(ctx, args.EventID)
	if err != nil {
		w.logger.Error("Failed to get event record", "error", err, "event_id", args.EventID)
		_ = w.webhookRepo.UpdateDeliveryStatus(ctx, args.DeliveryID, store.StatusFailed, 0, "", fmt.Sprintf("Failed to get event record: %v", err))
		return fmt.Errorf("failed to get event record: %w", err)
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

		err := w.webhookRepo.UpdateDeliveryStatus(ctx, args.DeliveryID, store.StatusExpired, 0, "", "Delivery expired")
		if err != nil {
			log.Error("Failed to update delivery status to expired", "error", err)
		}
		return nil
	}

	log.Info("Processing webhook delivery", "event_id", args.EventID, "url", webhook.URL, "method", http.MethodPost, "namespace", args.Namespace, "event", eventRecord.Event)

	// Create webhook payload using event data from database
	webhookPayload := struct {
		EventID string         `json:"event_id"`
		Event   string         `json:"event"`
		Payload map[string]any `json:"payload"`
	}{
		EventID: args.EventID,
		Event:   eventRecord.Event,
		Payload: eventRecord.Payload,
	}
	payloadBytes, err := json.Marshal(webhookPayload)
	if err != nil {
		log.Error("Failed to marshal webhook payload", "error", err)
		return err
	}

	// Create HTTP request (always POST for webhooks)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Error("Failed to create request", "url", webhook.URL, "error", err)

		_ = w.webhookRepo.UpdateDeliveryStatus(ctx, args.DeliveryID, store.StatusFailed, 0, "", fmt.Sprintf("Failed to create request: %v", err))
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Store the request body in the delivery record
	if err := w.webhookRepo.UpdateDeliveryRequestBody(ctx, args.DeliveryID, string(payloadBytes)); err != nil {
		log.Warn("Failed to store request body", "error", err, "delivery_id", args.DeliveryID)
	}

	// Set Content-Type from webhook configuration or default
	contentType := "application/json"
	if webhook.ContentType != "" {
		contentType = webhook.ContentType
	}
	req.Header.Set("Content-Type", contentType)

	// Set User-Agent from webhook configuration or default
	userAgent := "Sparrow-Webhook/1.0"
	if webhook.UserAgent != "" {
		userAgent = webhook.UserAgent
	}
	req.Header.Set("User-Agent", userAgent)

	// Add HMAC signature if webhook secret is configured
	if webhook.WebhookSecret != "" {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		signature := w.generateHMACSignature(payloadBytes, webhook.WebhookSecret, timestamp)

		req.Header.Set("X-Sparrow-Signature-256", "sha256="+signature)
		req.Header.Set("X-Sparrow-Timestamp", timestamp)
	}

	// Add custom headers from webhook configuration
	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}

	// Create HTTP client with advanced configuration from webhook settings
	client := w.createConfiguredHTTPClient(webhook)

	// Send the request
	startTime := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		log.Error("Failed to send webhook",
			"job_id", job.ID,
			"delivery_id", args.DeliveryID,
			"url", webhook.URL,
			"method", "POST",
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)

		_ = w.webhookRepo.UpdateDeliveryStatus(ctx, args.DeliveryID, store.StatusFailed, 0, "", fmt.Sprintf("Request failed: %v", err))
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	// Read response body based on webhook configuration
	var body []byte
	var bodyErr error
	if webhook.CaptureResponseBody {
		// Read full response body when capture is enabled
		body, bodyErr = io.ReadAll(resp.Body)
		if bodyErr != nil {
			log.Warn("Failed to read response body", "error", bodyErr)
			body = []byte("Failed to read response body")
		}
	} else {
		// Read limited response body for logging (first 1000 chars)
		body, bodyErr = io.ReadAll(io.LimitReader(resp.Body, 1000))
		if bodyErr != nil {
			log.Warn("Failed to read response body", "error", bodyErr)
			body = []byte("Failed to read response body")
		}
	}

	log.Info("Webhook response received",
		"job_id", job.ID,
		"delivery_id", args.DeliveryID,
		"url", webhook.URL,
		"method", "POST",
		"status_code", resp.StatusCode,
		"status", resp.Status,
		"duration_ms", duration.Milliseconds(),
	)

	// Check if status code is in expected range from webhook configuration
	isSuccess := w.isSuccessStatusCode(resp.StatusCode, webhook.ExpectedStatusCodes)
	if isSuccess {
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
			"url", webhook.URL,
			"status_code", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
		)

		err := w.webhookRepo.UpdateDeliveryStatus(ctx, args.DeliveryID,
			store.StatusSuccess, resp.StatusCode, string(body), "")
		if err != nil {
			log.Error("Failed to update delivery status to success", "error", err)
		}

		// Record health event for successful delivery
		if err := w.webhookRepo.RecordWebhookHealthEvent(ctx, args.WebhookID, args.DeliveryID, true, int(duration.Milliseconds()), resp.StatusCode, ""); err != nil {
			log.Error("Failed to record webhook health event", "error", err)
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
		"url", webhook.URL,
		"status_code", resp.StatusCode,
		"status", resp.Status,
		"duration_ms", duration.Milliseconds(),
	)

	err = w.webhookRepo.UpdateDeliveryStatus(ctx, args.DeliveryID,
		store.StatusFailed, resp.StatusCode, string(body), errorMessage)
	if err != nil {
		log.Error("Failed to update delivery status to failed", "error", err)
	}

	// Record health event for failed delivery
	if err := w.webhookRepo.RecordWebhookHealthEvent(ctx, args.WebhookID, args.DeliveryID, false, int(duration.Milliseconds()), resp.StatusCode, errorMessage); err != nil {
		log.Error("Failed to record webhook health event", "error", err)
	}

	return fmt.Errorf("webhook delivery failed: %s", errorMessage)
}

// createConfiguredHTTPClient creates an HTTP client configured with webhook settings
func (w *WebhookWorker) createConfiguredHTTPClient(webhook *store.WebhookRegistration) *http.Client {
	// Create transport with webhook-specific configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !webhook.VerifySSL,
		},
		DisableCompression: false,
		ForceAttemptHTTP2:  false,
	}

	// Configure redirect policy
	var checkRedirect func(req *http.Request, via []*http.Request) error
	if !webhook.FollowRedirects {
		checkRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// Create client with timeout based on webhook configuration
	timeout := time.Duration(webhook.Timeout) * time.Second
	if webhook.RequestTimeoutSeconds > 0 {
		timeout = time.Duration(webhook.RequestTimeoutSeconds) * time.Second
	}

	return &http.Client{
		Transport:     otelhttp.NewTransport(transport),
		Timeout:       timeout,
		CheckRedirect: checkRedirect,
	}
}

// isSuccessStatusCode checks if a status code is considered successful based on webhook configuration
func (w *WebhookWorker) isSuccessStatusCode(statusCode int, expectedStatusCodes []int64) bool {
	// If no expected status codes are configured, default to 2xx success
	if len(expectedStatusCodes) == 0 {
		return statusCode >= 200 && statusCode < 300
	}

	// Check if status code matches any of the expected codes
	for _, expected := range expectedStatusCodes {
		if statusCode == int(expected) {
			return true
		}
	}

	// Also check for ranges (e.g., 200-299)
	for _, expected := range expectedStatusCodes {
		expectedStr := strconv.FormatInt(expected, 10)
		if len(expectedStr) == 3 {
			// Single status code (e.g., 200)
			if statusCode == int(expected) {
				return true
			}
		} else if len(expectedStr) == 2 {
			// Range check (e.g., 20 means 200-209)
			rangeStart := int(expected) * 10
			rangeEnd := rangeStart + 9
			if statusCode >= rangeStart && statusCode <= rangeEnd {
				return true
			}
		} else if len(expectedStr) == 1 {
			// Broader range (e.g., 2 means 200-299)
			rangeStart := int(expected) * 100
			rangeEnd := rangeStart + 99
			if statusCode >= rangeStart && statusCode <= rangeEnd {
				return true
			}
		}
	}

	return false
}

// generateHMACSignature generates an HMAC-SHA256 signature for webhook payload verification
// The signature is created by HMAC-SHA256(secret, timestamp.payload) and hex-encoded.
// Webhook receivers should verify the signature using the same method:
// 1. Extract timestamp from X-Sparrow-Timestamp header
// 2. Extract signature from X-Sparrow-Signature-256 header (remove "sha256=" prefix)
// 3. Compute HMAC-SHA256(secret, timestamp.payload)
// 4. Compare computed signature with received signature (use constant-time comparison)
// 5. Optionally check timestamp to prevent replay attacks (e.g., reject if older than 5 minutes)
func (w *WebhookWorker) generateHMACSignature(payload []byte, secret, timestamp string) string {
	// Create the message to sign: timestamp.payload
	message := timestamp + "." + string(payload)

	// Create HMAC hash
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))

	// Return hex encoded signature
	return hex.EncodeToString(h.Sum(nil))
}
