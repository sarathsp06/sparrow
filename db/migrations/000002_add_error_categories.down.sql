ALTER TABLE webhook_health_events DROP COLUMN IF EXISTS error_category;

DROP INDEX IF EXISTS idx_webhook_health_events_error_category;
DROP INDEX IF EXISTS idx_webhook_health_events_webhook_category;

ALTER TABLE webhook_health_summaries
    DROP COLUMN IF EXISTS client_errors,
    DROP COLUMN IF EXISTS server_errors,
    DROP COLUMN IF EXISTS timeout_errors,
    DROP COLUMN IF EXISTS network_errors;
