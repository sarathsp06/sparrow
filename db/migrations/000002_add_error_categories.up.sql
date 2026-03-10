BEGIN;

-- Add error_category column to webhook_health_events for error classification
-- Categories: success, client_error, server_error, timeout, dns_error, tls_error, connection_refused, network_error, unknown
ALTER TABLE webhook_health_events
    ADD COLUMN error_category VARCHAR(30) NOT NULL DEFAULT 'unknown';

-- Backfill existing records based on available data
UPDATE webhook_health_events
SET error_category = CASE
    WHEN success = true THEN 'success'
    WHEN response_code >= 400 AND response_code < 500 THEN 'client_error'
    WHEN response_code >= 500 THEN 'server_error'
    WHEN error_message ILIKE '%timeout%' THEN 'timeout'
    WHEN error_message ILIKE '%connection refused%' THEN 'connection_refused'
    WHEN error_message ILIKE '%no such host%' OR error_message ILIKE '%dns%' THEN 'dns_error'
    WHEN error_message ILIKE '%tls%' OR error_message ILIKE '%certificate%' OR error_message ILIKE '%x509%' THEN 'tls_error'
    WHEN error_message ILIKE '%connection reset%' OR error_message ILIKE '%broken pipe%' 
         OR error_message ILIKE '%unreachable%' THEN 'network_error'
    ELSE 'unknown'
END;

-- Index for filtering/aggregating by error category
CREATE INDEX idx_webhook_health_events_error_category ON webhook_health_events(error_category);

-- Composite index for per-webhook category aggregation
CREATE INDEX idx_webhook_health_events_webhook_category ON webhook_health_events(webhook_id, error_category, timestamp DESC);

-- Add error category breakdown columns to webhook_health_summaries
ALTER TABLE webhook_health_summaries
    ADD COLUMN client_errors INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN server_errors INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN timeout_errors INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN network_errors INTEGER NOT NULL DEFAULT 0;

COMMIT;
