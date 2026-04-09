-- Add idempotency_key column to event_records for client-provided deduplication.
-- The column is nullable: events without an idempotency key (including re-pushes)
-- are never constrained by the unique index.
ALTER TABLE event_records ADD COLUMN idempotency_key VARCHAR(255);

-- Partial unique index: only enforced when idempotency_key IS NOT NULL.
-- Scoped by (tenant_id, namespace) so the same key can be used independently
-- in different namespaces.
CREATE UNIQUE INDEX idx_event_records_idempotency_key
    ON event_records (tenant_id, namespace, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
