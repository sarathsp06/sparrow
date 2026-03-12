package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for webhook storage operations
//
//go:generate gowrap gen -i RepositoryInterface   -t ../../../templates/opentelemetry.tmpl --o RepositoryInterface_otel.go
type RepositoryInterface interface {
	// Webhook Management
	RegisterWebhook(ctx context.Context, tenantID uuid.UUID, registration *WebhookRegistration) error
	UnregisterWebhook(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID) error
	ListWebhooks(ctx context.Context, tenantID uuid.UUID, namespace string, event string, activeOnly bool) ([]*WebhookRegistration, error)
	ListWebhooksPaginated(ctx context.Context, tenantID uuid.UUID, namespace string, event string, activeOnly bool, limit, offset int) ([]*WebhookRegistration, int, error)
	GetWebhookByID(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, namespace string) (*WebhookRegistration, error)
	GetWebhooksByNamespace(ctx context.Context, tenantID uuid.UUID, namespace string, activeOnly bool) ([]*WebhookRegistration, error)
	UpdateWebhook(ctx context.Context, tenantID uuid.UUID, webhook *WebhookRegistration) error

	// Subscription Management
	CreateSubscription(ctx context.Context, tenantID uuid.UUID, sub *EventSubscription) error
	GetSubscription(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*EventSubscription, error)
	UpdateSubscription(ctx context.Context, tenantID uuid.UUID, sub *EventSubscription) error
	DeleteSubscription(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
	ListSubscriptions(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID) ([]*EventSubscription, error)
	ListSubscriptionsByNamespace(ctx context.Context, tenantID uuid.UUID, namespace string, limit, offset int) ([]*EventSubscription, int, error)
	GetSubscriptionsByEvent(ctx context.Context, tenantID uuid.UUID, namespace, event string) ([]*EventSubscription, error)
	GetSubscriptionsWithWebhooksByEvent(ctx context.Context, tenantID uuid.UUID, namespace, event string) ([]*SubscriptionWithWebhook, error)

	// Event Management (event registrations are tenant-scoped)
	RegisterEvent(ctx context.Context, tenantID uuid.UUID, event *EventRegistration) error
	GetEventByName(ctx context.Context, tenantID uuid.UUID, eventName string) (*EventRegistration, error)
	ListEvents(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]*EventRegistration, error)
	ListEventsPaginated(ctx context.Context, tenantID uuid.UUID, activeOnly bool, limit, offset int) ([]*EventRegistration, int, error)
	UpdateEvent(ctx context.Context, tenantID uuid.UUID, event *EventRegistration) error
	DeleteEvent(ctx context.Context, tenantID uuid.UUID, eventName string) error

	// Event Record and Delivery Management
	StoreEvent(ctx context.Context, tenantID uuid.UUID, event *EventRecord) error
	CreateDelivery(ctx context.Context, tenantID uuid.UUID, delivery *WebhookDelivery) error
	UpdateDeliveryStatus(ctx context.Context, deliveryID uuid.UUID, status WebhookDeliveryStatus, responseCode int, responseBody, errorMessage, errorCategory string) error
	UpdateDeliveryRequestBody(ctx context.Context, deliveryID uuid.UUID, requestBody string) error
	GetDeliveriesByWebhook(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID) ([]*WebhookDelivery, error)
	GetDeliveriesByEvent(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) ([]*WebhookDelivery, error)
	GetDeliveryByID(ctx context.Context, tenantID uuid.UUID, deliveryID uuid.UUID, namespace string) (*WebhookDelivery, error)
	GetDeliveriesByWebhookID(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, namespace string, limit, offset int) ([]*WebhookDelivery, int, error)
	GetDeliveriesByEventPaginated(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID, namespace string, limit, offset int) ([]*WebhookDelivery, int, error)
	ListDeliveriesPaginated(ctx context.Context, tenantID uuid.UUID, namespace string, limit, offset int) ([]*WebhookDelivery, int, error)
	GetRetriableDeliveries(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, namespace string, force bool) ([]*WebhookDelivery, error)
	ResetDeliveryForRetry(ctx context.Context, deliveryID uuid.UUID) error
	GetEventByID(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) (*EventRecord, error)
	ListEventReports(ctx context.Context, tenantID uuid.UUID, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error)
	ListEventReportsWithStats(ctx context.Context, tenantID uuid.UUID, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error)
	GetEventDeliveryStats(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) (int32, int32, int32, int32, error)

	// Webhook Health Management
	UpdateWebhookHealthState(ctx context.Context, webhookID uuid.UUID, success bool, eventTimestamp time.Time) error
	CalculateWebhookHealth(ctx context.Context, webhookID uuid.UUID, lookbackHours int) (string, error)
	RecordWebhookHealthEvent(ctx context.Context, webhookID, deliveryID uuid.UUID, success bool, responseTime, responseCode int, errorMessage string, errorCategory string) error
	GetDeliveryAttempts(ctx context.Context, deliveryID uuid.UUID) ([]*WebhookHealthEvent, error)
	GetWebhookHealthState(ctx context.Context, webhookID uuid.UUID) (*WebhookHealthMetrics, error)
	GetWebhookHealthSummary(ctx context.Context, webhookID uuid.UUID, hours int) (*WebhookHealthSummary, error)
	GetWebhookHealthTimeSeries(ctx context.Context, webhookID uuid.UUID, hours int, bucketSize string) ([]*WebhookHealthEvent, error)
	AggregateHealthSummaries(ctx context.Context) (int, error)
	GetWebhooksByHealth(ctx context.Context, tenantID uuid.UUID, health WebhookHealth) ([]*WebhookRegistration, error)
	GetWebhooksByHealthPaginated(ctx context.Context, tenantID uuid.UUID, health WebhookHealth, limit, offset int) ([]*WebhookRegistration, int, error)
	GetHealthSummary(ctx context.Context, tenantID uuid.UUID) (map[WebhookHealth]int, error)
	GetNamespaceStats(ctx context.Context, tenantID uuid.UUID, namespace string) (*NamespaceStats, error)
}

var _ RepositoryInterface = (*Repository)(nil)
