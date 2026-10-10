CREATE TABLE IF NOT EXISTS webhook_health_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    total_deliveries INTEGER NOT NULL DEFAULT 0,
    successful_deliveries INTEGER NOT NULL DEFAULT 0,
    failed_deliveries INTEGER NOT NULL DEFAULT 0,
    success_rate NUMERIC(5,4) NOT NULL DEFAULT 0.0000,
    avg_response_time INTEGER NOT NULL DEFAULT 0,
    min_response_time INTEGER NOT NULL DEFAULT 0,
    max_response_time INTEGER NOT NULL DEFAULT 0,
    p95_response_time INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    client_errors INTEGER NOT NULL DEFAULT 0,
    server_errors INTEGER NOT NULL DEFAULT 0,
    timeout_errors INTEGER NOT NULL DEFAULT 0,
    network_errors INTEGER NOT NULL DEFAULT 0,
    unexpected_status_errors INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_webhook_health_summaries_unique ON webhook_health_summaries (webhook_id, window_start, window_end);
CREATE INDEX IF NOT EXISTS idx_webhook_health_summaries_webhook_id ON webhook_health_summaries (webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_health_summaries_window ON webhook_health_summaries (window_start, window_end);
