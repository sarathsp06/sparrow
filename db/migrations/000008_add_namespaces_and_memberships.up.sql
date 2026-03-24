-- 000008_add_namespaces_and_memberships.up.sql
-- Adds first-class namespace entities.
-- Namespaces are sub-tenant scopes for organizing webhooks, subscriptions,
-- and deliveries.

-- Namespaces table: first-class namespace entities within a tenant
CREATE TABLE namespaces (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT namespaces_tenant_name_unique UNIQUE (tenant_id, name)
);

CREATE INDEX idx_namespaces_tenant_id ON namespaces(tenant_id);
