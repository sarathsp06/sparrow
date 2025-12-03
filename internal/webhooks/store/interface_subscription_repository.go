package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// CreateSubscription creates a new event subscription
func (r *Repository) CreateSubscription(ctx context.Context, sub *EventSubscription) error {
	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	now := time.Now()
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	sub.UpdatedAt = now

	query := `
		INSERT INTO event_subscriptions (
			id, webhook_id, event_name, namespace, headers, method,
			transform_enabled, transform_template, timeout, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	headersJSON, err := json.Marshal(sub.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		sub.ID,
		sub.WebhookID,
		sub.EventName,
		sub.Namespace,
		headersJSON,
		sub.Method,
		sub.TransformEnabled,
		sub.TransformTemplate,
		sub.Timeout,
		sub.CreatedAt,
		sub.UpdatedAt,
	)
	return storage.Error(err)
}

// GetSubscription gets a subscription by ID
func (r *Repository) GetSubscription(ctx context.Context, id uuid.UUID) (*EventSubscription, error) {
	query := `
		SELECT id, webhook_id, event_name, namespace, headers, method,
		       transform_enabled, transform_template, timeout, created_at, updated_at
		FROM event_subscriptions
		WHERE id = $1
	`
	var sub EventSubscription
	err := r.db.GetContext(ctx, &sub, query, id)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}
	return &sub, nil
}

// UpdateSubscription updates a subscription
func (r *Repository) UpdateSubscription(ctx context.Context, sub *EventSubscription) error {
	sub.UpdatedAt = time.Now()

	headersJSON, err := json.Marshal(sub.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	query := `
		UPDATE event_subscriptions
		SET headers = $2, method = $3, transform_enabled = $4,
		    transform_template = $5, timeout = $6, updated_at = $7
		WHERE id = $1
	`

	_, err = r.db.ExecContext(ctx, query,
		sub.ID,
		headersJSON,
		sub.Method,
		sub.TransformEnabled,
		sub.TransformTemplate,
		sub.Timeout,
		sub.UpdatedAt,
	)
	return storage.Error(err)
}

// DeleteSubscription deletes a subscription
func (r *Repository) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM event_subscriptions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return storage.Error(err)
}

// ListSubscriptions lists subscriptions for a webhook
func (r *Repository) ListSubscriptions(ctx context.Context, webhookID uuid.UUID) ([]*EventSubscription, error) {
	query := `
		SELECT id, webhook_id, event_name, namespace, headers, method,
		       transform_enabled, transform_template, timeout, created_at, updated_at
		FROM event_subscriptions
		WHERE webhook_id = $1
		ORDER BY created_at DESC
	`
	var subs []*EventSubscription
	err := r.db.SelectContext(ctx, &subs, query, webhookID)
	if err != nil {
		return nil, storage.Error(err)
	}
	return subs, nil
}

// GetSubscriptionsByEvent finds all active subscriptions for a specific event in a namespace.
func (r *Repository) GetSubscriptionsByEvent(ctx context.Context, namespace, event string) ([]*EventSubscription, error) {
	query := `
		SELECT es.id, es.webhook_id, es.event_name, es.namespace, es.headers, es.method, 
		       es.transform_enabled, es.transform_template, es.timeout, es.created_at, es.updated_at
		FROM event_subscriptions es
		JOIN webhook_registrations wr ON es.webhook_id = wr.id
		WHERE es.namespace = $1 AND es.event_name = $2 AND wr.active = true
	`
	var subscriptions []*EventSubscription

	err := r.db.SelectContext(ctx, &subscriptions, query, namespace, event)
	if err != nil {
		return nil, storage.Error(err)
	}
	return subscriptions, nil
}
