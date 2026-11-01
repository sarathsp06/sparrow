package webhooks

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cryptosvc "github.com/sarathsp06/sparrow/internal/crypto"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// fakeSecretRepo is a SecretRewrapRepository over a map, ordered by id.
type fakeSecretRepo struct {
	rows map[uuid.UUID]*store.WebhookSecrets
}

func (f *fakeSecretRepo) ListWebhookSecrets(_ context.Context, after uuid.UUID, limit int) ([]store.WebhookSecrets, error) {
	var out []store.WebhookSecrets
	for _, r := range f.rows {
		if bytes.Compare(r.ID[:], after[:]) > 0 {
			out = append(out, *r)
		}
	}
	slices.SortFunc(out, func(a, b store.WebhookSecrets) int { return bytes.Compare(a.ID[:], b.ID[:]) })
	return out[:min(limit, len(out))], nil
}

func (f *fakeSecretRepo) ReplaceWebhookSecret(_ context.Context, id uuid.UUID, column string, old, replacement []byte) (bool, error) {
	r := f.rows[id]
	field := map[string]*[]byte{
		store.ColumnWebhookSecret:     &r.WebhookSecret,
		store.ColumnSecretHeaders:     &r.SecretHeaders,
		store.ColumnEd25519PrivateKey: &r.Ed25519PrivateKey,
	}[column]
	if !bytes.Equal(*field, old) {
		return false, nil
	}
	*field = replacement
	return true, nil
}

func keyring(t *testing.T, primary string, ids ...string) *cryptosvc.Service {
	t.Helper()
	material := map[string][]byte{"old": bytes.Repeat([]byte{1}, 32), "new": bytes.Repeat([]byte{2}, 32)}
	var keys []cryptosvc.Key
	for _, id := range ids {
		keys = append(keys, cryptosvc.Key{ID: id, Material: material[id]})
	}
	kr, err := cryptosvc.NewKeyring(keys, primary)
	require.NoError(t, err)
	svc, err := cryptosvc.NewServiceFromKeyring(kr)
	require.NoError(t, err)
	return svc
}

// After a key rotation, the pass moves every value to the new primary key,
// so the old key can then be removed.
func TestRewrapWebhookSecrets_MovesValuesToThePrimaryKey(t *testing.T) {
	old := keyring(t, "old", "old")
	rotated := keyring(t, "new", "old", "new")
	repo := &fakeSecretRepo{rows: map[uuid.UUID]*store.WebhookSecrets{}}
	for i := range rewrapPageSize + 3 { // more than one page
		id := uuid.New()
		secret, err := old.EncryptString("whsec_x", store.WebhookSecretBinding(id, store.ColumnWebhookSecret))
		require.NoError(t, err)
		row := &store.WebhookSecrets{ID: id, WebhookSecret: secret}
		if i == 0 {
			row.SecretHeaders, err = old.EncryptJSON(map[string]string{"A": "b"}, store.WebhookSecretBinding(id, store.ColumnSecretHeaders))
			require.NoError(t, err)
		}
		repo.rows[id] = row
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	res, err := RewrapWebhookSecrets(context.Background(), repo, rotated, logger)
	require.NoError(t, err)
	assert.Equal(t, RewrapResult{Rewrapped: rewrapPageSize + 4}, res)

	newOnly := keyring(t, "new", "new")
	for id, r := range repo.rows {
		got, err := newOnly.DecryptString(r.WebhookSecret, store.WebhookSecretBinding(id, store.ColumnWebhookSecret))
		require.NoError(t, err)
		assert.Equal(t, "whsec_x", got)
	}

	res, err = RewrapWebhookSecrets(context.Background(), repo, rotated, logger)
	require.NoError(t, err)
	assert.Zero(t, res, "a second pass has nothing to do")
}

// A value that does not decrypt is counted and left alone; the pass goes on.
func TestRewrapWebhookSecrets_LeavesUndecryptableValues(t *testing.T) {
	old := keyring(t, "old", "old")
	rotated := keyring(t, "new", "old", "new")
	victim, thief := uuid.New(), uuid.New()
	stolen, err := old.EncryptString("whsec_victim", store.WebhookSecretBinding(victim, store.ColumnWebhookSecret))
	require.NoError(t, err)
	repo := &fakeSecretRepo{rows: map[uuid.UUID]*store.WebhookSecrets{
		thief: {ID: thief, WebhookSecret: stolen}, // copied from another webhook
	}}

	res, err := RewrapWebhookSecrets(context.Background(), repo, rotated, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	assert.Equal(t, RewrapResult{Failed: 1}, res)
	assert.Equal(t, stolen, repo.rows[thief].WebhookSecret)
}
