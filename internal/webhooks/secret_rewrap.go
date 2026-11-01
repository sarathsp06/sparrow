package webhooks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/crypto"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

const rewrapPageSize = 500

// RewrapResult counts what RewrapWebhookSecrets did.
type RewrapResult struct {
	Rewrapped int // values encrypted again
	Failed    int // values that did not decrypt; left as they were
}

// RewrapWebhookSecrets encrypts again every stored webhook secret, signing
// key and secret-header set that NeedsRewrap reports: values written before
// they were bound to their webhook, and values under a key that is no
// longer primary. After a full pass, a key that was rotated out can be
// removed from SPARROW_ENCRYPTION_KEYS.
//
// Each value is replaced only if it is unchanged since it was read, so the
// pass is safe to run on several instances at once and alongside writes.
// A value that does not decrypt is logged, counted and left alone.
func RewrapWebhookSecrets(ctx context.Context, repo store.SecretRewrapRepository, c *crypto.Service, logger *slog.Logger) (RewrapResult, error) {
	var res RewrapResult
	if !c.Enabled() {
		return res, nil
	}
	after := uuid.Nil
	for {
		page, err := repo.ListWebhookSecrets(ctx, after, rewrapPageSize)
		if err != nil {
			return res, err
		}
		for _, w := range page {
			for _, col := range []struct {
				name  string
				value []byte
			}{
				{store.ColumnWebhookSecret, w.WebhookSecret},
				{store.ColumnSecretHeaders, w.SecretHeaders},
				{store.ColumnEd25519PrivateKey, w.Ed25519PrivateKey},
			} {
				if !c.NeedsRewrap(col.value) {
					continue
				}
				replaced, err := rewrapOne(ctx, repo, c, w.ID, col.name, col.value)
				switch {
				case errors.Is(err, errUndecryptable):
					res.Failed++
					logger.ErrorContext(ctx, "Stored secret does not decrypt; left as it is", "webhook_id", w.ID, "column", col.name, "error", err)
				case err != nil:
					return res, err
				case replaced:
					res.Rewrapped++
				}
			}
		}
		if len(page) < rewrapPageSize {
			return res, nil
		}
		after = page[len(page)-1].ID
	}
}

var errUndecryptable = errors.New("does not decrypt")

// rewrapOne re-encrypts one value and reports whether it was replaced (it
// is not when another writer changed it first).
func rewrapOne(ctx context.Context, repo store.SecretRewrapRepository, c *crypto.Service, id uuid.UUID, column string, old []byte) (bool, error) {
	binding := store.WebhookSecretBinding(id, column)
	plain, err := c.EnvelopeDecrypt(old, binding)
	if err != nil {
		return false, fmt.Errorf("%w: %w", errUndecryptable, err)
	}
	defer clear(plain)
	replacement, err := c.EnvelopeEncrypt(plain, binding)
	if err != nil {
		return false, fmt.Errorf("re-encrypt %s of webhook %s: %w", column, id, err)
	}
	return repo.ReplaceWebhookSecret(ctx, id, column, old, replacement)
}
