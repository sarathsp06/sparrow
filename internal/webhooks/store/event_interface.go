package store

import (
	"context"

	"github.com/google/uuid"
)

// EventRepository defines operations for event records, deliveries, and filtered list operations.
type EventRepository interface {
	StoreEvent(ctx context.Context, tenantID uuid.UUID, event *EventRecord) error
	GetEventByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, namespace, idempotencyKey string) (*EventRecord, error)
	GetEventByID(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) (*EventRecord, error)
	GetEventDeliveryStats(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) (int32, int32, int32, int32, error)
	DeleteEventByID(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) error

	CreateDelivery(ctx context.Context, tenantID uuid.UUID, delivery *WebhookDelivery) error
	BatchCreateDeliveries(ctx context.Context, tenantID uuid.UUID, deliveries []*WebhookDelivery) error
	UpdateDeliveryStatus(ctx context.Context, deliveryID uuid.UUID, status WebhookDeliveryStatus, responseCode int, responseBody, errorMessage, errorCategory string) error
	UpdateDeliveryRequestBody(ctx context.Context, deliveryID uuid.UUID, requestBody string) error
	GetDeliveryByID(ctx context.Context, tenantID uuid.UUID, deliveryID uuid.UUID, namespace string) (*WebhookDelivery, error)
	GetDeliveriesByWebhookID(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, namespace string, limit, offset int) ([]*WebhookDelivery, int, error)
	GetDeliveriesByEventPaginated(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID, namespace string, limit, offset int) ([]*WebhookDelivery, int, error)
	ListDeliveriesPaginated(ctx context.Context, tenantID uuid.UUID, namespace string, limit, offset int) ([]*WebhookDelivery, int, error)
	GetRetriableDeliveries(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, namespace string, force bool) ([]*WebhookDelivery, error)
	ResetDeliveryForRetry(ctx context.Context, deliveryID uuid.UUID) error
	DeleteDeliveryByID(ctx context.Context, deliveryID uuid.UUID) error

	ListEventReports(ctx context.Context, tenantID uuid.UUID, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error)
	ListEventReportsWithStats(ctx context.Context, tenantID uuid.UUID, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error)
	ListEventReportsFiltered(ctx context.Context, tenantID uuid.UUID, filter EventReportFilter) ([]*EventReportWithStats, int, error)
	ListDeliveriesFiltered(ctx context.Context, tenantID uuid.UUID, filter DeliveryFilter) ([]*WebhookDelivery, int, error)
}

// EventReportFilter is defined in event_repository.go
// DeliveryFilter is defined in delivery_repository.go
