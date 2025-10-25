-- Create event_registrations table
CREATE TABLE IF NOT EXISTS event_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    schema TEXT, -- JSON schema for payload validation
    metadata JSONB DEFAULT '{}'::jsonb,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_event_registrations_name ON event_registrations(name);
CREATE INDEX IF NOT EXISTS idx_event_registrations_active ON event_registrations(active);
CREATE INDEX IF NOT EXISTS idx_event_registrations_created_at ON event_registrations(created_at);

-- Add comment
COMMENT ON TABLE event_registrations IS 'Registry of available event types that can trigger webhooks';