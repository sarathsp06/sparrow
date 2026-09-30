DROP INDEX IF EXISTS idx_webhook_deliveries_paused;
ALTER TABLE event_subscriptions
    DROP COLUMN IF EXISTS paused_reason,
    DROP COLUMN IF EXISTS paused_at;
