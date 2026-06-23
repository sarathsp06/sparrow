package store

import (
	"context"

	"github.com/google/uuid"
)

// EventTypeRepository defines operations for event type registrations and schemas.
type EventTypeRepository interface {
	RegisterEvent(ctx context.Context, tenantID uuid.UUID, event *EventRegistration) error
	GetEventByName(ctx context.Context, tenantID uuid.UUID, eventName string) (*EventRegistration, error)
	ListEvents(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]*EventRegistration, error)
	ListEventsPaginated(ctx context.Context, tenantID uuid.UUID, activeOnly bool, limit, offset int) ([]*EventRegistration, int, error)
	UpdateEvent(ctx context.Context, tenantID uuid.UUID, event *EventRegistration) error
	DeleteEvent(ctx context.Context, tenantID uuid.UUID, eventName string) error
}
