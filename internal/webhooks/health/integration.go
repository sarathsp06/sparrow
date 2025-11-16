package health

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Example integration with webhook workers
// This shows how to use the health service in your existing webhook delivery code

// WebhookWorkerIntegration demonstrates how to integrate health tracking
type WebhookWorkerIntegration struct {
	healthService *Service
	logger        *slog.Logger
}

// NewWebhookWorkerIntegration creates a new integration helper
func NewWebhookWorkerIntegration(healthService *Service, logger *slog.Logger) *WebhookWorkerIntegration {
	return &WebhookWorkerIntegration{
		healthService: healthService,
		logger:        logger,
	}
}

// ProcessWebhookDelivery is an example of how to wrap webhook delivery with health tracking
func (wwi *WebhookWorkerIntegration) ProcessWebhookDelivery(ctx context.Context, webhookID string, deliveryID string, payload []byte) error {
	startTime := time.Now()

	// Your existing webhook delivery logic here
	success, responseCode, err := wwi.deliverWebhook(ctx, webhookID, payload)

	responseTime := int(time.Since(startTime).Milliseconds())

	// Record the health event
	var errorMessage string
	if err != nil {
		errorMessage = err.Error()
	}

	// Parse delivery ID to UUID
	deliveryUUID, parseErr := parseDeliveryID(deliveryID)
	if parseErr != nil {
		wwi.logger.Error("Failed to parse delivery ID", "delivery_id", deliveryID, "error", parseErr)
		// Still record the health event with a new UUID
		deliveryUUID = generateNewUUID()
	}

	// Record the health result - this will trigger health calculation and notifications
	healthErr := wwi.healthService.RecordDeliveryResult(ctx, webhookID, deliveryUUID, success, responseCode, responseTime, errorMessage)
	if healthErr != nil {
		wwi.logger.Error("Failed to record health result",
			"webhook_id", webhookID,
			"delivery_id", deliveryID,
			"error", healthErr)
	}

	return err
}

// deliverWebhook simulates your existing webhook delivery logic
func (wwi *WebhookWorkerIntegration) deliverWebhook(ctx context.Context, webhookID string, payload []byte) (bool, int, error) {
	// This is where your actual webhook delivery code would go
	// For example:

	// 1. Get webhook configuration from database
	// 2. Make HTTP request to webhook URL
	// 3. Handle response
	// 4. Return success status, response code, and any error

	// Placeholder implementation:
	// return httpClient.Post(webhookURL, payload)

	return true, 200, nil
}

// Example of how to use this in your webhook worker:
/*
func (w *WebhookWorker) processDelivery(ctx context.Context, delivery *WebhookDelivery) error {
	integration := health.NewWebhookWorkerIntegration(w.healthService, w.logger)
	return integration.ProcessWebhookDelivery(ctx, delivery.WebhookID, delivery.ID, delivery.Payload)
}
*/

// Example of setting up the health service in your main application:
/*
func setupHealthService(db *sqlx.DB, databaseURL string, logger *slog.Logger) (*health.Service, error) {
	healthService := health.NewService(db, databaseURL, logger)

	// Start the notification listener
	if err := healthService.Start(); err != nil {
		return nil, fmt.Errorf("failed to start health service: %w", err)
	}

	// Set up periodic aggregation (optional)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				processed, err := healthService.AggregateHealthMetrics(context.Background(), 2)
				if err != nil {
					logger.Error("Failed to aggregate health metrics", "error", err)
				} else {
					logger.Info("Aggregated health metrics", "processed_count", processed)
				}
			}
		}
	}()

	return healthService, nil
}
*/

// Helper functions (you would implement these based on your system)
func parseDeliveryID(deliveryID string) (uuid.UUID, error) {
	// Implementation depends on your delivery ID format
	return uuid.Parse(deliveryID)
}

func generateNewUUID() uuid.UUID {
	return uuid.New()
}
