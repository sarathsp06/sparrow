-- The evaluator used to re-sum up to 24h of per-minute buckets for every
-- webhook it touched, every pass, to get the success rate behind the health
-- label. It now keeps that window as two running counters in webhook_metrics:
-- each pass adds its new outcomes and subtracts the buckets that slid out of
-- the window, so a pass reads about one minute of buckets instead of 24 hours.
ALTER TABLE webhook_metrics
    ADD COLUMN IF NOT EXISTS window_attempts BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS window_failures BIGINT NOT NULL DEFAULT 0;

-- Start from the window as of the evaluator's watermark (24h: the label
-- window, store.DefaultHealthRules.WindowHours).
INSERT INTO webhook_metrics (webhook_id, window_attempts, window_failures)
SELECT b.webhook_id, SUM(b.attempts), SUM(b.failures)
FROM webhook_health_buckets b
JOIN webhook_registrations wr ON wr.id = b.webhook_id
WHERE b.bucket_start >= date_trunc('minute', (SELECT watermark FROM webhook_health_evaluator WHERE singleton)) - INTERVAL '24 hours'
GROUP BY b.webhook_id
ON CONFLICT (webhook_id) DO UPDATE SET
    window_attempts = EXCLUDED.window_attempts,
    window_failures = EXCLUDED.window_failures;

-- The rollup only inserts buckets for existing webhooks (it joins
-- webhook_registrations) and buckets expire after 25h, so the foreign key
-- only cost a lookup per inserted bucket (about 40% of the rollup).
ALTER TABLE webhook_health_buckets DROP CONSTRAINT IF EXISTS webhook_health_buckets_webhook_id_fkey;
