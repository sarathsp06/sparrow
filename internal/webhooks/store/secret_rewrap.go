package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// WebhookSecrets are the encrypted columns of one webhook, as stored.
type WebhookSecrets struct {
	ID                uuid.UUID `db:"id"`
	WebhookSecret     []byte    `db:"webhook_secret"`
	SecretHeaders     []byte    `db:"secret_headers"`
	Ed25519PrivateKey []byte    `db:"ed25519_private_key"`
}

// SecretRewrapRepository reads and replaces encrypted webhook columns so
// they can be encrypted again (a new envelope version or primary key).
type SecretRewrapRepository interface {
	//sparrow:no-tenant re-encrypts stored secrets of every tenant; values are bound to their webhook id
	ListWebhookSecrets(ctx context.Context, after uuid.UUID, limit int) ([]WebhookSecrets, error)
	//sparrow:no-tenant replaces one value by webhook id, only if it is still the one that was read
	ReplaceWebhookSecret(ctx context.Context, webhookID uuid.UUID, column string, old, replacement []byte) (bool, error)
}

// ListWebhookSecrets returns webhooks with at least one encrypted column,
// ordered by id after the given one. Deleted webhooks are included: their
// values are still encrypted under a key that may be retired.
func (r *Repository) ListWebhookSecrets(ctx context.Context, after uuid.UUID, limit int) ([]WebhookSecrets, error) {
	var out []WebhookSecrets
	err := r.conn.SelectContext(ctx, &out, `
		SELECT id, webhook_secret, secret_headers, ed25519_private_key
		FROM webhook_registrations
		WHERE id > $1
		  AND (webhook_secret IS NOT NULL OR secret_headers IS NOT NULL OR ed25519_private_key IS NOT NULL)
		ORDER BY id
		LIMIT $2`, after, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhook secrets: %w", err)
	}
	return out, nil
}

// replaceSecretQueries holds one statement per encrypted column, so the
// column name never comes from a caller.
var replaceSecretQueries = map[string]string{
	ColumnWebhookSecret:     `UPDATE webhook_registrations SET webhook_secret = $3 WHERE id = $1 AND webhook_secret = $2`,
	ColumnSecretHeaders:     `UPDATE webhook_registrations SET secret_headers = $3 WHERE id = $1 AND secret_headers = $2`,
	ColumnEd25519PrivateKey: `UPDATE webhook_registrations SET ed25519_private_key = $3 WHERE id = $1 AND ed25519_private_key = $2`,
}

// ReplaceWebhookSecret sets column to replacement if it still holds old,
// and reports whether it did. A value changed in between (a rotated secret,
// say) is left alone: it was written by current code.
func (r *Repository) ReplaceWebhookSecret(ctx context.Context, webhookID uuid.UUID, column string, old, replacement []byte) (bool, error) {
	query, ok := replaceSecretQueries[column]
	if !ok {
		return false, fmt.Errorf("unknown encrypted column %q", column)
	}
	res, err := r.conn.ExecContext(ctx, query, webhookID, old, replacement)
	if err != nil {
		return false, fmt.Errorf("failed to replace %s: %w", column, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}
