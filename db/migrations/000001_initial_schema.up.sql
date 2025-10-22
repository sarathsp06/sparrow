-- Initial schema for sparrow webhook system
-- Create webhook_registrations table
CREATE TABLE webhook_registrations (
    id VARCHAR(255) PRIMARY KEY,
    namespace VARCHAR(255) NOT NULL,
    events JSONB NOT NULL,           -- Array of events this webhook listens to
    url TEXT NOT NULL,
    headers JSONB DEFAULT '{}',      -- Custom headers as JSON
    timeout INTEGER DEFAULT 30,     -- Timeout in seconds
    active BOOLEAN DEFAULT true,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for efficient querying
CREATE INDEX idx_webhook_registrations_namespace ON webhook_registrations(namespace);
CREATE INDEX idx_webhook_registrations_active ON webhook_registrations(active);
CREATE INDEX idx_webhook_registrations_events ON webhook_registrations USING GIN(events);

-- Create event_records table
CREATE TABLE event_records (
    id VARCHAR(255) PRIMARY KEY,
    namespace VARCHAR(255) NOT NULL,
    event VARCHAR(255) NOT NULL,
    payload TEXT NOT NULL,           -- JSON payload
    ttl BIGINT NOT NULL,            -- TTL in seconds
    metadata JSONB DEFAULT '{}',     -- Additional metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create indexes for event_records
CREATE INDEX idx_event_records_namespace ON event_records(namespace);
CREATE INDEX idx_event_records_event ON event_records(event);
CREATE INDEX idx_event_records_created_at ON event_records(created_at);
CREATE INDEX idx_event_records_expires_at ON event_records(expires_at);

-- Create webhook delivery status enum
CREATE TYPE webhook_delivery_status AS ENUM ('pending', 'sending', 'success', 'failed', 'retrying', 'expired');

-- Create webhook_deliveries table
CREATE TABLE webhook_deliveries (
    id VARCHAR(255) PRIMARY KEY,
    webhook_id VARCHAR(255) NOT NULL REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    event_id VARCHAR(255) NOT NULL REFERENCES event_records(id) ON DELETE CASCADE,
    status webhook_delivery_status DEFAULT 'pending',
    attempt_count INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_attempted_at TIMESTAMP WITH TIME ZONE,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    response_code INTEGER DEFAULT 0,
    response_body TEXT DEFAULT '',
    error_message TEXT DEFAULT ''
);

-- Create indexes for webhook_deliveries
CREATE INDEX idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX idx_webhook_deliveries_event_id ON webhook_deliveries(event_id);
CREATE INDEX idx_webhook_deliveries_status ON webhook_deliveries(status);
CREATE INDEX idx_webhook_deliveries_created_at ON webhook_deliveries(created_at);
CREATE INDEX idx_webhook_deliveries_expires_at ON webhook_deliveries(expires_at);

-- Create function to automatically update updated_at column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for auto-updating updated_at
CREATE TRIGGER update_webhook_registrations_updated_at 
    BEFORE UPDATE ON webhook_registrations 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();