-- DLQ view: terminal-failed deliveries are queried per webhook (list + depth
-- gauge). Partial index keeps those lookups cheap regardless of table size.
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_dlq
    ON webhook_deliveries (webhook_id, created_at DESC)
    WHERE status = 'failed';
