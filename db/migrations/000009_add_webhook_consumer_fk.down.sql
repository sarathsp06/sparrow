-- 000009_add_webhook_consumer_fk.down.sql
-- Drops the foreign key added in the up migration.
-- Does NOT remove backfilled consumer rows because they may now be in use.

ALTER TABLE webhook_registrations
    DROP CONSTRAINT IF EXISTS webhook_registrations_consumer_fk;
