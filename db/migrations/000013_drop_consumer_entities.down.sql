-- 000013_drop_consumer_entities.down.sql
-- Recreates the consumers table and FK dropped in the up migration.

-- Step 1: Recreate the consumers table.
CREATE TABLE consumers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT consumers_tenant_name_unique UNIQUE (tenant_id, name)
);

CREATE INDEX idx_consumers_tenant_id ON consumers(tenant_id);

-- Step 2: Backfill consumer records from existing webhook_registrations.
INSERT INTO consumers (tenant_id, name, description)
SELECT DISTINCT wr.tenant_id, wr.consumer, ''
FROM webhook_registrations wr
WHERE wr.consumer IS NOT NULL AND wr.consumer != ''
ON CONFLICT (tenant_id, name) DO NOTHING;

-- Step 3: Restore the foreign key.
ALTER TABLE webhook_registrations
    ADD CONSTRAINT webhook_registrations_consumer_fk
    FOREIGN KEY (tenant_id, consumer)
    REFERENCES consumers(tenant_id, name)
    ON DELETE CASCADE;
