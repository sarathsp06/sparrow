-- Consumer and global stats used to count every webhook_deliveries row on
-- each request (4.8s for a busy consumer on 6M deliveries). The health
-- evaluator now keeps running per-webhook counters in their own table, adding
-- each pass's outcomes, so stats sum at most one row per webhook and nothing
-- is recounted from history.
CREATE TABLE IF NOT EXISTS webhook_metrics (
    webhook_id     UUID PRIMARY KEY REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    total_attempts BIGINT NOT NULL DEFAULT 0,
    total_failures BIGINT NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed the totals from the outcomes the evaluator has already folded in
-- (everything up to its watermark); later passes add what comes after.
INSERT INTO webhook_metrics (webhook_id, total_attempts, total_failures)
SELECT e.webhook_id, COUNT(*), COUNT(*) FILTER (WHERE NOT e.success)
FROM webhook_health_events e
JOIN webhook_registrations wr ON wr.id = e.webhook_id
WHERE e.timestamp <= (SELECT watermark FROM webhook_health_evaluator WHERE singleton)
GROUP BY e.webhook_id
ON CONFLICT (webhook_id) DO NOTHING;
