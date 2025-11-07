package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// RepositoryInterface defines the interface for webhook storage operations
//
//go:generate gowrap gen -i RepositoryInterface -t opentelemetry -o RepositoryInterface_otel.go
type RepositoryInterface interface {
	UpdateWebhookHealthState(ctx context.Context, webhookID string, success bool, eventTimestamp time.Time) error
	CalculateWebhookHealth(ctx context.Context, webhookID string, lookbackHours int) (string, error)
	StoreEventTx(ctx context.Context, tx pgx.Tx, event *EventRecord) error
	GetWebhooksByEventTx(ctx context.Context, tx pgx.Tx, namespace, event string) ([]*WebhookRegistration, error)
	CreateDeliveryTx(ctx context.Context, tx pgx.Tx, delivery *WebhookDelivery) error
	RegisterWebhook(ctx context.Context, registration *WebhookRegistration) error
	UnregisterWebhook(ctx context.Context, webhookID string) error
	GetWebhooksByEvent(ctx context.Context, namespace, event string) ([]*WebhookRegistration, error)
	ListWebhooks(ctx context.Context, namespace string, activeOnly bool) ([]*WebhookRegistration, error)
	StoreEvent(ctx context.Context, event *EventRecord) error
	CreateDelivery(ctx context.Context, delivery *WebhookDelivery) error
	UpdateDeliveryStatus(ctx context.Context, deliveryID string, status WebhookDeliveryStatus, responseCode int, responseBody, errorMessage string) error
	GetDeliveriesByWebhook(ctx context.Context, webhookID string) ([]*WebhookDelivery, error)
	GetDeliveriesByEvent(ctx context.Context, eventID string) ([]*WebhookDelivery, error)
	GetWebhookByID(ctx context.Context, webhookID, namespace string) (*WebhookRegistration, error)
	GetWebhooksByNamespace(ctx context.Context, namespace string, activeOnly bool) ([]*WebhookRegistration, error)
	UpdateWebhook(ctx context.Context, webhook *WebhookRegistration) error
	GetDeliveryByID(ctx context.Context, deliveryID, namespace string) (*WebhookDelivery, error)
	GetDeliveriesByWebhookID(ctx context.Context, webhookID, namespace string, limit, offset int) ([]*WebhookDelivery, int, error)
	RegisterEvent(ctx context.Context, event *EventRegistration) error
	GetEventByName(ctx context.Context, eventName string) (*EventRegistration, error)
	ListEvents(ctx context.Context, activeOnly bool) ([]*EventRegistration, error)
	UpdateEvent(ctx context.Context, event *EventRegistration) error
	DeleteEvent(ctx context.Context, eventName string) error
	RecordWebhookHealthEvent(ctx context.Context, webhookID, deliveryID string, success bool, responseTime, responseCode int, errorMessage string) error
	GetWebhookHealthState(ctx context.Context, webhookID string) (*WebhookHealthMetrics, error)
	GetWebhookHealthSummary(ctx context.Context, webhookID string, hours int) (*WebhookHealthSummary, error)
	GetWebhookHealthTimeSeries(ctx context.Context, webhookID string, hours int, bucketSize string) ([]*WebhookHealthEvent, error)
	AggregateHealthSummaries(ctx context.Context) (int, error)
	GetWebhooksByHealth(ctx context.Context, health WebhookHealth) ([]*WebhookRegistration, error)
	GetHealthSummary(ctx context.Context) (map[WebhookHealth]int, error)
	GetRetriableDeliveries(ctx context.Context, webhookID, namespace string, force bool) ([]*WebhookDelivery, error)
	ResetDeliveryForRetry(ctx context.Context, deliveryID string) error
	GetEventByID(ctx context.Context, eventID string) (*EventRecord, error)
	ListEventReports(ctx context.Context, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error)
}

var _ RepositoryInterface = (*Repository)(nil)
