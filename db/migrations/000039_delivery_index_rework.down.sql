DROP INDEX IF EXISTS idx_webhook_deliveries_unsuccessful;

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id ON webhook_deliveries (webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_event_id ON webhook_deliveries (event_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status ON webhook_deliveries (status);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_error_category ON webhook_deliveries (error_category)
    WHERE error_category <> '' AND error_category <> 'success';
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_paused ON webhook_deliveries (subscription_id, created_at)
    WHERE status = 'paused';
