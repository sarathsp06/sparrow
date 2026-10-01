-- Saved history of a subscription's transform_template. A row is written
-- whenever a save changes the template (create with a template, or update
-- with a different one), so the newest row is the current template and the
-- rows after it are what it replaced. The service keeps the last 20 per
-- subscription. Deleting the subscription deletes its history.
CREATE TABLE subscription_template_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    subscription_id UUID NOT NULL REFERENCES event_subscriptions(id) ON DELETE CASCADE,
    template TEXT NOT NULL,
    source VARCHAR(32) NOT NULL DEFAULT 'manual',
    notes TEXT NOT NULL DEFAULT '',
    saved_by VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscription_template_versions_lookup
    ON subscription_template_versions(tenant_id, subscription_id, created_at DESC);

COMMENT ON TABLE subscription_template_versions IS 'History of saved transform templates per subscription (newest = current), capped at 20 by the service.';
COMMENT ON COLUMN subscription_template_versions.source IS 'manual or ai_draft: how the template was produced before it was saved.';
