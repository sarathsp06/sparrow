-- 000008_add_consumers_and_memberships.up.sql
-- Adds first-class consumer entities.
-- Consumers are sub-tenant scopes for organizing webhooks, subscriptions,
-- and deliveries.

-- Consumers table: first-class consumer entities within a tenant
CREATE TABLE consumers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT consumers_tenant_name_unique UNIQUE (tenant_id, name)
);

CREATE INDEX idx_consumers_tenant_id ON consumers(tenant_id);
