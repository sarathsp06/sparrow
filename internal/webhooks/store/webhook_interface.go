package store

import (
	"context"

	"github.com/google/uuid"
)

// WebhookRepository defines operations for webhook_registrations.
type WebhookRepository interface {
	RegisterWebhook(ctx context.Context, tenantID uuid.UUID, registration *WebhookRegistration) error
	UnregisterWebhook(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID) error
	ListWebhooks(ctx context.Context, tenantID uuid.UUID, namespace string, event string, activeOnly bool) ([]*WebhookRegistration, error)
	ListWebhooksPaginated(ctx context.Context, tenantID uuid.UUID, namespace string, event string, activeOnly bool, limit, offset int) ([]*WebhookRegistration, int, error)
	GetWebhookByID(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, namespace string) (*WebhookRegistration, error)
	UpdateWebhook(ctx context.Context, tenantID uuid.UUID, webhook *WebhookRegistration) error
}
