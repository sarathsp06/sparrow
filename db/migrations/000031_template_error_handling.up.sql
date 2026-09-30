-- Payload transform errors are visible and configurable per subscription.
--
-- on_transform_error: what happens when a subscription's template fails.
--   fail     (default) the delivery fails with error category template_error,
--            is not retried automatically, and can be retried by hand once the
--            template is fixed.
--   fallback the default envelope payload is sent instead; the error is still
--            recorded on the delivery.
-- template_missing_key: how a template reads a key the payload does not have.
--   error    (default) the render fails, so a field removed from the schema
--            cannot silently become "<no value>" in the delivered body.
--   zero     the key renders as "<no value>".
ALTER TABLE event_subscriptions
    ADD COLUMN on_transform_error VARCHAR(16) NOT NULL DEFAULT 'fail'
        CHECK (on_transform_error IN ('fail', 'fallback')),
    ADD COLUMN template_missing_key VARCHAR(8) NOT NULL DEFAULT 'error'
        CHECK (template_missing_key IN ('error', 'zero'));

-- The last template error for a delivery, if its transform failed.
ALTER TABLE webhook_deliveries ADD COLUMN template_error TEXT;
