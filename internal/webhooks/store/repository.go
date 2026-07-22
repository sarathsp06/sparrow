package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// Repository provides data access layer for webhook operations.
// It handles CRUD operations for webhooks, events, deliveries, and health tracking.
// All query/exec operations use the conn field (storage.DBTX) so that the
// repository works transparently against either a plain connection or a transaction.
//
// Use WithConn to obtain a repository clone that runs against a specific
// connection (e.g. a transaction obtained from storage.WithTransaction):
//
//	storage.WithTransaction(db, func(tx storage.DBTX) error {
//	    return repo.WithConn(tx).RegisterWebhook(ctx, ...)
//	})
//
// Methods are distributed across separate files based on their primary table:
// - webhook_repository.go: webhook_registrations table operations
// - event_repository.go: event_records table operations
// - delivery_repository.go: webhook_deliveries table operations
// - health_repository.go: webhook_health_* table operations
// - subscription_repository.go: event_subscriptions table operations
// - event_registration_repository.go: event_registrations table operations
type Repository struct {
	db   storage.DB   // full connection — used for Beginx/Ping/Close
	conn storage.DBTX // query/exec target — either db or a transaction
}

// NewRepository creates a new Repository instance with the provided database connection.
// The storage.DB interface allows for dependency injection and easier testing with mock implementations.
func NewRepository(db storage.DB) *Repository {
	return &Repository{
		db:   db,
		conn: db, // default: queries go directly to the pool
	}
}

// WithConn returns a shallow copy of the repository that executes all
// queries against conn instead of the original database pool. This is
// the primary mechanism for enlisting a repository in an external
// transaction started via storage.WithTransaction.
func (r *Repository) WithConn(conn storage.DBTX) *Repository {
	return &Repository{
		db:   r.db,
		conn: conn,
	}
}

// RunInTransaction executes fn within a database transaction. The fn
// receives a transactional RepositoryInterface backed by the same tx.
func (r *Repository) RunInTransaction(fn func(RepositoryInterface) error) error {
	return storage.WithTransaction(r.db, func(tx storage.DBTX) error {
		return fn(r.WithConn(tx))
	})
}

// StoreEventTx persists an event record within an existing database transaction.
// Automatically generates UUID if event.ID is empty and sets created_at/expires_at timestamps.
// The expires_at is calculated from TTL (time-to-live) in seconds from creation time.
// This transactional version ensures atomic operations when creating events with related deliveries.
func (r *Repository) StoreEventTx(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, event *EventRecord) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	event.TenantID = tenantID
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	if event.ExpiresAt.IsZero() {
		if event.TTL <= 0 {
			event.ExpiresAt = NoExpiryTime
		} else {
			event.ExpiresAt = time.Now().Add(time.Duration(event.TTL) * time.Second)
		}
	}

	query := `
		INSERT INTO event_records (
			id, tenant_id, namespace, event, payload, ttl, metadata, labels, schema_valid, idempotency_key, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	labelsJSON, err := json.Marshal(event.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = tx.Exec(ctx, query,
		event.ID,
		event.TenantID,
		event.Namespace,
		event.Event,
		string(payloadJSON),
		event.TTL,
		metadataJSON,
		labelsJSON,
		event.SchemaValid,
		event.IdempotencyKey,
		event.CreatedAt,
		event.ExpiresAt,
	)
	return storage.Error(err)
}

// GetWebhooksByEventTx retrieves all active webhooks subscribed to a specific event within a transaction.
// Only returns webhooks that are active=true and match the tenant and namespace for tenant isolation.
// Includes complete webhook configuration including HTTP settings for delivery customization.
func (r *Repository) GetWebhooksByEventTx(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, namespace, event string) ([]*WebhookRegistration, error) {
	query := `
		SELECT id, tenant_id, namespace, url, headers, timeout, active, description, health,
		       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
		       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
		       user_agent, content_type, secret_headers, created_at, updated_at
		FROM webhook_registrations 
		WHERE tenant_id = $1 AND namespace = $2 AND active = true
	`

	rows, err := tx.Query(ctx, query, tenantID, namespace)
	if err != nil {
		return nil, storage.Error(err)
	}
	defer rows.Close()

	var webhooks []*WebhookRegistration
	for rows.Next() {
		var wh WebhookRegistration
		var headersJSON []byte

		err := rows.Scan(
			&wh.ID,
			&wh.TenantID,
			&wh.Namespace,
			&wh.URL,
			&headersJSON,
			&wh.Timeout,
			&wh.Active,
			&wh.Description,
			&wh.Health,
			&wh.MaxRetries,
			&wh.RetryBackoffSeconds,
			&wh.CaptureResponseBody,
			&wh.FollowRedirects,
			&wh.VerifySSL,
			&wh.RequestTimeoutSeconds,
			&wh.ExpectedStatusCodes,
			&wh.WebhookSecret,
			&wh.UserAgent,
			&wh.ContentType,
			&wh.SecretHeaders,
			&wh.CreatedAt,
			&wh.UpdatedAt,
		)
		if err != nil {
			return nil, storage.Error(err)
		}

		if err := json.Unmarshal(headersJSON, &wh.Headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}

		webhooks = append(webhooks, &wh)
	}

	return webhooks, nil
}
