package webhooks

import (
	"context"
	"fmt"
	"time"

	otelcodes "go.opentelemetry.io/otel/codes"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

// WebhookHealthData represents webhook health information
type WebhookHealthData struct {
	WebhookID            string              `json:"webhook_id"`
	Health               store.WebhookHealth `json:"health"`
	TotalDeliveries      int                 `json:"total_deliveries"`
	SuccessfulDeliveries int                 `json:"successful_deliveries"`
	FailedDeliveries     int                 `json:"failed_deliveries"`
	ConsecutiveFailures  int                 `json:"consecutive_failures"`
	LastSuccessAt        *time.Time          `json:"last_success_at"`
	LastFailureAt        *time.Time          `json:"last_failure_at"`
	SuccessRate          float64             `json:"success_rate"`
	AvgResponseTime      int                 `json:"avg_response_time"` // milliseconds
	CreatedAt            time.Time           `json:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at"`

	// Error category breakdown
	ClientErrors           int `json:"client_errors"`            // 4xx responses
	ServerErrors           int `json:"server_errors"`            // 5xx responses
	TimeoutErrors          int `json:"timeout_errors"`           // Timeouts
	NetworkErrors          int `json:"network_errors"`           // DNS, TLS, connection refused, and other network errors
	UnexpectedStatusErrors int `json:"unexpected_status_errors"` // 2xx/3xx not in expected_status_codes
}

// HealthSummaryData represents health summary information
type HealthSummaryData struct {
	HealthyCount   int `json:"healthy_count"`
	DegradedCount  int `json:"degraded_count"`
	UnhealthyCount int `json:"unhealthy_count"`
	UnknownCount   int `json:"unknown_count"`
	TotalCount     int `json:"total_count"`
}

// NamespaceStatsData represents namespace statistics
type NamespaceStatsData struct {
	TotalWebhooks        int     `json:"total_webhooks"`
	ActiveWebhooks       int     `json:"active_webhooks"`
	TotalDeliveries      int     `json:"total_deliveries"`
	SuccessfulDeliveries int     `json:"successful_deliveries"`
	FailedDeliveries     int     `json:"failed_deliveries"`
	PendingDeliveries    int     `json:"pending_deliveries"`
	SuccessRate          float64 `json:"success_rate"`
}

// GetWebhookHealth retrieves health metrics for a webhook
func (s *WebhookService) GetWebhookHealth(ctx context.Context, webhookID string, namespace string) (*WebhookHealthData, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetWebhookHealth")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing get webhook health request",
		"webhook_id", webhookID,
		"namespace", namespace)

	// Validate required fields
	if webhookID == "" {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "webhook ID is required")
	}

	if namespace == "" {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "namespace is required")
	}

	tenantID := tenant.DefaultTenantID

	id, err := parseUUID(webhookID, "webhook ID")
	if err != nil {
		return nil, err
	}

	// Get webhook to verify it exists and get current health
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, tenantID, id, namespace)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "Failed to get webhook")
		s.logger.ErrorContext(ctx, "Failed to get webhook", "error", err)
		return nil, svcerrors.Wrapf(err, svcerrors.NotFound, "webhook not found")
	}

	// Get health state (current status and consecutive failures)
	healthState, err := s.webhookRepo.GetWebhookHealthState(ctx, id)
	if err != nil {
		// If no health state exists yet, return basic health info
		s.logger.InfoContext(ctx, "No health state found for webhook", "webhook_id", webhookID)
		return &WebhookHealthData{
			WebhookID: webhookID,
			Health:    webhook.Health,
		}, nil
	}

	// Get health summary for the last 24 hours
	healthSummary, err := s.webhookRepo.GetWebhookHealthSummary(ctx, id, 24)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get health summary", "error", err)
		// Continue with just the state info
	}

	// Convert to response format
	healthData := &WebhookHealthData{
		WebhookID:           webhookID,
		Health:              webhook.Health,
		ConsecutiveFailures: healthState.ConsecutiveFailures,
		LastSuccessAt:       healthState.LastSuccessAt,
		LastFailureAt:       healthState.LastFailureAt,
		CreatedAt:           healthState.CreatedAt,
		UpdatedAt:           healthState.UpdatedAt,
	}

	// Add summary data if available
	if healthSummary != nil {
		healthData.TotalDeliveries = healthSummary.TotalDeliveries
		healthData.SuccessfulDeliveries = healthSummary.SuccessfulDeliveries
		healthData.FailedDeliveries = healthSummary.FailedDeliveries
		healthData.SuccessRate = healthSummary.SuccessRate
		healthData.AvgResponseTime = healthSummary.AvgResponseTime
		healthData.ClientErrors = healthSummary.ClientErrors
		healthData.ServerErrors = healthSummary.ServerErrors
		healthData.TimeoutErrors = healthSummary.TimeoutErrors
		healthData.NetworkErrors = healthSummary.NetworkErrors
		healthData.UnexpectedStatusErrors = healthSummary.UnexpectedStatusErrors
	}

	s.logger.InfoContext(ctx, "Webhook health retrieved successfully",
		"webhook_id", webhookID,
		"health", webhook.Health,
		"success_rate", healthData.SuccessRate)

	return healthData, nil
}

// ListWebhooksByHealth retrieves webhooks filtered by health status
func (s *WebhookService) ListWebhooksByHealth(ctx context.Context, health store.WebhookHealth, limit, offset int32) ([]*store.WebhookRegistration, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListWebhooksByHealth")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing list webhooks by health request", "health", health, "limit", limit, "offset", offset)

	tenantID := tenant.DefaultTenantID

	// This is a cross-namespace query — only tenant-level roles can do this

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// Get webhooks by health status
	webhooksList, totalCount, err := s.webhookRepo.GetWebhooksByHealthPaginated(ctx, tenantID, health, int(limit), int(offset))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "Failed to get webhooks by health")
		s.logger.ErrorContext(ctx, "Failed to get webhooks by health", "error", err)
		return nil, 0, fmt.Errorf("failed to retrieve webhooks: %w", err)
	}

	s.logger.InfoContext(ctx, "Webhooks retrieved successfully",
		"health", health,
		"count", len(webhooksList),
		"total", totalCount)

	return webhooksList, int32(totalCount), nil
}

// GetHealthSummary retrieves a summary of webhook health across all namespaces
func (s *WebhookService) GetHealthSummary(ctx context.Context) (*HealthSummaryData, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetHealthSummary")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing get health summary request")

	tenantID := tenant.DefaultTenantID

	// Health summary is a cross-namespace query — only tenant-level roles can do this

	// Get health summary from repository
	summary, err := s.webhookRepo.GetHealthSummary(ctx, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "Failed to get health summary")
		s.logger.ErrorContext(ctx, "Failed to get health summary", "error", err)
		return nil, fmt.Errorf("failed to retrieve health summary: %w", err)
	}

	// Convert to response format
	healthSummary := &HealthSummaryData{
		HealthyCount:   summary[store.HealthHealthy],
		DegradedCount:  summary[store.HealthDegraded],
		UnhealthyCount: summary[store.HealthUnhealthy],
		UnknownCount:   summary[store.HealthUnknown],
	}

	// Calculate total
	healthSummary.TotalCount = healthSummary.HealthyCount + healthSummary.DegradedCount +
		healthSummary.UnhealthyCount + healthSummary.UnknownCount

	s.logger.InfoContext(ctx, "Health summary retrieved successfully",
		"healthy", healthSummary.HealthyCount,
		"degraded", healthSummary.DegradedCount,
		"unhealthy", healthSummary.UnhealthyCount,
		"unknown", healthSummary.UnknownCount,
		"total", healthSummary.TotalCount)

	return healthSummary, nil
}

// GetNamespaceStats retrieves statistics for a namespace, or across all namespaces if empty
func (s *WebhookService) GetNamespaceStats(ctx context.Context, namespace string) (*NamespaceStatsData, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetNamespaceStats")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing get namespace stats request", "namespace", namespace)

	tenantID := tenant.DefaultTenantID

	stats, err := s.webhookRepo.GetNamespaceStats(ctx, tenantID, namespace)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get namespace stats", "error", err)
		return nil, err
	}

	res := &NamespaceStatsData{
		TotalWebhooks:        stats.TotalWebhooks,
		ActiveWebhooks:       stats.ActiveWebhooks,
		TotalDeliveries:      stats.TotalDeliveries,
		SuccessfulDeliveries: stats.SuccessfulDeliveries,
		FailedDeliveries:     stats.FailedDeliveries,
		PendingDeliveries:    stats.PendingDeliveries,
		SuccessRate:          stats.SuccessRate,
	}

	s.logger.InfoContext(ctx, "Namespace stats retrieved successfully",
		"namespace", namespace,
		"total_webhooks", res.TotalWebhooks,
		"active_webhooks", res.ActiveWebhooks,
		"success_rate", res.SuccessRate)
	return res, nil
}
