BEGIN;

-- Consolidated schema for sparrow webhook system
-- This migration creates all tables with their final structure to avoid ALTER TABLE statements

-- Create webhook delivery status enum
CREATE TYPE webhook_delivery_status AS ENUM ('pending', 'sending', 'success', 'failed', 'retrying', 'expired');

-- Create event_registrations table for event type registry
CREATE TABLE event_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    schema TEXT, -- JSON schema for payload validation
    metadata JSONB DEFAULT '{}'::jsonb,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for event_registrations
CREATE INDEX idx_event_registrations_name ON event_registrations(name, active);
CREATE INDEX idx_event_registrations_created_at ON event_registrations(created_at);

-- Add comment
COMMENT ON TABLE event_registrations IS 'Registry of available event types that can trigger webhooks';

-- Create webhook_registrations table with all columns including health and HTTP config
CREATE TABLE webhook_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace VARCHAR(255) NOT NULL,
    events TEXT[],           -- Array of events this webhook listens to
    url TEXT NOT NULL,
    headers JSONB DEFAULT '{}'::JSONB,      -- Custom headers as JSON
    timeout INTEGER DEFAULT 30,     -- Timeout in seconds
    active BOOLEAN DEFAULT true,
    description TEXT,
    health VARCHAR(20) DEFAULT 'unknown' NOT NULL,
    
    -- HTTP Configuration
    max_retries INTEGER DEFAULT 3,
    retry_backoff_seconds INTEGER DEFAULT 60,     -- Base backoff time between retries
    capture_response_body BOOLEAN DEFAULT false,  -- Whether to capture and store response body
    follow_redirects BOOLEAN DEFAULT true,        -- Whether to follow HTTP redirects
    verify_ssl BOOLEAN DEFAULT true,              -- Whether to verify SSL certificates
    request_timeout_seconds INTEGER DEFAULT 30,   -- Per-request timeout
    expected_status_codes INTEGER[] DEFAULT '{200,201,202,204}'::INTEGER[], -- Expected success status codes
    webhook_secret TEXT,                          -- Secret for webhook signature verification
    user_agent TEXT DEFAULT 'Sparrow-Webhook/1.0', -- Custom User-Agent header
    content_type TEXT DEFAULT 'application/json', -- Content-Type for requests
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_namespace_url UNIQUE (namespace, url),
    CONSTRAINT webhook_health_check CHECK (health IN ('healthy', 'degraded', 'unhealthy', 'unknown')),
    CONSTRAINT max_retries_check CHECK (max_retries >= 0 AND max_retries <= 10),
    CONSTRAINT retry_backoff_check CHECK (retry_backoff_seconds > 0 AND retry_backoff_seconds <= 3600),
    CONSTRAINT request_timeout_check CHECK (request_timeout_seconds > 0 AND request_timeout_seconds <= 300)
);

-- Create indexes for webhook_registrations
CREATE INDEX idx_webhook_registrations_namespace ON webhook_registrations(namespace);
CREATE INDEX idx_webhook_registrations_active ON webhook_registrations(active);
CREATE INDEX idx_webhook_registrations_events ON webhook_registrations USING GIN(events);
CREATE INDEX idx_webhook_registrations_health ON webhook_registrations(health);

-- Create event_records table
CREATE TABLE event_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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

-- Create webhook_deliveries table with all columns including request_body
CREATE TABLE webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES event_records(id) ON DELETE CASCADE,
    status webhook_delivery_status DEFAULT 'pending',
    attempt_count INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    request_body TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
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
CREATE INDEX idx_webhook_deliveries_request_body_gin ON webhook_deliveries USING gin(to_tsvector('english', request_body));

-- Create webhook health tracking tables
CREATE TABLE webhook_health_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    delivery_id UUID NOT NULL,
    success BOOLEAN NOT NULL,
    response_time INTEGER NOT NULL DEFAULT 0, -- milliseconds
    response_code INTEGER NOT NULL DEFAULT 0,
    error_message TEXT DEFAULT '',
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for webhook_health_events
CREATE INDEX idx_webhook_health_events_webhook_id_timestamp ON webhook_health_events(webhook_id, timestamp DESC);
CREATE INDEX idx_webhook_health_events_timestamp ON webhook_health_events(timestamp DESC);
CREATE INDEX idx_webhook_health_events_success ON webhook_health_events(success);
CREATE INDEX idx_webhook_health_events_delivery_id ON webhook_health_events(delivery_id);

-- Create webhook health summaries table
CREATE TABLE webhook_health_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    window_end TIMESTAMP WITH TIME ZONE NOT NULL,
    total_deliveries INTEGER NOT NULL DEFAULT 0,
    successful_deliveries INTEGER NOT NULL DEFAULT 0,
    failed_deliveries INTEGER NOT NULL DEFAULT 0,
    success_rate DECIMAL(5,4) NOT NULL DEFAULT 0.0000,
    avg_response_time INTEGER NOT NULL DEFAULT 0, -- milliseconds
    min_response_time INTEGER NOT NULL DEFAULT 0,
    max_response_time INTEGER NOT NULL DEFAULT 0,
    p95_response_time INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for webhook_health_summaries
CREATE UNIQUE INDEX idx_webhook_health_summaries_unique ON webhook_health_summaries(webhook_id, window_start, window_end);
CREATE INDEX idx_webhook_health_summaries_webhook_id ON webhook_health_summaries(webhook_id);
CREATE INDEX idx_webhook_health_summaries_window ON webhook_health_summaries(window_start, window_end);

-- Create webhook health state table
CREATE TABLE webhook_health_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL UNIQUE REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    last_success_at TIMESTAMP WITH TIME ZONE,
    last_failure_at TIMESTAMP WITH TIME ZONE,
    last_event_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for webhook_health_state
CREATE INDEX idx_webhook_health_state_webhook_id ON webhook_health_state(webhook_id);
CREATE INDEX idx_webhook_health_state_last_event ON webhook_health_state(last_event_at DESC);

-- Create function to automatically update updated_at column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for auto-updating updated_at on webhook_registrations
CREATE TRIGGER update_webhook_registrations_updated_at 
    BEFORE UPDATE ON webhook_registrations 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create trigger for auto-updating updated_at on event_registrations
CREATE TRIGGER update_event_registrations_updated_at 
    BEFORE UPDATE ON event_registrations 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create notification function for webhook health events
CREATE OR REPLACE FUNCTION notify_webhook_health_event()
RETURNS TRIGGER AS $$
BEGIN
    -- Send PostgreSQL notification for webhook health events
    -- This allows Go applications to listen for real-time health updates
    PERFORM pg_notify(
        'webhook_health_event',
        json_build_object(
            'webhook_id', NEW.webhook_id,
            'success', NEW.success,
            'response_code', NEW.response_code,
            'response_time', NEW.response_time,
            'timestamp', NEW.timestamp
        )::text
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create notification trigger for webhook health events
CREATE TRIGGER webhook_health_event_notification_trigger
    AFTER INSERT ON webhook_health_events
    FOR EACH ROW
    EXECUTE FUNCTION notify_webhook_health_event();
COMMIT;