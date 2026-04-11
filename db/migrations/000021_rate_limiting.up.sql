-- Add per-webhook rate limiting support.
--
-- rate_limit_rps: optional requests-per-second limit for webhook deliveries.
-- NULL means no limit (default). Positive values enable the leaky bucket.
ALTER TABLE webhook_registrations
    ADD COLUMN rate_limit_rps REAL CHECK (rate_limit_rps IS NULL OR rate_limit_rps > 0);

-- Leaky bucket state table: one row per rate-limited webhook.
-- next_delivery_at tracks the earliest time the next delivery can be sent.
-- Rows are created on first use and deleted when rate limiting is removed.
CREATE TABLE webhook_rate_limit_state (
    webhook_id       UUID PRIMARY KEY REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    next_delivery_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
