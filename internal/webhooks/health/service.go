package health

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Service provides high-level webhook health management
type Service struct {
	calculator *HealthCalculator
	listener   *NotificationListener
	logger     *slog.Logger
}

// NewService creates a new webhook health service
func NewService(db *sqlx.DB, databaseURL string, logger *slog.Logger) *Service {
	calculator := NewHealthCalculator(db, logger)

	// Create notification handler that can interact with the calculator if needed
	handler := &HealthServiceHandler{
		calculator: calculator,
		logger:     logger,
	}

	listener := NewNotificationListener(databaseURL, handler, logger)

	return &Service{
		calculator: calculator,
		listener:   listener,
		logger:     logger,
	}
}

// Start starts the health service (notification listener)
func (s *Service) Start() error {
	// Initialize health states for all existing webhooks
	if err := s.calculator.InitializeWebhookHealthState(context.Background()); err != nil {
		s.logger.Error("Failed to initialize webhook health states", "error", err)
		// Don't fail startup for this
	}

	return s.listener.Start()
}

// Stop stops the health service
func (s *Service) Stop() error {
	return s.listener.Stop()
}

// RecordDeliveryResult records the result of a webhook delivery
func (s *Service) RecordDeliveryResult(ctx context.Context, webhookID string, deliveryID uuid.UUID, success bool, responseCode int, responseTimeMs int, errorMessage string) error {
	event := &WebhookHealthEvent{
		WebhookID:    webhookID,
		DeliveryID:   deliveryID,
		Success:      success,
		ResponseTime: responseTimeMs,
		ResponseCode: responseCode,
		ErrorMessage: errorMessage,
		Timestamp:    time.Now(),
	}

	return s.calculator.RecordHealthEvent(ctx, event)
}

// GetWebhookHealthState returns the current health state for a webhook
func (s *Service) GetWebhookHealthState(ctx context.Context, webhookID string) (*WebhookHealthState, error) {
	return s.calculator.GetWebhookHealthState(ctx, webhookID)
}

// AggregateHealthMetrics aggregates health metrics for the specified lookback period
func (s *Service) AggregateHealthMetrics(ctx context.Context, lookbackHours int) (int, error) {
	return s.calculator.AggregateHealthHourly(ctx, lookbackHours)
}

// HealthServiceHandler implements NotificationHandler for the health service
type HealthServiceHandler struct {
	calculator *HealthCalculator
	logger     *slog.Logger
}

// HandleWebhookHealthEvent processes webhook health event notifications
func (hsh *HealthServiceHandler) HandleWebhookHealthEvent(ctx context.Context, event *WebhookHealthEventNotification) error {
	hsh.logger.Debug("Processing webhook health event notification",
		"webhook_id", event.WebhookID,
		"success", event.Success,
		"response_code", event.ResponseCode)

	// For now, just log the event
	if !event.Success && event.ResponseCode >= 500 {
		hsh.logger.Warn("Webhook delivery failed with server error",
			"webhook_id", event.WebhookID,
			"response_code", event.ResponseCode,
			"response_time", event.ResponseTime)
	}

	return nil
}

// HealthMetrics provides aggregated health metrics for monitoring
type HealthMetrics struct {
	WebhookID           string    `json:"webhook_id"`
	TotalEvents         int       `json:"total_events"`
	SuccessfulEvents    int       `json:"successful_events"`
	FailedEvents        int       `json:"failed_events"`
	SuccessRate         float64   `json:"success_rate"`
	AvgResponseTime     int       `json:"avg_response_time_ms"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastEventAt         time.Time `json:"last_event_at"`
	Health              string    `json:"health"`
}

// GetHealthMetrics returns aggregated health metrics for a webhook
func (s *Service) GetHealthMetrics(ctx context.Context, webhookID string, hours int) (*HealthMetrics, error) {
	// Get basic stats from recent events
	var metrics HealthMetrics
	eventQuery := `
		SELECT 
			COUNT(*) as total_events,
			SUM(CASE WHEN success THEN 1 ELSE 0 END) as successful_events,
			SUM(CASE WHEN success THEN 0 ELSE 1 END) as failed_events,
			COALESCE(AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END), 0) as success_rate,
			COALESCE(AVG(response_time), 0) as avg_response_time
		FROM webhook_health_events 
		WHERE webhook_id = $1 
		  AND timestamp >= NOW() - INTERVAL '%d hours'`

	row := s.calculator.db.QueryRowContext(ctx, eventQuery, webhookID, hours)
	err := row.Scan(&metrics.TotalEvents, &metrics.SuccessfulEvents, &metrics.FailedEvents,
		&metrics.SuccessRate, &metrics.AvgResponseTime)
	if err != nil {
		return nil, err
	}

	metrics.WebhookID = webhookID

	// Get health state
	state, err := s.calculator.GetWebhookHealthState(ctx, webhookID)
	if err != nil {
		return nil, err
	}

	if state != nil {
		metrics.ConsecutiveFailures = state.ConsecutiveFailures
		if state.LastEventAt != nil {
			metrics.LastEventAt = *state.LastEventAt
		}
	}

	// Get current health status
	var health string
	healthQuery := `SELECT health FROM webhook_registrations WHERE id = $1`
	err = s.calculator.db.QueryRowContext(ctx, healthQuery, webhookID).Scan(&health)
	if err != nil {
		return nil, err
	}
	metrics.Health = health

	return &metrics, nil
}
