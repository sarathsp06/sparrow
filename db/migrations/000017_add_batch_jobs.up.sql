-- Batch jobs table for deterministic bulk operations (re-push events, retry deliveries).
-- Each row snapshots a set of IDs at query time so that the bulk action operates on
-- exactly what the user saw, not a live re-query.
CREATE TABLE batch_jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    namespace     VARCHAR(255) NOT NULL,
    job_type      VARCHAR(50) NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending',
    data          JSONB NOT NULL,
    total         INTEGER NOT NULL DEFAULT 0,
    processed     INTEGER NOT NULL DEFAULT 0,
    failed        INTEGER NOT NULL DEFAULT 0,
    ttl_seconds   INTEGER NOT NULL DEFAULT 900,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_batch_jobs_tenant_status ON batch_jobs (tenant_id, status);
CREATE INDEX idx_batch_jobs_expires_at ON batch_jobs (expires_at) WHERE status NOT IN ('completed', 'cancelled');
