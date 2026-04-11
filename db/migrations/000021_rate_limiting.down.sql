DROP TABLE IF EXISTS webhook_rate_limit_state;
ALTER TABLE webhook_registrations DROP COLUMN IF EXISTS rate_limit_rps;
