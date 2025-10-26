ALTER TABLE webhook_registrations 
ADD COLUMN health VARCHAR(20) DEFAULT 'unknown' NOT NULL;

-- Add check constraint for valid health values
ALTER TABLE webhook_registrations 
ADD CONSTRAINT webhook_health_check 
CHECK (health IN ('healthy', 'degraded', 'unhealthy', 'unknown'));

-- Create index for efficient health-based queries
CREATE INDEX idx_webhook_registrations_health ON webhook_registrations(health);