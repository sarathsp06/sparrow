-- 000015: Envelope encryption infrastructure
--
-- 1. system_settings table for auto-generated encryption key storage
-- 2. Convert webhook_secret from TEXT to BYTEA for encrypted storage

-- System settings table for persisting auto-generated KEK and other config.
-- When SPARROW_ENCRYPTION_KEY env var is not set, Sparrow auto-generates a
-- key on first boot and stores it here. Env var always takes precedence.
CREATE TABLE IF NOT EXISTS system_settings (
    key   TEXT PRIMARY KEY,
    value BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Convert webhook_secret from TEXT to BYTEA.
-- Existing plaintext values are preserved as their UTF-8 byte representation.
-- The application will detect and re-encrypt them with envelope encryption on startup.
ALTER TABLE webhook_registrations
    ALTER COLUMN webhook_secret TYPE BYTEA
    USING CASE
        WHEN webhook_secret = '' THEN NULL
        ELSE webhook_secret::bytea
    END;
