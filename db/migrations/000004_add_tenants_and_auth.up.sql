BEGIN;

-- ============================================================================
-- Phase 1: Create tenants table
-- ============================================================================
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'active',
    settings JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT tenant_status_check CHECK (status IN ('active', 'suspended', 'archived'))
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_status ON tenants(status);

COMMENT ON TABLE tenants IS 'Organizations/teams using Sparrow. Each tenant has isolated namespaces, events, and webhooks.';
COMMENT ON COLUMN tenants.slug IS 'URL-safe identifier for the tenant, used in API key prefixes';
COMMENT ON COLUMN tenants.settings IS 'Per-tenant configuration (rate limits, quotas, feature flags)';

-- Create updated_at trigger for tenants
CREATE TRIGGER update_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- Phase 2: Create api_keys table
-- ============================================================================
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    key_prefix TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    role TEXT NOT NULL,
    namespace_scope TEXT,
    is_platform_admin BOOLEAN NOT NULL DEFAULT false,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ,
    CONSTRAINT api_key_role_check CHECK (role IN (
        'tenant:admin', 'tenant:member',
        'namespace:admin', 'namespace:member', 'namespace:viewer'
    )),
    CONSTRAINT api_key_namespace_scope_check CHECK (
        (role IN ('namespace:admin', 'namespace:member', 'namespace:viewer') AND namespace_scope IS NOT NULL)
        OR
        (role IN ('tenant:admin', 'tenant:member') AND namespace_scope IS NULL)
    )
);

CREATE INDEX idx_api_keys_prefix ON api_keys(key_prefix);
CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);

COMMENT ON TABLE api_keys IS 'API keys for authenticating programmatic access. Keys are hashed; the plaintext is shown only once at creation.';
COMMENT ON COLUMN api_keys.key_prefix IS 'First segment of the key (e.g. sk_default_) for identification without exposing the full key';
COMMENT ON COLUMN api_keys.key_hash IS 'SHA-256 hash of the full API key';
COMMENT ON COLUMN api_keys.role IS 'RBAC role assigned to this key';
COMMENT ON COLUMN api_keys.namespace_scope IS 'For namespace-scoped roles, which namespace this key has access to. NULL for tenant-wide roles.';

-- ============================================================================
-- Phase 3: Insert default tenant for backward compatibility
-- ============================================================================
INSERT INTO tenants (id, name, slug, status)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default', 'default', 'active');

-- ============================================================================
-- Phase 4: Add tenant_id to existing tables and backfill
-- ============================================================================

-- event_registrations: add tenant_id (events become tenant-scoped)
ALTER TABLE event_registrations
    ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;

UPDATE event_registrations SET tenant_id = '00000000-0000-0000-0000-000000000001';

ALTER TABLE event_registrations
    ALTER COLUMN tenant_id SET NOT NULL;

-- Drop the old unique constraint and create a new tenant-scoped one
ALTER TABLE event_registrations DROP CONSTRAINT IF EXISTS event_registrations_name_key;
CREATE UNIQUE INDEX idx_event_registrations_tenant_name ON event_registrations(tenant_id, name);

-- webhook_registrations: add tenant_id
ALTER TABLE webhook_registrations
    ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;

UPDATE webhook_registrations SET tenant_id = '00000000-0000-0000-0000-000000000001';

ALTER TABLE webhook_registrations
    ALTER COLUMN tenant_id SET NOT NULL;

-- Drop old unique constraint and create new tenant-scoped one
ALTER TABLE webhook_registrations DROP CONSTRAINT IF EXISTS unique_namespace_url;
CREATE UNIQUE INDEX idx_webhook_registrations_tenant_namespace_url ON webhook_registrations(tenant_id, namespace, url);

-- event_subscriptions: add tenant_id
ALTER TABLE event_subscriptions
    ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;

UPDATE event_subscriptions SET tenant_id = '00000000-0000-0000-0000-000000000001';

ALTER TABLE event_subscriptions
    ALTER COLUMN tenant_id SET NOT NULL;

-- Drop old unique constraint and create new tenant-scoped one
ALTER TABLE event_subscriptions DROP CONSTRAINT IF EXISTS event_subscriptions_webhook_id_event_name_namespace_key;
CREATE UNIQUE INDEX idx_event_subscriptions_tenant_unique ON event_subscriptions(tenant_id, webhook_id, event_name, namespace);

-- event_records: add tenant_id
ALTER TABLE event_records
    ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;

UPDATE event_records SET tenant_id = '00000000-0000-0000-0000-000000000001';

ALTER TABLE event_records
    ALTER COLUMN tenant_id SET NOT NULL;

-- ============================================================================
-- Phase 5: Add composite indexes for tenant-scoped queries
-- ============================================================================
CREATE INDEX idx_event_registrations_tenant ON event_registrations(tenant_id);
CREATE INDEX idx_webhook_registrations_tenant ON webhook_registrations(tenant_id);
CREATE INDEX idx_webhook_registrations_tenant_namespace ON webhook_registrations(tenant_id, namespace);
CREATE INDEX idx_event_subscriptions_tenant ON event_subscriptions(tenant_id);
CREATE INDEX idx_event_subscriptions_tenant_namespace ON event_subscriptions(tenant_id, namespace);
CREATE INDEX idx_event_records_tenant ON event_records(tenant_id);
CREATE INDEX idx_event_records_tenant_namespace ON event_records(tenant_id, namespace);

COMMIT;
