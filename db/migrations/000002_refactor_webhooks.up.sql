BEGIN;

-- Remove events column from webhook_registrations
ALTER TABLE webhook_registrations DROP COLUMN IF EXISTS events;

-- Create event_subscriptions table
CREATE TABLE IF NOT EXISTS event_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    event_name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    headers JSONB,
    method VARCHAR(10) DEFAULT 'POST',
    transform_enabled BOOLEAN DEFAULT FALSE,
    transform_template TEXT,
    timeout INT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(webhook_id, event_name, namespace)
);

CREATE INDEX IF NOT EXISTS idx_event_subscriptions_event ON event_subscriptions(namespace, event_name);

COMMIT;
