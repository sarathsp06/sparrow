-- 000013_drop_consumer_entities.up.sql
-- Simplifies consumers: removes the managed consumers table and foreign key.
-- The consumer VARCHAR column stays on webhook_registrations, event_subscriptions,
-- and event_records as a free-form string field.

-- Step 1: Drop the foreign key from webhook_registrations to consumers.
ALTER TABLE webhook_registrations
    DROP CONSTRAINT IF EXISTS webhook_registrations_consumer_fk;

-- Step 2: Drop the consumers table (no longer needed).
DROP TABLE IF EXISTS consumers;
