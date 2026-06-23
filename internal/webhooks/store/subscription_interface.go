package store

import (
	"context"

	"github.com/google/uuid"
)

// SubscriptionRepository defines operations for event subscriptions.
type SubscriptionRepository interface {
	CreateSubscription(ctx context.Context, tenantID uuid.UUID, sub *EventSubscription) error
	GetSubscription(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*EventSubscription, error)
	UpdateSubscription(ctx context.Context, tenantID uuid.UUID, sub *EventSubscription) error
	DeleteSubscription(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
	ListSubscriptions(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID) ([]*EventSubscription, error)
	ListSubscriptionsByNamespace(ctx context.Context, tenantID uuid.UUID, namespace string, limit, offset int) ([]*EventSubscription, int, error)
	GetSubscriptionsByEvent(ctx context.Context, tenantID uuid.UUID, namespace, event string, labels map[string]string) ([]*EventSubscription, error)
	GetSubscriptionsWithWebhooksByEvent(ctx context.Context, tenantID uuid.UUID, namespace, event string, labels map[string]string) ([]*SubscriptionWithWebhook, error)
	ListSubscriptionsByWebhookIDs(ctx context.Context, tenantID uuid.UUID, webhookIDs []uuid.UUID) ([]*EventSubscription, error)
}
