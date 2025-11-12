-- Remove request_body column from webhook_deliveries table
DROP INDEX IF EXISTS idx_webhook_deliveries_request_body_gin;
ALTER TABLE webhook_deliveries DROP COLUMN IF EXISTS request_body;