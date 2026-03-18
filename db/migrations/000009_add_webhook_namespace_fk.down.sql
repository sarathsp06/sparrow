-- 000009_add_webhook_namespace_fk.down.sql
-- Drops the foreign key added in the up migration.
-- Does NOT remove backfilled namespace rows because they may now be in use.

ALTER TABLE webhook_registrations
    DROP CONSTRAINT IF EXISTS webhook_registrations_namespace_fk;
