-- Deleting a webhook used to cascade through every delivery and health event
-- that referenced it (8s and a held row lock for a webhook with 600k
-- deliveries). Webhooks are now soft-deleted: deleted_at is set, the row is
-- hidden from every webhook read and from fan-out, and its deliveries,
-- health history and subscriptions are kept, so no foreign key is touched.
-- webhook_registrations holds one row per endpoint, so the filter needs no
-- index of its own.
ALTER TABLE webhook_registrations ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- A deleted webhook's URL can be registered again in the same consumer.
DROP INDEX IF EXISTS idx_webhook_registrations_tenant_consumer_url;
CREATE UNIQUE INDEX IF NOT EXISTS idx_webhook_registrations_tenant_consumer_url
    ON webhook_registrations (tenant_id, consumer, url)
    WHERE deleted_at IS NULL;
