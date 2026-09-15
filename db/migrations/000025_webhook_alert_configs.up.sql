-- Tenant-facing opt-in email alert recipients for Sparrow's self-generated
-- webhook.health_changed / webhook.delivery_failed system events. A NULL
-- webhook_id means "every webhook in this consumer"; a set webhook_id scopes
-- the alert to one webhook and cascades away with it.
CREATE TABLE webhook_alert_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    consumer VARCHAR(255) NOT NULL,
    webhook_id UUID REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    email VARCHAR(320) NOT NULL,
    event_types TEXT[] NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT webhook_alert_configs_event_types_check CHECK (array_length(event_types, 1) > 0)
);

CREATE INDEX idx_webhook_alert_configs_lookup ON webhook_alert_configs(tenant_id, consumer, webhook_id);

COMMENT ON TABLE webhook_alert_configs IS 'Opt-in email recipients for Sparrow-generated webhook health/delivery-failure alerts.';
COMMENT ON COLUMN webhook_alert_configs.webhook_id IS 'NULL = consumer-wide (every webhook in the consumer); set = scoped to one webhook.';
