-- Access tokens: rename idempotency_key to external_id (pkg/access). The block
-- below must stay identical to pkg/access/pgstore/schema_003_external_id.sql
-- (checked by internal/accessauth tests).

-- The per-token key of Service.GetOrCreateToken is the caller's external id
-- for what the token is for (e.g. a user id), not a per-request idempotency key.
ALTER TABLE access_tokens RENAME COLUMN idempotency_key TO external_id;
ALTER INDEX idx_access_tokens_idempotency RENAME TO idx_access_tokens_external_id;
