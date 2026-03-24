-- 000010_scaling_and_audit_log.up.sql
-- Add missing composite indexes for high-throughput query paths

-- ============================================================================
-- Scaling indexes
-- ============================================================================

-- Critical for EventProcessingWorker fan-out: GetSubscriptionsWithWebhooksByEvent()
-- JOINs event_subscriptions + webhook_registrations filtered by tenant_id, namespace, event_name.
CREATE INDEX IF NOT EXISTS idx_event_subscriptions_tenant_ns_event
    ON event_subscriptions(tenant_id, namespace, event_name);

-- Paginated delivery listing by webhook (GetDeliveriesByWebhookID)
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_created
    ON webhook_deliveries(webhook_id, created_at DESC);

-- Paginated event record listing (ListEventReports, ListEventReportsWithStats)
CREATE INDEX IF NOT EXISTS idx_event_records_tenant_ns_created
    ON event_records(tenant_id, namespace, created_at DESC);

-- Event record filtering by event name (used when eventName filter is provided)
CREATE INDEX IF NOT EXISTS idx_event_records_tenant_event_created
    ON event_records(tenant_id, event, created_at DESC);

-- Paginated delivery listing by event (GetDeliveriesByEventPaginated)
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_event_created
    ON webhook_deliveries(event_id, created_at DESC);

-- Filtered webhook listing (ListWebhooks with activeOnly=true)
CREATE INDEX IF NOT EXISTS idx_webhook_registrations_tenant_ns_active
    ON webhook_registrations(tenant_id, namespace, active);

-- Drop unused GIN index on webhook_deliveries.request_body — no query uses
-- full-text search on this column, and it adds write overhead on every INSERT.
DROP INDEX IF EXISTS idx_webhook_deliveries_request_body_gin;
