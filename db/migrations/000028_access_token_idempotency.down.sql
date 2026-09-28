DROP INDEX IF EXISTS idx_access_tokens_idempotency;
ALTER TABLE access_tokens DROP COLUMN IF EXISTS sealed_secret;
ALTER TABLE access_tokens DROP COLUMN IF EXISTS idempotency_key;
