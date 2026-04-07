-- Recreate system_settings table (reverts 000019).
-- This only recreates the schema; it does NOT restore any previously stored
-- encryption key. After rolling back, set SPARROW_ENCRYPTION_KEY env var or
-- let the application auto-generate a new key on next startup.
CREATE TABLE IF NOT EXISTS system_settings (
    key   TEXT PRIMARY KEY,
    value BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
