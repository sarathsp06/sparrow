-- Every delivery attempt updates status, error_category and the response
-- columns, and an update that changes an indexed column writes a new entry
-- into every index on the table. Only the primary key serves the delivery
-- path; the rest serve the UI, retries and retention. Measured on 6M
-- deliveries, these five can go without slowing any query:
--   webhook_id, event_id: left prefixes of (webhook_id, created_at) and
--     (event_id, created_at), which also serve the foreign keys;
--   status, error_category: UI-only filters; a list page walks newest-first
--     and stops after one page;
--   paused: the (subscription_id) key never matched how the resume counts
--     query (through the consumer's webhooks).
DROP INDEX IF EXISTS idx_webhook_deliveries_webhook_id;
DROP INDEX IF EXISTS idx_webhook_deliveries_event_id;
DROP INDEX IF EXISTS idx_webhook_deliveries_status;
DROP INDEX IF EXISTS idx_webhook_deliveries_error_category;
DROP INDEX IF EXISTS idx_webhook_deliveries_paused;

-- One small index over the ~10% of deliveries that did not succeed serves
-- every status-filtered lookup by webhook: retry snapshots, resume counts of
-- paused deliveries, stats' in-flight count (index-only) and retriable
-- deliveries. Status must reach it as a webhook_delivery_status parameter,
-- not text cast to the enum, or the planner cannot match the predicate.
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_unsuccessful
    ON webhook_deliveries (webhook_id, status, created_at DESC)
    WHERE status <> 'success';
