-- Add error_category column to webhook_deliveries table
-- This stores the classified error category for each delivery attempt
-- (e.g., client_error, server_error, timeout, dns_error, tls_error, connection_refused, network_error)

ALTER TABLE webhook_deliveries
    ADD COLUMN IF NOT EXISTS error_category VARCHAR(30) NOT NULL DEFAULT '';

-- Backfill existing records based on response_code and error_message
UPDATE webhook_deliveries
SET error_category = CASE
    WHEN status = 'success' THEN 'success'
    WHEN response_code >= 400 AND response_code < 500 THEN 'client_error'
    WHEN response_code >= 500 THEN 'server_error'
    WHEN error_message ILIKE '%timeout%' OR error_message ILIKE '%deadline%' THEN 'timeout'
    WHEN error_message ILIKE '%dns%' OR error_message ILIKE '%no such host%' THEN 'dns_error'
    WHEN error_message ILIKE '%tls%' OR error_message ILIKE '%certificate%' THEN 'tls_error'
    WHEN error_message ILIKE '%connection refused%' THEN 'connection_refused'
    WHEN error_message ILIKE '%connection%' OR error_message ILIKE '%network%' THEN 'network_error'
    WHEN status IN ('failed', 'expired') THEN 'unknown'
    ELSE ''
END
WHERE error_category = '';

-- Index for filtering deliveries by error category
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_error_category
    ON webhook_deliveries (error_category)
    WHERE error_category != '' AND error_category != 'success';
