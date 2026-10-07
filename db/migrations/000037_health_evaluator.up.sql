-- Webhook health is evaluated periodically from webhook_health_events instead
-- of on every delivery attempt. Per-minute rollups keep the 24h window cheap
-- to sum; the evaluator's single-row state holds how far it has read.
CREATE TABLE IF NOT EXISTS webhook_health_buckets (
    webhook_id   UUID NOT NULL REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    bucket_start TIMESTAMP WITH TIME ZONE NOT NULL,
    attempts     INTEGER NOT NULL DEFAULT 0,
    failures     INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (webhook_id, bucket_start)
);
CREATE INDEX IF NOT EXISTS idx_webhook_health_buckets_start ON webhook_health_buckets(bucket_start);

CREATE TABLE IF NOT EXISTS webhook_health_evaluator (
    singleton  BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    watermark  TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
