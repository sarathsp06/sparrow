package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// AlertConfig is an opt-in email recipient for Sparrow-generated
// webhook.health_changed / webhook.delivery_failed system events.
// A nil WebhookID means "every webhook in this consumer".
type AlertConfig struct {
	ID         uuid.UUID      `json:"id" db:"id"`
	TenantID   uuid.UUID      `json:"tenant_id" db:"tenant_id"`
	Consumer   string         `json:"consumer" db:"consumer"`
	WebhookID  *uuid.UUID     `json:"webhook_id,omitempty" db:"webhook_id"`
	Email      string         `json:"email" db:"email"`
	EventTypes pq.StringArray `json:"event_types" db:"event_types"`
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
}

// AlertConfigRepository defines operations for webhook_alert_configs.
type AlertConfigRepository interface {
	CreateAlertConfig(ctx context.Context, cfg *AlertConfig) error
	ListAlertConfigs(ctx context.Context, tenantID uuid.UUID, consumer string, webhookID *uuid.UUID) ([]*AlertConfig, error)
	GetAlertConfig(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*AlertConfig, error)
	DeleteAlertConfig(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
	// ResolveAlertRecipients returns the distinct emails subscribed to eventType
	// for webhookID, matching both webhook-scoped and consumer-wide configs.
	ResolveAlertRecipients(ctx context.Context, tenantID, webhookID uuid.UUID, consumer, eventType string) ([]string, error)
}

// CreateAlertConfig inserts a new alert recipient config.
func (r *Repository) CreateAlertConfig(ctx context.Context, cfg *AlertConfig) error {
	if cfg.ID == uuid.Nil {
		cfg.ID = uuid.New()
	}
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = time.Now()
	}
	_, err := r.conn.ExecContext(ctx, `
		INSERT INTO webhook_alert_configs (id, tenant_id, consumer, webhook_id, email, event_types, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, cfg.ID, cfg.TenantID, cfg.Consumer, cfg.WebhookID, cfg.Email, cfg.EventTypes, cfg.CreatedAt)
	return storage.Error(err)
}

// ListAlertConfigs lists alert configs in a consumer, optionally filtered to
// one webhook's own configs (webhookID != nil) or every config in the
// consumer, including consumer-wide ones (webhookID == nil).
func (r *Repository) ListAlertConfigs(ctx context.Context, tenantID uuid.UUID, consumer string, webhookID *uuid.UUID) ([]*AlertConfig, error) {
	var configs []*AlertConfig
	var err error
	if webhookID != nil {
		err = r.conn.SelectContext(ctx, &configs, `
			SELECT id, tenant_id, consumer, webhook_id, email, event_types, created_at
			FROM webhook_alert_configs
			WHERE tenant_id = $1 AND consumer = $2 AND webhook_id = $3
			ORDER BY created_at
		`, tenantID, consumer, *webhookID)
	} else {
		err = r.conn.SelectContext(ctx, &configs, `
			SELECT id, tenant_id, consumer, webhook_id, email, event_types, created_at
			FROM webhook_alert_configs
			WHERE tenant_id = $1 AND consumer = $2
			ORDER BY created_at
		`, tenantID, consumer)
	}
	if err != nil {
		return nil, storage.Error(err)
	}
	return configs, nil
}

// GetAlertConfig gets an alert config by id within a tenant.
func (r *Repository) GetAlertConfig(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*AlertConfig, error) {
	var cfg AlertConfig
	err := r.conn.GetContext(ctx, &cfg, `
		SELECT id, tenant_id, consumer, webhook_id, email, event_types, created_at
		FROM webhook_alert_configs
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id)
	if err != nil {
		return nil, storage.Error(err)
	}
	return &cfg, nil
}

// DeleteAlertConfig deletes an alert config by id within a tenant.
func (r *Repository) DeleteAlertConfig(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	result, err := r.conn.ExecContext(ctx, `DELETE FROM webhook_alert_configs WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return storage.Error(err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return storage.Error(err)
	}
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// ResolveAlertRecipients returns the distinct emails opted into eventType for
// webhookID: configs scoped to that webhook plus consumer-wide configs
// (webhook_id IS NULL) in the same consumer.
func (r *Repository) ResolveAlertRecipients(ctx context.Context, tenantID, webhookID uuid.UUID, consumer, eventType string) ([]string, error) {
	var emails []string
	err := r.conn.SelectContext(ctx, &emails, `
		SELECT DISTINCT email
		FROM webhook_alert_configs
		WHERE tenant_id = $1 AND consumer = $2
		  AND (webhook_id = $3 OR webhook_id IS NULL)
		  AND $4 = ANY(event_types)
	`, tenantID, consumer, webhookID, eventType)
	if err != nil {
		return nil, storage.Error(err)
	}
	return emails, nil
}
