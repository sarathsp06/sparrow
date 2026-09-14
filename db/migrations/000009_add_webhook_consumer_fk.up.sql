-- 000009_add_webhook_consumer_fk.up.sql
-- Adds a foreign key from webhook_registrations(tenant_id, consumer) to
-- consumers(tenant_id, name) with ON DELETE CASCADE so that deleting a
-- consumer automatically cleans up its webhooks, subscriptions, deliveries,
-- and health data (via existing cascading FKs from those tables to
-- webhook_registrations).
--
-- Before adding the FK we backfill any consumers that exist in
-- webhook_registrations but are missing from the consumers table.

-- Step 1: Backfill missing consumer records.
INSERT INTO consumers (tenant_id, name, description)
SELECT DISTINCT wr.tenant_id, wr.consumer, ''
FROM webhook_registrations wr
WHERE NOT EXISTS (
    SELECT 1 FROM consumers n
    WHERE n.tenant_id = wr.tenant_id AND n.name = wr.consumer
)
ON CONFLICT (tenant_id, name) DO NOTHING;

-- Step 2: Add the foreign key constraint.
ALTER TABLE webhook_registrations
    ADD CONSTRAINT webhook_registrations_consumer_fk
    FOREIGN KEY (tenant_id, consumer)
    REFERENCES consumers(tenant_id, name)
    ON DELETE CASCADE;
