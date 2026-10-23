//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// TestTenantCreateAndPurge fills three tenants (the default one, a control
// tenant and the one to erase) with the same kinds of data, purges one, and
// checks that every tenant-owned row of that tenant is gone while the other
// two tenants are byte-for-byte untouched. Tables are found from the schema,
// so a new tenant-owned table is covered without editing this test.
func TestTenantCreateAndPurge(t *testing.T) {
	ctx := context.Background()
	sp := startEmbeddedSparrow(t)

	victim := sp.addTenant(t, "victim")
	control := sp.addTenant(t, "control")

	// CreateTenant is idempotent and seeds Sparrow's own event types.
	require.NoError(t, sp.srv.CreateTenant(ctx, victim, "victim"))
	var sysTypes int
	require.NoError(t, sp.db.Get(&sysTypes, `SELECT count(*) FROM event_registrations WHERE tenant_id = $1 AND name LIKE 'sparrow.%'`, victim))
	require.Positive(t, sysTypes, "a new tenant gets the sparrow.* system event types")

	// A receiver that always fails keeps a delivery job retrying, so the
	// purge has a live job to cancel.
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failing.Close)

	populate := func(tenantID uuid.UUID) {
		c := sp.client(t, tenantID)
		const consumer = "acme"
		resp, err := c.post(ctx, "/v1/event-types", map[string]any{"name": "order.created", "active": true}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		ok := newCountingReceiver(t)
		for _, url := range []string{ok.srv.URL, failing.URL} {
			resp, err = c.post(ctx, "/v1/consumers/"+consumer+"/webhooks", map[string]any{
				"events": []string{"order.created"}, "url": url + "/hook", "active": true,
				"rate_limit_rps":     50,
				"transform_template": `{"transformed":true}`,
				"http_config":        map[string]any{"max_retries": 10, "retry_backoff_seconds": 1},
			}, nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode)
		}
		resp, err = c.post(ctx, "/v1/consumers/"+consumer+"/alert-configs", map[string]any{
			"email": "ops@example.com", "event_types": []string{"sparrow.webhook.disabled"},
		}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		resp, err = c.post(ctx, "/v1/consumers/"+consumer+"/events?event=order.created",
			map[string]any{"payload": map[string]any{"order_id": "1"}}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		ok.waitFor(t, 1)

		// A prepared retry leaves a pending batch job.
		var list struct {
			RetryID string `json:"retry_id"`
		}
		resp, err = c.get(ctx, "/v1/consumers/"+consumer+"/deliveries?prepare_retry=true", &list)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.NotEmpty(t, list.RetryID)

		// Sparrow's own tokens always live in the default realm; a tenant's
		// realm is its id, so write one token and one invite there directly.
		realm := tenantID.String()
		if tenantID == uuid.Nil {
			realm = "00000000-0000-0000-0000-000000000001"
		}
		_, err = sp.db.Exec(`INSERT INTO access_tokens (id, realm, name, secret_hash, created_by, created_at) VALUES ($1, $2, 'n', $3, 'test', now())`,
			"tok_"+uuid.NewString(), realm, []byte(uuid.NewString()))
		require.NoError(t, err)
		_, err = sp.db.Exec(`INSERT INTO access_invites (id, realm, name, secret_hash, created_by, created_at, expires_at) VALUES ($1, $2, 'n', $3, 'test', now(), now() + interval '1 day')`,
			"inv_"+uuid.NewString(), realm, []byte(uuid.NewString()))
		require.NoError(t, err)
	}
	populate(uuid.Nil) // the default tenant
	populate(control)
	populate(victim)

	// Wait until the health evaluator has folded attempts into health state,
	// buckets and metrics for every tenant, so those tables hold rows too.
	require.Eventually(t, func() bool {
		var n int
		require.NoError(t, sp.db.Get(&n, `
			SELECT count(DISTINCT wr.tenant_id) FROM webhook_health_state hs
			JOIN webhook_registrations wr ON wr.id = hs.webhook_id
			WHERE hs.consecutive_failures > 0`))
		return n == 3
	}, 30*time.Second, 200*time.Millisecond)

	const defaultTenant = "00000000-0000-0000-0000-000000000001"
	victimWebhooks := webhookIDs(t, sp.db, victim.String())
	require.Len(t, victimWebhooks, 2)
	require.Positive(t, liveJobs(t, sp.db, victim.String()), "the victim should have a retrying job to cancel")

	before := map[string]map[string]int{
		defaultTenant:    tenantRowCounts(t, sp.db, defaultTenant, webhookIDs(t, sp.db, defaultTenant)),
		control.String(): tenantRowCounts(t, sp.db, control.String(), webhookIDs(t, sp.db, control.String())),
	}
	victimBefore := tenantRowCounts(t, sp.db, victim.String(), victimWebhooks)
	for table, n := range victimBefore {
		t.Logf("victim before purge: %s=%d", table, n)
		// Every tenant-owned table must hold rows, or the purge is not
		// really tested for it.
		require.Positive(t, n, "test setup left %s empty for the victim", table)
	}

	// The default tenant can never be purged.
	require.Error(t, sp.srv.PurgeTenant(ctx, uuid.MustParse(defaultTenant)))
	require.Error(t, sp.srv.PurgeTenant(ctx, uuid.Nil))

	require.NoError(t, sp.srv.PurgeTenant(ctx, victim))
	require.NoError(t, sp.srv.PurgeTenant(ctx, victim), "a second purge is a no-op")

	for table, n := range tenantRowCounts(t, sp.db, victim.String(), victimWebhooks) {
		require.Zero(t, n, "%s still has %d rows of the purged tenant", table, n)
	}
	require.Zero(t, liveJobs(t, sp.db, victim.String()), "the purged tenant still has runnable jobs")
	var tenants int
	require.NoError(t, sp.db.Get(&tenants, `SELECT count(*) FROM tenants WHERE id = $1`, victim))
	require.Zero(t, tenants)

	for tenantID, counts := range before {
		after := tenantRowCounts(t, sp.db, tenantID, webhookIDs(t, sp.db, tenantID))
		for table, n := range counts {
			// Deliveries keep retrying while the test runs, so the
			// failing webhook may add health events and buckets, never lose them.
			if after[table] < n {
				t.Fatalf("tenant %s lost rows in %s: %d before, %d after", tenantID, table, n, after[table])
			}
		}
	}
}

// webhookIDs lists a tenant's webhooks, including soft-deleted ones.
func webhookIDs(t *testing.T, db *sqlx.DB, tenantID string) []string {
	t.Helper()
	var ids []string
	require.NoError(t, db.Select(&ids, `SELECT id::text FROM webhook_registrations WHERE tenant_id = $1`, tenantID))
	return ids
}

// liveJobs counts a tenant's jobs that could still run.
func liveJobs(t *testing.T, db *sqlx.DB, tenantID string) int {
	t.Helper()
	var n int
	require.NoError(t, db.Get(&n, `
		SELECT count(*) FROM river_job
		WHERE args @> jsonb_build_object('tenant_id', $1::text)
		  AND state NOT IN ('completed', 'cancelled', 'discarded')`, tenantID))
	return n
}

// tenantRowCounts counts a tenant's rows in every public table that has a
// tenant_id column, every table keyed by webhook_id without one, and the
// access tables keyed by realm.
func tenantRowCounts(t *testing.T, db *sqlx.DB, tenantID string, webhooks []string) map[string]int {
	t.Helper()
	counts := map[string]int{}

	var tenantTables []string
	require.NoError(t, db.Select(&tenantTables, `
		SELECT table_name FROM information_schema.columns
		WHERE table_schema = 'public' AND column_name = 'tenant_id'`))
	require.NotEmpty(t, tenantTables)
	for _, table := range tenantTables {
		var n int
		require.NoError(t, db.Get(&n, fmt.Sprintf(`SELECT count(*) FROM %s WHERE tenant_id = $1`, pq.QuoteIdentifier(table)), tenantID))
		counts[table] = n
	}

	var webhookTables []string
	require.NoError(t, db.Select(&webhookTables, `
		SELECT c.table_name FROM information_schema.columns c
		WHERE c.table_schema = 'public' AND c.column_name = 'webhook_id'
		  AND NOT EXISTS (SELECT 1 FROM information_schema.columns t
		                  WHERE t.table_schema = 'public' AND t.table_name = c.table_name AND t.column_name = 'tenant_id')`))
	require.NotEmpty(t, webhookTables)
	for _, table := range webhookTables {
		var n int
		require.NoError(t, db.Get(&n, fmt.Sprintf(`SELECT count(*) FROM %s WHERE webhook_id::text = ANY($1)`, pq.QuoteIdentifier(table)), pq.Array(webhooks)))
		counts[table] = n
	}

	for _, table := range []string{"access_tokens", "access_invites"} {
		var n int
		require.NoError(t, db.Get(&n, fmt.Sprintf(`SELECT count(*) FROM %s WHERE realm = $1`, table), tenantID))
		counts[table] = n
	}
	return counts
}
