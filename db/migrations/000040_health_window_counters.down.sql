DELETE FROM webhook_health_buckets b
WHERE NOT EXISTS (SELECT 1 FROM webhook_registrations wr WHERE wr.id = b.webhook_id);
ALTER TABLE webhook_health_buckets
    ADD CONSTRAINT webhook_health_buckets_webhook_id_fkey
    FOREIGN KEY (webhook_id) REFERENCES webhook_registrations(id) ON DELETE CASCADE;

ALTER TABLE webhook_metrics
    DROP COLUMN IF EXISTS window_failures,
    DROP COLUMN IF EXISTS window_attempts;
