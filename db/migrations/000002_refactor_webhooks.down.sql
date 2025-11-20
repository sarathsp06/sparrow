BEGIN;

-- Drop event_subscriptions table
DROP TABLE IF EXISTS event_subscriptions;

-- Add events column back to webhook_registrations
ALTER TABLE webhook_registrations ADD COLUMN events TEXT[];

COMMIT;
