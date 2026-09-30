ALTER TABLE webhook_deliveries DROP COLUMN IF EXISTS template_error;
ALTER TABLE event_subscriptions
    DROP COLUMN IF EXISTS template_missing_key,
    DROP COLUMN IF EXISTS on_transform_error;
