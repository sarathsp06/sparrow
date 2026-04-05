-- Revert envelope encryption infrastructure

-- Convert webhook_secret back from BYTEA to TEXT.
-- This assumes stored values are either NULL or valid UTF-8 bytes.
ALTER TABLE webhook_registrations
    ALTER COLUMN webhook_secret TYPE TEXT
    USING COALESCE(encode(webhook_secret, 'escape'), '');

-- Drop system settings table
DROP TABLE IF EXISTS system_settings;
