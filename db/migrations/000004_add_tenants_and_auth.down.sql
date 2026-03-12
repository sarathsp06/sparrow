BEGIN;

-- Remove tenant_id from existing tables
DROP INDEX IF EXISTS idx_event_records_tenant_namespace;
DROP INDEX IF EXISTS idx_event_records_tenant;
DROP INDEX IF EXISTS idx_event_subscriptions_tenant_namespace;
DROP INDEX IF EXISTS idx_event_subscriptions_tenant;
DROP INDEX IF EXISTS idx_webhook_registrations_tenant_namespace;
DROP INDEX IF EXISTS idx_webhook_registrations_tenant;
DROP INDEX IF EXISTS idx_event_registrations_tenant;

-- Restore original unique constraints
DROP INDEX IF EXISTS idx_event_subscriptions_tenant_unique;
ALTER TABLE event_subscriptions ADD CONSTRAINT event_subscriptions_webhook_id_event_name_namespace_key UNIQUE (webhook_id, event_name, namespace);

DROP INDEX IF EXISTS idx_webhook_registrations_tenant_namespace_url;
ALTER TABLE webhook_registrations ADD CONSTRAINT unique_namespace_url UNIQUE (namespace, url);

DROP INDEX IF EXISTS idx_event_registrations_tenant_name;
ALTER TABLE event_registrations ADD CONSTRAINT event_registrations_name_key UNIQUE (name);

-- Remove tenant_id columns
ALTER TABLE event_records DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE event_subscriptions DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE webhook_registrations DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE event_registrations DROP COLUMN IF EXISTS tenant_id;

-- Drop auth tables
DROP TABLE IF EXISTS api_keys;
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
DROP TABLE IF EXISTS tenants;

COMMIT;
