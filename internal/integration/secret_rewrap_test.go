//go:build integration

package integration

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/crypto"
	storePg "github.com/sarathsp06/sparrow/internal/storage/postgres"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// After a key rotation, the startup pass re-encrypts stored webhook secrets
// under the new primary key in Postgres, so the old key can be removed.
func TestRewrapWebhookSecrets_Postgres(t *testing.T) {
	ctx := context.Background()
	databaseURL, _ := setupTestDB(t, ctx)
	runMigrations(t, ctx, databaseURL)
	db, err := storePg.Open(databaseURL, 3)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, tenant.Bootstrap(ctx, db))
	repo := store.NewRepository(db)

	ring := func(primary string, ids ...string) *crypto.Service {
		material := map[string][]byte{"old": bytes.Repeat([]byte{1}, 32), "new": bytes.Repeat([]byte{2}, 32)}
		var keys []crypto.Key
		for _, id := range ids {
			keys = append(keys, crypto.Key{ID: id, Material: material[id]})
		}
		kr, err := crypto.NewKeyring(keys, primary)
		require.NoError(t, err)
		svc, err := crypto.NewServiceFromKeyring(kr)
		require.NoError(t, err)
		return svc
	}
	old := ring("old", "old")

	id := uuid.New()
	secret, err := old.EncryptString("whsec_x", store.WebhookSecretBinding(id, store.ColumnWebhookSecret))
	require.NoError(t, err)
	headers, err := old.EncryptJSON(map[string]string{"X-Api-Key": "k"}, store.WebhookSecretBinding(id, store.ColumnSecretHeaders))
	require.NoError(t, err)
	require.NoError(t, repo.RegisterWebhook(ctx, tenant.DefaultTenantID, &store.WebhookRegistration{
		ID: id, Consumer: "default", URL: "https://example.com/hook", Active: true,
		MaxRetries: 3, RetryBackoffSeconds: 60, RequestTimeoutSeconds: 30, Timeout: 30, ExpectedStatusCodes: pq.Int64Array{200},
		ContentType: "application/json", SignatureType: store.SignatureTypeHMAC,
		WebhookSecret: secret, SecretHeaders: headers,
	}))

	res, err := webhooks.RewrapWebhookSecrets(ctx, repo, ring("new", "old", "new"), slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	assert.Equal(t, webhooks.RewrapResult{Rewrapped: 2}, res)

	got, err := repo.GetWebhookByID(ctx, tenant.DefaultTenantID, id, "default")
	require.NoError(t, err)
	newOnly := ring("new", "new")
	plain, err := newOnly.DecryptString(got.WebhookSecret, store.WebhookSecretBinding(id, store.ColumnWebhookSecret))
	require.NoError(t, err)
	assert.Equal(t, "whsec_x", plain)
	var h map[string]string
	require.NoError(t, newOnly.DecryptJSON(got.SecretHeaders, &h, store.WebhookSecretBinding(id, store.ColumnSecretHeaders)))
	assert.Equal(t, map[string]string{"X-Api-Key": "k"}, h)

	// A value changed since it was read is left alone.
	replaced, err := repo.ReplaceWebhookSecret(ctx, id, store.ColumnWebhookSecret, secret, []byte("stale"))
	require.NoError(t, err)
	assert.False(t, replaced)
}
