-- 000013_drop_namespace_entities.up.sql
-- Simplifies namespaces: removes the managed namespaces table and foreign key.
-- The namespace VARCHAR column stays on webhook_registrations, event_subscriptions,
-- and event_records as a free-form string field.

-- Step 1: Drop the foreign key from webhook_registrations to namespaces.
ALTER TABLE webhook_registrations
    DROP CONSTRAINT IF EXISTS webhook_registrations_namespace_fk;

-- Step 2: Drop the namespaces table (no longer needed).
DROP TABLE IF EXISTS namespaces;
