package store

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// maxConsumerScan bounds how many distinct consumer names ListConsumers walks
// per table before filtering. Consumers are implicit (a name on webhooks and
// events), so this is the only place that enumerates them.
const maxConsumerScan = 10000

// ListConsumers returns up to limit consumer names, sorted, that own a webhook
// or an event in the tenant and contain query (case-insensitive; empty matches
// all). Distinct names come from a loose index scan over the (tenant_id,
// consumer) indexes, which costs one index probe per consumer rather than a
// read of every event.
func (r *Repository) ListConsumers(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]string, error) {
	q := `
		WITH RECURSIVE
		hooks AS (
			(SELECT consumer FROM webhook_registrations WHERE tenant_id = $1 ORDER BY consumer LIMIT 1)
			UNION ALL
			SELECT (SELECT w.consumer FROM webhook_registrations w
			        WHERE w.tenant_id = $1 AND w.consumer > h.consumer ORDER BY w.consumer LIMIT 1)
			FROM hooks h WHERE h.consumer IS NOT NULL
		),
		events AS (
			(SELECT consumer FROM event_records WHERE tenant_id = $1 ORDER BY consumer LIMIT 1)
			UNION ALL
			SELECT (SELECT e.consumer FROM event_records e
			        WHERE e.tenant_id = $1 AND e.consumer > v.consumer ORDER BY e.consumer LIMIT 1)
			FROM events v WHERE v.consumer IS NOT NULL
		),
		names AS (
			(SELECT consumer FROM hooks WHERE consumer IS NOT NULL LIMIT $4)
			UNION
			(SELECT consumer FROM events WHERE consumer IS NOT NULL LIMIT $4)
		)
		SELECT consumer FROM names
		WHERE $2 = '' OR strpos(lower(consumer), $2) > 0
		ORDER BY consumer
		LIMIT $3
	`
	var names []string
	if err := r.conn.SelectContext(ctx, &names, q, tenantID, strings.ToLower(strings.TrimSpace(query)), limit, maxConsumerScan); err != nil {
		return nil, storage.Error(err)
	}
	return names, nil
}
