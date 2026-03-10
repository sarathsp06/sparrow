-- Reverse migration: remove error_category from webhook_deliveries

DROP INDEX IF EXISTS idx_webhook_deliveries_error_category;

ALTER TABLE webhook_deliveries
    DROP COLUMN IF EXISTS error_category;
