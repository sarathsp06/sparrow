-- The health label used to be recomputed on every delivery with a 24h scan
-- of webhook_health_events. It is now recomputed at most every few seconds
-- per webhook, and immediately when the delivery outcome flips; this column
-- records when it was last computed.
ALTER TABLE webhook_health_state
    ADD COLUMN IF NOT EXISTS health_label_computed_at TIMESTAMP WITH TIME ZONE;
