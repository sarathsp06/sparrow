-- Idempotent tokens (Service.CreateTokenIdempotent): the key a token was
-- created with, while it holds it, and its secret sealed by the application's
-- SecretSealer so it can be returned again. Both are cleared on revocation and
-- when an expired holder is replaced.
ALTER TABLE access_tokens ADD COLUMN idempotency_key TEXT;
ALTER TABLE access_tokens ADD COLUMN sealed_secret BYTEA;

-- One holder per key within a realm and scope (NULL scope = full access).
CREATE UNIQUE INDEX idx_access_tokens_idempotency
    ON access_tokens (realm, COALESCE(scope, ''), idempotency_key)
    WHERE idempotency_key IS NOT NULL;
