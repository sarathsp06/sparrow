-- Drop GIN index on label_filters
DROP INDEX IF EXISTS idx_event_subscriptions_label_filters;

-- Remove label_filters column from event_subscriptions
ALTER TABLE event_subscriptions
    DROP COLUMN IF EXISTS label_filters;

-- Remove labels column from event_records
ALTER TABLE event_records
    DROP COLUMN IF EXISTS labels;
