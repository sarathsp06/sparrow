ALTER INDEX IF EXISTS idx_access_tokens_external_id RENAME TO idx_access_tokens_idempotency;
ALTER TABLE access_tokens RENAME COLUMN external_id TO idempotency_key;
