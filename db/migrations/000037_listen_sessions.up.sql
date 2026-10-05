-- Listen sessions: `sparrow listen` against a Sparrow that cannot reach the
-- developer's machine. A session is an ordinary webhook whose URL is
-- sparrow-cli://<webhook id>; instead of POSTing, the delivery worker hands
-- the signed request to listen_deliveries and waits for the CLI, which
-- long-polls for it, forwards it locally and reports the response.
--
-- The session row carries what the webhook row has no column for: when the
-- session ends on its own and when the CLI was last heard from (a delivery to
-- a CLI that stopped polling fails fast instead of waiting).
CREATE TABLE listen_sessions (
    webhook_id   UUID PRIMARY KEY REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    tenant_id    UUID NOT NULL,
    consumer     VARCHAR(255) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_listen_sessions_consumer ON listen_sessions (tenant_id, consumer);
CREATE INDEX idx_listen_sessions_expires_at ON listen_sessions (expires_at);

-- One row per in-flight attempt, keyed by delivery: a retry replaces the row.
-- The worker deletes it once the attempt is over (answered or timed out).
CREATE TABLE listen_deliveries (
    delivery_id      UUID PRIMARY KEY REFERENCES webhook_deliveries(id) ON DELETE CASCADE,
    webhook_id       UUID NOT NULL REFERENCES listen_sessions(webhook_id) ON DELETE CASCADE,
    method           TEXT NOT NULL,
    headers          JSONB NOT NULL DEFAULT '{}'::JSONB,
    body             BYTEA NOT NULL,
    queued_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    claimed_at       TIMESTAMPTZ,
    response_status  INTEGER,
    response_headers JSONB,
    response_body    BYTEA,
    responded_at     TIMESTAMPTZ
);

CREATE INDEX idx_listen_deliveries_unclaimed ON listen_deliveries (webhook_id, queued_at) WHERE claimed_at IS NULL;
