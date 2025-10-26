-- Drop time-series health tracking system
DROP TRIGGER IF EXISTS webhook_health_state_update_trigger ON webhook_health_events;
DROP FUNCTION IF EXISTS update_webhook_health_state();
DROP FUNCTION IF EXISTS calculate_webhook_health(UUID, INTEGER);
DROP FUNCTION IF EXISTS aggregate_webhook_health_hourly();

-- Drop tables in dependency order
DROP TABLE IF EXISTS webhook_health_summaries;
DROP TABLE IF EXISTS webhook_health_state;
DROP TABLE IF EXISTS webhook_health_events;

-- Remove health column from webhook_registrations
ALTER TABLE webhook_registrations DROP COLUMN IF EXISTS health;