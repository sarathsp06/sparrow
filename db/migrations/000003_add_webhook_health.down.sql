-- Drop trigger and function
DROP TRIGGER IF EXISTS webhook_health_update_trigger ON webhook_health_metrics;
DROP FUNCTION IF EXISTS update_webhook_health();

-- Drop webhook_health_metrics table
DROP TABLE IF EXISTS webhook_health_metrics;

-- Drop indexes
DROP INDEX IF EXISTS idx_webhook_registrations_health;

-- Remove health column from webhook_registrations
ALTER TABLE webhook_registrations DROP COLUMN IF EXISTS health;