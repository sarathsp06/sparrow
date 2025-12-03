BEGIN;

ALTER TABLE webhook_deliveries 
ADD COLUMN subscription_id UUID REFERENCES event_subscriptions(id) ON DELETE SET NULL;

CREATE INDEX idx_webhook_deliveries_subscription_id ON webhook_deliveries(subscription_id);

COMMIT;
