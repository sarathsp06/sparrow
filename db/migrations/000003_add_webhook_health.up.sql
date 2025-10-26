-- Add health column to webhook_registrations table
ALTER TABLE webhook_registrations 
ADD COLUMN health VARCHAR(20) DEFAULT 'unknown' NOT NULL;

-- Add check constraint for valid health values
ALTER TABLE webhook_registrations 
ADD CONSTRAINT webhook_health_check 
CHECK (health IN ('healthy', 'degraded', 'unhealthy', 'unknown'));

-- Create index for efficient health-based queries
CREATE INDEX idx_webhook_registrations_health ON webhook_registrations(health);

-- Create webhook_health_metrics table for tracking health statistics
CREATE TABLE webhook_health_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    total_deliveries INTEGER DEFAULT 0,
    successful_deliveries INTEGER DEFAULT 0,
    failed_deliveries INTEGER DEFAULT 0,
    consecutive_failures INTEGER DEFAULT 0,
    last_success_at TIMESTAMP WITH TIME ZONE,
    last_failure_at TIMESTAMP WITH TIME ZONE,
    success_rate DECIMAL(5,4) DEFAULT 0.0000, -- Percentage as decimal (0.9500 = 95%)
    avg_response_time INTEGER DEFAULT 0, -- Average response time in milliseconds
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for efficient querying
CREATE INDEX idx_webhook_health_metrics_webhook_id ON webhook_health_metrics(webhook_id);
CREATE INDEX idx_webhook_health_metrics_success_rate ON webhook_health_metrics(success_rate);
CREATE INDEX idx_webhook_health_metrics_last_success ON webhook_health_metrics(last_success_at);

-- Create function to automatically update webhook health based on metrics
CREATE OR REPLACE FUNCTION update_webhook_health()
RETURNS TRIGGER AS $$
BEGIN
    -- Update webhook health based on metrics
    UPDATE webhook_registrations 
    SET health = CASE
        WHEN NEW.consecutive_failures >= 5 THEN 'unhealthy'
        WHEN NEW.success_rate < 0.8000 AND NEW.total_deliveries >= 10 THEN 'degraded'
        WHEN NEW.success_rate >= 0.9000 AND NEW.total_deliveries >= 5 THEN 'healthy'
        WHEN NEW.total_deliveries = 0 THEN 'unknown'
        ELSE 'degraded'
    END,
    updated_at = NOW()
    WHERE id = NEW.webhook_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update health when metrics change
CREATE TRIGGER webhook_health_update_trigger
    AFTER INSERT OR UPDATE ON webhook_health_metrics
    FOR EACH ROW
    EXECUTE FUNCTION update_webhook_health();

-- Initialize health metrics for existing webhooks
INSERT INTO webhook_health_metrics (webhook_id)
SELECT id FROM webhook_registrations
ON CONFLICT DO NOTHING;