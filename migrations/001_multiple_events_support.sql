-- Migration to update webhook_registrations table for multiple events support
-- and enforce POST-only method

-- First, let's see the current structure and update it
-- This assumes you're starting fresh, but if you have existing data,
-- you might need to handle data migration

-- Drop the existing table if it exists (use with caution in production)
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS event_records;
DROP TABLE IF EXISTS webhook_registrations;

-- Create webhook_registrations table with new structure
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

-- Create webhook_deliveries table
CREATE TYPE webhook_delivery_status AS ENUM ('pending', 'sending', 'success', 'failed', 'retrying', 'expired');

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

-- Insert some sample data for testing
INSERT INTO webhook_registrations (id, namespace, events, url, headers, timeout, active, description) VALUES
('webhook-1', 'user', '["signup", "login", "logout"]', 'https://httpbin.org/post', '{"Authorization": "Bearer token1"}', 30, true, 'User activity webhook'),
('webhook-2', 'order', '["created", "updated", "cancelled"]', 'https://httpbin.org/post', '{"X-Service": "OrderProcessor"}', 30, true, 'Order lifecycle webhook'),
('webhook-3', 'payment', '["processed", "failed", "refunded"]', 'https://httpbin.org/post', '{"X-Secret": "payment-secret"}', 15, true, 'Payment processing webhook');

COMMIT;