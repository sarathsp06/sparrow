package store

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface composes all per-domain narrow repository interfaces.
// This is the interface used by services and workers that need access to
// all storage operations. Consumers should prefer the per-domain interfaces
// (WebhookRepository, SubscriptionRepository, etc.) to minimize dependencies.
//
//go:generate gowrap gen -p github.com/sarathsp06/sparrow/internal/webhooks/store -i RepositoryInterface -t ../../../templates/opentelemetry.tmpl -o RepositoryInterface_otel.go -l ""
type RepositoryInterface interface {
	Transactor
	WebhookRepository
	SubscriptionRepository
	EventTypeRepository
	EventRepository
	HealthRepository
	BatchRepository
	RateLimitRepository

	// Composite operations that span multiple domains.
	RegisterWebhookWithSubscriptions(ctx context.Context, tenantID uuid.UUID, registration *WebhookRegistration, subscriptions []*EventSubscription) error
	ReplaceWebhookSubscriptions(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, namespace string, newSubscriptions []*EventSubscription) error
}

// Transactor allows executing operations within a database transaction.
// The fn receives a full RepositoryInterface, enabling cross-domain operations
// within a single transaction.
type Transactor interface {
	RunInTransaction(fn func(RepositoryInterface) error) error
}

var _ RepositoryInterface = (*Repository)(nil)
