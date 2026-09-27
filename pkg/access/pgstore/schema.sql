-- Schema for github.com/sarathsp06/sparrow/pkg/access/pgstore.
-- Only SHA-256 hashes of secrets are stored.
CREATE TABLE access_tokens (
    id           TEXT PRIMARY KEY,
    realm        TEXT NOT NULL,
    scope        TEXT,
    name         TEXT NOT NULL,
    secret_hash  BYTEA NOT NULL UNIQUE,
    created_by   TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL,
    expires_at   TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ
);

CREATE INDEX idx_access_tokens_realm ON access_tokens (realm, created_at DESC);

CREATE TABLE access_invites (
    id                TEXT PRIMARY KEY,
    realm             TEXT NOT NULL,
    scope             TEXT,
    name              TEXT NOT NULL,
    secret_hash       BYTEA NOT NULL UNIQUE,
    token_ttl_seconds BIGINT,
    created_by        TEXT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL,
    expires_at        TIMESTAMPTZ NOT NULL,
    redeemed_at       TIMESTAMPTZ,
    cancelled_at      TIMESTAMPTZ,
    token_id          TEXT REFERENCES access_tokens (id) ON DELETE SET NULL
);

CREATE INDEX idx_access_invites_realm ON access_invites (realm, created_at DESC);
