ALTER TABLE webhook_registrations
    DROP COLUMN IF EXISTS auto_disabled_reason,
    DROP COLUMN IF EXISTS auto_disabled_at;

ALTER TABLE webhook_health_state
    DROP COLUMN IF EXISTS failing_since;
