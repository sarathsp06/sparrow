package store

import (
	"context"
	"time"
)

// RepositoryInterface defines the interface for webhook storage operations
//
//go:generate gowrap gen -i RepositoryInterface -t opentelemetry -o RepositoryInterface_otel.go
type RepositoryInterface interface {
	RegisterWebhook(ctx context.Context, registration *WebhookRegistration) error
	UnregisterWebhook(ctx context.Context, webhookID string) error
	ListWebhooks(ctx context.Context, namespace string, activeOnly bool) ([]*WebhookRegistration, error)
	GetWebhookByID(ctx context.Context, webhookID, namespace string) (*WebhookRegistration, error)
	GetWebhooksByNamespace(ctx context.Context, namespace string, activeOnly bool) ([]*WebhookRegistration, error)
	UpdateWebhook(ctx context.Context, webhook *WebhookRegistration) error

	// Subscription Management
	CreateSubscription(ctx context.Context, sub *EventSubscription) error
	GetSubscription(ctx context.Context, id string) (*EventSubscription, error)
	UpdateSubscription(ctx context.Context, sub *EventSubscription) error
	DeleteSubscription(ctx context.Context, id string) error
	ListSubscriptions(ctx context.Context, webhookID string) ([]*EventSubscription, error)
	GetSubscriptionsByEvent(ctx context.Context, namespace, event string) ([]*EventSubscription, error)

	// Event Management
	RegisterEvent(ctx context.Context, event *EventRegistration) error
	GetEventByName(ctx context.Context, eventName string) (*EventRegistration, error)
	ListEvents(ctx context.Context, activeOnly bool) ([]*EventRegistration, error)
	UpdateEvent(ctx context.Context, event *EventRegistration) error
	DeleteEvent(ctx context.Context, eventName string) error

	// Event Record and Delivery Management
	StoreEvent(ctx context.Context, event *EventRecord) error
	CreateDelivery(ctx context.Context, delivery *WebhookDelivery) error
	UpdateDeliveryStatus(ctx context.Context, deliveryID string, status WebhookDeliveryStatus, responseCode int, responseBody, errorMessage string) error
	UpdateDeliveryRequestBody(ctx context.Context, deliveryID string, requestBody string) error
	GetDeliveriesByWebhook(ctx context.Context, webhookID string) ([]*WebhookDelivery, error)
	GetDeliveriesByEvent(ctx context.Context, eventID string) ([]*WebhookDelivery, error)
	GetDeliveryByID(ctx context.Context, deliveryID, namespace string) (*WebhookDelivery, error)
	GetDeliveriesByWebhookID(ctx context.Context, webhookID, namespace string, limit, offset int) ([]*WebhookDelivery, int, error)
	GetRetriableDeliveries(ctx context.Context, webhookID, namespace string, force bool) ([]*WebhookDelivery, error)
	ResetDeliveryForRetry(ctx context.Context, deliveryID string) error
	GetEventByID(ctx context.Context, eventID string) (*EventRecord, error)
	ListEventReports(ctx context.Context, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error)
	ListEventReportsWithStats(ctx context.Context, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error)
	GetEventDeliveryStats(ctx context.Context, eventID string) (int32, int32, int32, int32, error)

	// Webhook Health Management
	UpdateWebhookHealthState(ctx context.Context, webhookID string, success bool, eventTimestamp time.Time) error
	CalculateWebhookHealth(ctx context.Context, webhookID string, lookbackHours int) (string, error)
	RecordWebhookHealthEvent(ctx context.Context, webhookID, deliveryID string, success bool, responseTime, responseCode int, errorMessage string) error
	GetWebhookHealthState(ctx context.Context, webhookID string) (*WebhookHealthMetrics, error)
	GetWebhookHealthSummary(ctx context.Context, webhookID string, hours int) (*WebhookHealthSummary, error)
	GetWebhookHealthTimeSeries(ctx context.Context, webhookID string, hours int, bucketSize string) ([]*WebhookHealthEvent, error)
	AggregateHealthSummaries(ctx context.Context) (int, error)
	GetWebhooksByHealth(ctx context.Context, health WebhookHealth) ([]*WebhookRegistration, error)
	GetHealthSummary(ctx context.Context) (map[WebhookHealth]int, error)
}

var _ RepositoryInterface = (*Repository)(nil)
