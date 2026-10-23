package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/storage"
)

// TenantRepository creates tenants and erases them. Self-hosted Sparrow has
// one tenant and never calls it; it serves programs embedding Sparrow
// (server.CreateTenant, server.PurgeTenant).
type TenantRepository interface {
	// CreateTenant inserts the tenant row if absent.
	CreateTenant(ctx context.Context, tenantID uuid.UUID, name string) error
	// PurgeTenant deletes every row that belongs to the tenant, then the
	// tenant itself. It is idempotent and safe to run again after a failure.
	PurgeTenant(ctx context.Context, tenantID uuid.UUID) error
}

// purgeChunk bounds each delete statement so a large tenant is erased in
// many short transactions rather than one cascade that holds locks for
// seconds (see migration 000041: a 600k-delivery cascade took about 8s).
const purgeChunk = 5000

// CreateTenant implements TenantRepository. The slug is the tenant id: the
// embedding program owns names and slugs for its own users.
func (r *Repository) CreateTenant(ctx context.Context, tenantID uuid.UUID, name string) error {
	_, err := r.conn.ExecContext(ctx, `
		INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)
		ON CONFLICT (id) DO NOTHING
	`, tenantID, name, tenantID.String())
	return storage.Error(err)
}

// PurgeTenant implements TenantRepository. The tenant's queued jobs must be
// cancelled first (see queue.Manager.CancelTenantJobs) and its traffic
// stopped, or workers may write new rows while the purge runs.
//
// Order matters: the bulky webhook-keyed tables go first, in chunks; then
// the tables a tenant delete would not reach (no foreign key to tenants,
// or one that restricts the delete); then the tenant row, whose cascade
// removes the rest (webhooks with their state, subscriptions, events,
// event types, batch jobs).
func (r *Repository) PurgeTenant(ctx context.Context, tenantID uuid.UUID) error {
	chunked := []struct{ name, query string }{
		{"deliveries", `
			DELETE FROM webhook_deliveries WHERE id IN (
				SELECT wd.id FROM webhook_deliveries wd
				JOIN webhook_registrations wr ON wr.id = wd.webhook_id
				WHERE wr.tenant_id = $1 LIMIT $2)`},
		{"health events", `
			DELETE FROM webhook_health_events WHERE id IN (
				SELECT he.id FROM webhook_health_events he
				JOIN webhook_registrations wr ON wr.id = he.webhook_id
				WHERE wr.tenant_id = $1 LIMIT $2)`},
		{"events", `
			DELETE FROM event_records WHERE id IN (
				SELECT id FROM event_records WHERE tenant_id = $1 LIMIT $2)`},
	}
	for _, c := range chunked {
		for {
			res, err := r.conn.ExecContext(ctx, c.query, tenantID, purgeChunk)
			if err != nil {
				return fmt.Errorf("purge tenant %s: %w", c.name, storage.Error(err))
			}
			n, err := res.RowsAffected()
			if err != nil {
				return storage.Error(err)
			}
			if n < purgeChunk {
				break
			}
		}
	}

	return storage.WithTransaction(r.db, func(tx storage.DBTX) error {
		for _, q := range []struct{ name, query string }{
			// No foreign key: buckets keep only a webhook id (migration 000040).
			{"health buckets", `
				DELETE FROM webhook_health_buckets b
				USING webhook_registrations wr
				WHERE wr.id = b.webhook_id AND wr.tenant_id = $1`},
			// No foreign key to tenants.
			{"alert configs", `DELETE FROM webhook_alert_configs WHERE tenant_id = $1`},
			{"template versions", `DELETE FROM subscription_template_versions WHERE tenant_id = $1`},
			// Restricts deleting its event type, so it must go first.
			{"event type versions", `DELETE FROM event_registration_versions WHERE tenant_id = $1`},
			{"tenant", `DELETE FROM tenants WHERE id = $1`},
		} {
			if _, err := tx.ExecContext(ctx, q.query, tenantID); err != nil {
				return fmt.Errorf("purge tenant %s: %w", q.name, storage.Error(err))
			}
		}
		return nil
	})
}
