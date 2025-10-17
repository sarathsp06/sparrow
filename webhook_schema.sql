-- Create webhook_registrations table
CREATE TABLE IF NOT EXISTS webhook_registrations (
    id VARCHAR(36) PRIMARY KEY,
    namespace VARCHAR(255) NOT NULL,
    event VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    method VARCHAR(10) NOT NULL DEFAULT 'POST',
    headers JSONB DEFAULT '{}',
    timeout INTEGER NOT NULL DEFAULT 30,
    active BOOLEAN NOT NULL DEFAULT true,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_webhook_registrations_namespace_event 
    ON webhook_registrations(namespace, event) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_webhook_registrations_namespace 
    ON webhook_registrations(namespace) WHERE active = true;

-- Create event_records table
CREATE TABLE IF NOT EXISTS event_records (
    id VARCHAR(36) PRIMARY KEY,
    namespace VARCHAR(255) NOT NULL,
    event VARCHAR(255) NOT NULL,
    payload TEXT NOT NULL,
    ttl BIGINT NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create indexes for event records
CREATE INDEX IF NOT EXISTS idx_event_records_namespace_event 
    ON event_records(namespace, event);
CREATE INDEX IF NOT EXISTS idx_event_records_expires_at 
    ON event_records(expires_at);

-- Create webhook_deliveries table
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id VARCHAR(36) PRIMARY KEY,
    webhook_id VARCHAR(36) NOT NULL,
    event_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_attempted_at TIMESTAMP WITH TIME ZONE,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    response_code INTEGER DEFAULT 0,
    response_body TEXT DEFAULT '',
    error_message TEXT DEFAULT '',
    FOREIGN KEY (webhook_id) REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES event_records(id) ON DELETE CASCADE
);

-- Create indexes for webhook deliveries
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id 
    ON webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_event_id 
    ON webhook_deliveries(event_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status 
    ON webhook_deliveries(status);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_expires_at 
    ON webhook_deliveries(expires_at);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_next_retry 
    ON webhook_deliveries(next_retry_at) WHERE next_retry_at IS NOT NULL;

-- Add a trigger to automatically update updated_at for webhook_registrations
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_webhook_registrations_updated_at BEFORE UPDATE
    ON webhook_registrations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();