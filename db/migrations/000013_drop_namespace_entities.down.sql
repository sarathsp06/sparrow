-- 000013_drop_namespace_entities.down.sql
-- Recreates the namespaces table and FK dropped in the up migration.

-- Step 1: Recreate the namespaces table.
CREATE TABLE namespaces (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT namespaces_tenant_name_unique UNIQUE (tenant_id, name)
);

CREATE INDEX idx_namespaces_tenant_id ON namespaces(tenant_id);

-- Step 2: Backfill namespace records from existing webhook_registrations.
INSERT INTO namespaces (tenant_id, name, description)
SELECT DISTINCT wr.tenant_id, wr.namespace, ''
FROM webhook_registrations wr
WHERE wr.namespace IS NOT NULL AND wr.namespace != ''
ON CONFLICT (tenant_id, name) DO NOTHING;

-- Step 3: Restore the foreign key.
ALTER TABLE webhook_registrations
    ADD CONSTRAINT webhook_registrations_namespace_fk
    FOREIGN KEY (tenant_id, namespace)
    REFERENCES namespaces(tenant_id, name)
    ON DELETE CASCADE;
