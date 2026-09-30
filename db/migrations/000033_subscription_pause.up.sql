-- Pausing a subscription stops its deliveries from being attempted without
-- losing them: fan-out still creates each delivery, with status 'paused' and
-- no queued job, so what was not delivered stays visible and can be retried.
-- A pause is not a receiver failure and never affects webhook health.
ALTER TABLE event_subscriptions
    ADD COLUMN paused_at TIMESTAMPTZ,
    ADD COLUMN paused_reason TEXT;

CREATE INDEX idx_webhook_deliveries_paused
    ON webhook_deliveries (subscription_id, created_at)
    WHERE status = 'paused';
