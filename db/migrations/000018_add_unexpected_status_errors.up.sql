-- Add unexpected_status_errors column to webhook_health_summaries
-- Tracks deliveries where HTTP response was 2xx/3xx but not in the webhook's expected_status_codes list.
ALTER TABLE webhook_health_summaries
    ADD COLUMN unexpected_status_errors INTEGER NOT NULL DEFAULT 0;
