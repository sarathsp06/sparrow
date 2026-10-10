-- Soft-deleted webhooks are removed for real (cascading as deletes did
-- before), so the full unique index can be restored.
DELETE FROM webhook_registrations WHERE deleted_at IS NOT NULL;
DROP INDEX IF EXISTS idx_webhook_registrations_tenant_consumer_url;
CREATE UNIQUE INDEX IF NOT EXISTS idx_webhook_registrations_tenant_consumer_url
    ON webhook_registrations (tenant_id, consumer, url);
ALTER TABLE webhook_registrations DROP COLUMN IF EXISTS deleted_at;
