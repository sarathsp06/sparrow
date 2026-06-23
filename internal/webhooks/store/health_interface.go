package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// HealthRepository defines operations for webhook health tracking.
type HealthRepository interface {
	UpdateWebhookHealthState(ctx context.Context, webhookID uuid.UUID, success bool, eventTimestamp time.Time) error
	CalculateWebhookHealth(ctx context.Context, webhookID uuid.UUID, lookbackHours int) (string, error)
	RecordWebhookHealthEvent(ctx context.Context, webhookID, deliveryID uuid.UUID, success bool, responseTime, responseCode int, errorMessage string, errorCategory string) error
	GetDeliveryAttempts(ctx context.Context, tenantID uuid.UUID, deliveryID uuid.UUID) ([]*WebhookHealthEvent, error)
	GetWebhookHealthState(ctx context.Context, webhookID uuid.UUID) (*WebhookHealthMetrics, error)
	GetWebhookHealthSummary(ctx context.Context, webhookID uuid.UUID, hours int) (*WebhookHealthSummary, error)
	GetWebhookHealthTimeSeries(ctx context.Context, webhookID uuid.UUID, hours int, bucketSize string) ([]*WebhookHealthEvent, error)
	AggregateHealthSummaries(ctx context.Context) (int, error)
	GetWebhooksByHealth(ctx context.Context, tenantID uuid.UUID, health WebhookHealth) ([]*WebhookRegistration, error)
	GetWebhooksByHealthPaginated(ctx context.Context, tenantID uuid.UUID, health WebhookHealth, limit, offset int) ([]*WebhookRegistration, int, error)
	GetHealthSummary(ctx context.Context, tenantID uuid.UUID) (map[WebhookHealth]int, error)
	GetNamespaceStats(ctx context.Context, tenantID uuid.UUID, namespace string) (*NamespaceStats, error)
}
