-- 000010_scaling_and_audit_log.down.sql
-- Reverses the scaling indexes and audit_logs table.

-- Drop audit log table and type
DROP TABLE IF EXISTS audit_logs;
DROP TYPE IF EXISTS audit_actor_type;

-- Drop scaling indexes
DROP INDEX IF EXISTS idx_event_subscriptions_tenant_ns_event;
DROP INDEX IF EXISTS idx_webhook_deliveries_webhook_created;
DROP INDEX IF EXISTS idx_event_records_tenant_ns_created;
DROP INDEX IF EXISTS idx_event_records_tenant_event_created;
DROP INDEX IF EXISTS idx_webhook_deliveries_event_created;
DROP INDEX IF EXISTS idx_webhook_registrations_tenant_ns_active;

-- Restore the GIN index that was dropped in the up migration
CREATE INDEX idx_webhook_deliveries_request_body_gin
    ON webhook_deliveries USING gin(to_tsvector('english', request_body));
