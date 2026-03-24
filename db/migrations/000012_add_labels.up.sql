-- Add labels column to event_records for label-based subscription matching
ALTER TABLE event_records
    ADD COLUMN labels JSONB NOT NULL DEFAULT '{}';

-- Add label_filters column to event_subscriptions for label-based subscription matching
ALTER TABLE event_subscriptions
    ADD COLUMN label_filters JSONB NOT NULL DEFAULT '{}';

-- GIN index on label_filters for fast JSONB containment queries (@> / <@)
CREATE INDEX idx_event_subscriptions_label_filters ON event_subscriptions USING GIN (label_filters);
