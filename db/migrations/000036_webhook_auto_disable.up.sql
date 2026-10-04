-- Automatic disabling of webhooks whose receiver keeps failing.
--
-- failing_since is when the current run of failed attempts started: set by
-- the first failure after a success, cleared by the next success, and cleared
-- when an operator resumes an auto-disabled webhook so it gets a fresh window.
-- Existing failure runs start counting from their most recent failure, which
-- can only delay (never hasten) a disable after this migration.
ALTER TABLE webhook_health_state
    ADD COLUMN failing_since TIMESTAMPTZ;

UPDATE webhook_health_state
SET failing_since = last_failure_at
WHERE consecutive_failures > 0;

-- Set when Sparrow paused the webhook itself (active = false) because it kept
-- failing; cleared when the webhook is resumed. A manual pause leaves these
-- NULL, so the UI and API can tell the two apart.
ALTER TABLE webhook_registrations
    ADD COLUMN auto_disabled_at TIMESTAMPTZ,
    ADD COLUMN auto_disabled_reason TEXT;
