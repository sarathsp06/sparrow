-- 000009_add_webhook_namespace_fk.up.sql
-- Adds a foreign key from webhook_registrations(tenant_id, namespace) to
-- namespaces(tenant_id, name) with ON DELETE CASCADE so that deleting a
-- namespace automatically cleans up its webhooks, subscriptions, deliveries,
-- and health data (via existing cascading FKs from those tables to
-- webhook_registrations).
--
-- Before adding the FK we backfill any namespaces that exist in
-- webhook_registrations but are missing from the namespaces table.

-- Step 1: Backfill missing namespace records.
INSERT INTO namespaces (tenant_id, name, description)
SELECT DISTINCT wr.tenant_id, wr.namespace, ''
FROM webhook_registrations wr
WHERE NOT EXISTS (
    SELECT 1 FROM namespaces n
    WHERE n.tenant_id = wr.tenant_id AND n.name = wr.namespace
)
ON CONFLICT (tenant_id, name) DO NOTHING;

-- Step 2: Add the foreign key constraint.
ALTER TABLE webhook_registrations
    ADD CONSTRAINT webhook_registrations_namespace_fk
    FOREIGN KEY (tenant_id, namespace)
    REFERENCES namespaces(tenant_id, name)
    ON DELETE CASCADE;
