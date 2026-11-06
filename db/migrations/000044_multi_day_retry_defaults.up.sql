-- Retry for days by default, like hosted webhook senders: 13 retries with a
-- 60 s base spans about three days (see DefaultMaxRetries). Only the column
-- default changes; existing webhooks keep the max_retries they were saved with.
ALTER TABLE webhook_registrations ALTER COLUMN max_retries SET DEFAULT 13;
ALTER TABLE webhook_registrations DROP CONSTRAINT max_retries_check;
ALTER TABLE webhook_registrations ADD CONSTRAINT max_retries_check CHECK (max_retries >= 0 AND max_retries <= 20);
