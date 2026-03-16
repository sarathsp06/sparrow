-- 000008_add_namespaces_and_memberships.up.sql
-- Adds first-class namespace entities and namespace membership (user-namespace role assignments).
--
-- Namespaces are sub-tenant scopes. Previously they were ad-hoc strings on webhooks;
-- now they are registered entities with CRUD operations.
--
-- Namespace memberships map users (identified by subject_id from JWT) to specific
-- namespaces with a role. When a user has memberships, they can only access those
-- namespaces — even if they have a tenant-level role like tenant:admin.

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

-- Namespace memberships: user-namespace role assignments
CREATE TABLE namespace_memberships (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subject_id  TEXT NOT NULL,  -- JWT sub claim (user identity)
    namespace   TEXT NOT NULL,  -- Namespace name (matches namespaces.name)
    role        TEXT NOT NULL,  -- namespace:admin, namespace:member, namespace:viewer
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT namespace_memberships_unique UNIQUE (tenant_id, subject_id, namespace),
    CONSTRAINT namespace_memberships_role_check CHECK (
        role IN ('namespace:admin', 'namespace:member', 'namespace:viewer')
    ),
    -- Ensure the namespace exists within the same tenant
    CONSTRAINT namespace_memberships_namespace_fk
        FOREIGN KEY (tenant_id, namespace)
        REFERENCES namespaces(tenant_id, name)
        ON DELETE CASCADE
);

CREATE INDEX idx_namespace_memberships_tenant_subject ON namespace_memberships(tenant_id, subject_id);
CREATE INDEX idx_namespace_memberships_tenant_namespace ON namespace_memberships(tenant_id, namespace);
