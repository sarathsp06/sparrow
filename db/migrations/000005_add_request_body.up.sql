-- Add request_body column to webhook_deliveries table to store the actual JSON payload sent to webhook
ALTER TABLE webhook_deliveries 
ADD COLUMN request_body TEXT DEFAULT '';

-- Add index for potential querying on request_body content
CREATE INDEX idx_webhook_deliveries_request_body_gin ON webhook_deliveries USING gin(to_tsvector('english', request_body));