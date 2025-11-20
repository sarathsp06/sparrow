BEGIN;

DROP INDEX IF EXISTS idx_webhook_deliveries_subscription_id;
ALTER TABLE webhook_deliveries DROP COLUMN IF EXISTS subscription_id;

COMMIT;
