-- 000010_scaling_and_audit_log.up.sql
-- Two-part migration:
--   1. Add missing composite indexes for high-throughput query paths
--   2. Create audit_logs table for tracking critical mutations

-- ============================================================================
-- Part 1: Scaling indexes
-- ============================================================================

-- Critical for EventProcessingWorker fan-out: GetSubscriptionsWithWebhooksByEvent()
-- JOINs event_subscriptions + webhook_registrations filtered by tenant_id, namespace, event_name.
-- The existing idx_event_subscriptions_event only covers (namespace, event_name).
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

-- ============================================================================
-- Part 2: Audit log table
-- ============================================================================

-- Actor type: how the caller authenticated
CREATE TYPE audit_actor_type AS ENUM ('api_key', 'user', 'system');

CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Who performed the action
    actor_id    TEXT NOT NULL,        -- SubjectID (JWT sub) or KeyID (API key UUID) or "system"
    actor_type  audit_actor_type NOT NULL,

    -- What was done
    action      TEXT NOT NULL,        -- e.g. "webhook.register", "subscription.delete"

    -- Which resource was affected
    resource_type TEXT NOT NULL,      -- e.g. "webhook", "event", "subscription", "tenant", "api_key", "namespace", "membership"
    resource_id   TEXT NOT NULL,      -- UUID or name of the resource

    -- Optional scoping
    namespace   TEXT NOT NULL DEFAULT '',

    -- Change details (before/after state, request parameters, etc.)
    metadata    JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Request context
    ip_address  TEXT NOT NULL DEFAULT '',

    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Primary query: list audit logs for a tenant, newest first
CREATE INDEX idx_audit_logs_tenant_created
    ON audit_logs(tenant_id, created_at DESC);

-- Filter by resource
CREATE INDEX idx_audit_logs_tenant_resource
    ON audit_logs(tenant_id, resource_type, resource_id);

-- Filter by action
CREATE INDEX idx_audit_logs_tenant_action
    ON audit_logs(tenant_id, action);

-- Filter by actor
CREATE INDEX idx_audit_logs_tenant_actor
    ON audit_logs(tenant_id, actor_id);

-- Filter by namespace
CREATE INDEX idx_audit_logs_tenant_namespace
    ON audit_logs(tenant_id, namespace, created_at DESC);

COMMENT ON TABLE audit_logs IS 'Immutable audit trail for all critical mutations (webhook, event, subscription, tenant, API key, namespace changes)';
COMMENT ON COLUMN audit_logs.actor_id IS 'JWT subject ID, API key UUID, or "system" for automated actions';
COMMENT ON COLUMN audit_logs.action IS 'Dot-separated action identifier, e.g. webhook.register, tenant.delete';
COMMENT ON COLUMN audit_logs.metadata IS 'JSONB with before/after state, request parameters, or other context';
