-- Rows saved with more than 10 retries are clamped so the old constraint holds.
UPDATE webhook_registrations SET max_retries = 10 WHERE max_retries > 10;
ALTER TABLE webhook_registrations DROP CONSTRAINT max_retries_check;
ALTER TABLE webhook_registrations ADD CONSTRAINT max_retries_check CHECK (max_retries >= 0 AND max_retries <= 10);
ALTER TABLE webhook_registrations ALTER COLUMN max_retries SET DEFAULT 3;
