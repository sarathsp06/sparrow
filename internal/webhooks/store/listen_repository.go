package store

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// ListenSession is the liveness record of a `sparrow listen` session. The
// session is the webhook WebhookID (URL sparrow-cli://<WebhookID>); this row
// adds when it ends on its own and when the CLI last polled.
type ListenSession struct {
	WebhookID  uuid.UUID `json:"webhook_id" db:"webhook_id"`
	TenantID   uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Consumer   string    `json:"consumer" db:"consumer"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	ExpiresAt  time.Time `json:"expires_at" db:"expires_at"`
	LastSeenAt time.Time `json:"last_seen_at" db:"last_seen_at"`
}

// ListenDelivery is one delivery attempt handed to a listen session: the
// fully prepared (signed) request, and the CLI's response once it reports it.
type ListenDelivery struct {
	DeliveryID      uuid.UUID     `json:"delivery_id" db:"delivery_id"`
	WebhookID       uuid.UUID     `json:"webhook_id" db:"webhook_id"`
	Method          string        `json:"method" db:"method"`
	Headers         JSONStringMap `json:"headers" db:"headers"`
	Body            []byte        `json:"body" db:"body"`
	QueuedAt        time.Time     `json:"queued_at" db:"queued_at"`
	ClaimedAt       *time.Time    `json:"claimed_at,omitempty" db:"claimed_at"`
	ResponseStatus  *int          `json:"response_status,omitempty" db:"response_status"`
	ResponseHeaders JSONStringMap `json:"response_headers,omitempty" db:"response_headers"`
	ResponseBody    []byte        `json:"response_body,omitempty" db:"response_body"`
	RespondedAt     *time.Time    `json:"responded_at,omitempty" db:"responded_at"`
}

// ListenRepository stores listen sessions and the deliveries handed to them.
type ListenRepository interface {
	CreateListenSession(ctx context.Context, session *ListenSession) error
	GetListenSession(ctx context.Context, tenantID, webhookID uuid.UUID) (*ListenSession, error)
	CountListenSessions(ctx context.Context, tenantID uuid.UUID, consumer string) (int, error)
	// TouchListenSession records that the CLI polled now.
	TouchListenSession(ctx context.Context, tenantID, webhookID uuid.UUID) error
	// DeleteExpiredListenSessions deletes the webhooks of sessions past
	// expires_at (the session, its subscriptions and deliveries cascade).
	DeleteExpiredListenSessions(ctx context.Context) (int64, error)

	// QueueListenDelivery stores a prepared attempt for the CLI to claim,
	// replacing an earlier attempt of the same delivery.
	QueueListenDelivery(ctx context.Context, d *ListenDelivery) error
	// ClaimListenDeliveries marks up to limit unclaimed attempts of a session
	// as claimed and returns them, oldest first.
	ClaimListenDeliveries(ctx context.Context, webhookID uuid.UUID, limit int) ([]*ListenDelivery, error)
	// RespondListenDelivery records the CLI's response to a claimed attempt.
	// It returns ErrNotFound when there is no such claimed, unanswered attempt
	// (it timed out and was removed, or was already answered).
	RespondListenDelivery(ctx context.Context, webhookID, deliveryID uuid.UUID, status int, headers map[string]string, body []byte) error
	GetListenDelivery(ctx context.Context, deliveryID uuid.UUID) (*ListenDelivery, error)
	DeleteListenDelivery(ctx context.Context, deliveryID uuid.UUID) error
}

// CreateListenSession inserts the session row for an existing webhook.
func (r *Repository) CreateListenSession(ctx context.Context, s *ListenSession) error {
	now := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.LastSeenAt.IsZero() {
		s.LastSeenAt = now
	}
	_, err := r.conn.ExecContext(ctx, `
		INSERT INTO listen_sessions (webhook_id, tenant_id, consumer, created_at, expires_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, s.WebhookID, s.TenantID, s.Consumer, s.CreatedAt, s.ExpiresAt, s.LastSeenAt)
	return storage.Error(err)
}

// GetListenSession returns a session, or ErrNotFound.
func (r *Repository) GetListenSession(ctx context.Context, tenantID, webhookID uuid.UUID) (*ListenSession, error) {
	var s ListenSession
	err := r.conn.GetContext(ctx, &s, `
		SELECT webhook_id, tenant_id, consumer, created_at, expires_at, last_seen_at
		FROM listen_sessions
		WHERE tenant_id = $1 AND webhook_id = $2
	`, tenantID, webhookID)
	if err != nil {
		return nil, storage.Error(err)
	}
	return &s, nil
}

// CountListenSessions counts a consumer's unexpired sessions.
func (r *Repository) CountListenSessions(ctx context.Context, tenantID uuid.UUID, consumer string) (int, error) {
	var n int
	err := r.conn.GetContext(ctx, &n, `
		SELECT COUNT(*) FROM listen_sessions
		WHERE tenant_id = $1 AND consumer = $2 AND expires_at > NOW()
	`, tenantID, consumer)
	return n, storage.Error(err)
}

// TouchListenSession sets last_seen_at to now.
func (r *Repository) TouchListenSession(ctx context.Context, tenantID, webhookID uuid.UUID) error {
	res, err := r.conn.ExecContext(ctx, `
		UPDATE listen_sessions SET last_seen_at = NOW()
		WHERE tenant_id = $1 AND webhook_id = $2
	`, tenantID, webhookID)
	if err != nil {
		return storage.Error(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// DeleteExpiredListenSessions deletes the webhooks behind expired sessions.
func (r *Repository) DeleteExpiredListenSessions(ctx context.Context) (int64, error) {
	res, err := r.conn.ExecContext(ctx, `
		DELETE FROM webhook_registrations
		WHERE id IN (SELECT webhook_id FROM listen_sessions WHERE expires_at < NOW())
	`)
	if err != nil {
		return 0, storage.Error(err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// QueueListenDelivery upserts an attempt, clearing any earlier claim/response.
func (r *Repository) QueueListenDelivery(ctx context.Context, d *ListenDelivery) error {
	if d.Headers == nil {
		d.Headers = JSONStringMap{}
	}
	_, err := r.conn.ExecContext(ctx, `
		INSERT INTO listen_deliveries (delivery_id, webhook_id, method, headers, body, queued_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (delivery_id) DO UPDATE SET
			method = EXCLUDED.method,
			headers = EXCLUDED.headers,
			body = EXCLUDED.body,
			queued_at = NOW(),
			claimed_at = NULL,
			response_status = NULL,
			response_headers = NULL,
			response_body = NULL,
			responded_at = NULL
	`, d.DeliveryID, d.WebhookID, d.Method, d.Headers, d.Body)
	return storage.Error(err)
}

// ClaimListenDeliveries claims the oldest unclaimed attempts of a session.
// SKIP LOCKED lets two pollers of one session (two terminals) share the
// work instead of both receiving every attempt.
func (r *Repository) ClaimListenDeliveries(ctx context.Context, webhookID uuid.UUID, limit int) ([]*ListenDelivery, error) {
	var out []*ListenDelivery
	err := r.conn.SelectContext(ctx, &out, `
		UPDATE listen_deliveries SET claimed_at = NOW()
		WHERE delivery_id IN (
			SELECT delivery_id FROM listen_deliveries
			WHERE webhook_id = $1 AND claimed_at IS NULL
			ORDER BY queued_at
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		RETURNING delivery_id, webhook_id, method, headers, body, queued_at, claimed_at
	`, webhookID, limit)
	if err != nil {
		return nil, storage.Error(err)
	}
	return out, nil
}

// RespondListenDelivery records a response to a claimed, unanswered attempt.
func (r *Repository) RespondListenDelivery(ctx context.Context, webhookID, deliveryID uuid.UUID, status int, headers map[string]string, body []byte) error {
	if headers == nil {
		headers = map[string]string{}
	}
	if body == nil {
		body = []byte{}
	}
	res, err := r.conn.ExecContext(ctx, `
		UPDATE listen_deliveries
		SET response_status = $3, response_headers = $4, response_body = $5, responded_at = NOW()
		WHERE webhook_id = $1 AND delivery_id = $2 AND claimed_at IS NOT NULL AND responded_at IS NULL
	`, webhookID, deliveryID, status, JSONStringMap(headers), body)
	if err != nil {
		return storage.Error(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// GetListenDelivery returns an attempt, or ErrNotFound.
func (r *Repository) GetListenDelivery(ctx context.Context, deliveryID uuid.UUID) (*ListenDelivery, error) {
	var d ListenDelivery
	err := r.conn.GetContext(ctx, &d, `
		SELECT delivery_id, webhook_id, method, headers, body, queued_at, claimed_at,
		       response_status, response_headers, response_body, responded_at
		FROM listen_deliveries
		WHERE delivery_id = $1
	`, deliveryID)
	if err != nil {
		return nil, storage.Error(err)
	}
	return &d, nil
}

// DeleteListenDelivery removes an attempt once the worker is done with it.
func (r *Repository) DeleteListenDelivery(ctx context.Context, deliveryID uuid.UUID) error {
	_, err := r.conn.ExecContext(ctx, `DELETE FROM listen_deliveries WHERE delivery_id = $1`, deliveryID)
	return storage.Error(err)
}
