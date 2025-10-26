-- High-performance time-series health tracking for webhooks
-- This replaces the previous health system with a more scalable approach

-- Drop old health system first
DROP TRIGGER IF EXISTS webhook_health_update_trigger ON webhook_health_metrics;
DROP FUNCTION IF EXISTS update_webhook_health();
DROP TABLE IF EXISTS webhook_health_metrics;

-- Create time-series table for individual health events
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

-- Create hypertable for time-series performance (if TimescaleDB is available)
-- This will be ignored if TimescaleDB extension is not installed
DO $$
BEGIN
    -- Try to create hypertable, ignore if TimescaleDB is not available
    PERFORM create_hypertable('webhook_health_events', 'timestamp', if_not_exists => TRUE);
EXCEPTION WHEN OTHERS THEN
    -- Log that regular table is being used instead of hypertable
    RAISE NOTICE 'TimescaleDB not available, using regular table for webhook_health_events';
END $$;

-- Create indexes for time-series queries
CREATE INDEX idx_webhook_health_events_webhook_id_timestamp ON webhook_health_events(webhook_id, timestamp DESC);
CREATE INDEX idx_webhook_health_events_timestamp ON webhook_health_events(timestamp DESC);
CREATE INDEX idx_webhook_health_events_success ON webhook_health_events(success);
CREATE INDEX idx_webhook_health_events_delivery_id ON webhook_health_events(delivery_id);

-- Create aggregated health summaries table (pre-computed for performance)
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

-- Create indexes for summary queries
CREATE UNIQUE INDEX idx_webhook_health_summaries_unique ON webhook_health_summaries(webhook_id, window_start, window_end);
CREATE INDEX idx_webhook_health_summaries_webhook_id ON webhook_health_summaries(webhook_id);
CREATE INDEX idx_webhook_health_summaries_window ON webhook_health_summaries(window_start, window_end);

-- Create current health state table (lightweight, frequently updated)
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

-- Create index for state queries
CREATE INDEX idx_webhook_health_state_webhook_id ON webhook_health_state(webhook_id);
CREATE INDEX idx_webhook_health_state_last_event ON webhook_health_state(last_event_at DESC);

-- Function to calculate health status based on recent events
CREATE OR REPLACE FUNCTION calculate_webhook_health(p_webhook_id UUID, p_lookback_hours INTEGER DEFAULT 24)
RETURNS TEXT AS $$
DECLARE
    recent_events_count INTEGER;
    recent_success_rate DECIMAL;
    consecutive_failures_count INTEGER;
BEGIN
    -- Get recent event statistics
    SELECT 
        COUNT(*),
        COALESCE(AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END), 0)
    INTO recent_events_count, recent_success_rate
    FROM webhook_health_events 
    WHERE webhook_id = p_webhook_id 
      AND timestamp >= NOW() - INTERVAL '1 hour' * p_lookback_hours;
    
    -- Get consecutive failures
    SELECT COALESCE(consecutive_failures, 0)
    INTO consecutive_failures_count
    FROM webhook_health_state
    WHERE webhook_id = p_webhook_id;
    
    -- Calculate health status
    IF recent_events_count = 0 THEN
        RETURN 'unknown';
    ELSIF consecutive_failures_count >= 5 THEN
        RETURN 'unhealthy';
    ELSIF recent_success_rate < 0.8000 AND recent_events_count >= 10 THEN
        RETURN 'unhealthy';
    ELSIF recent_success_rate < 0.9000 AND recent_events_count >= 5 THEN
        RETURN 'degraded';
    ELSIF recent_success_rate >= 0.9000 AND recent_events_count >= 3 THEN
        RETURN 'healthy';
    ELSE
        RETURN 'unknown';
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to update webhook health state after each delivery
CREATE OR REPLACE FUNCTION update_webhook_health_state()
RETURNS TRIGGER AS $$
BEGIN
    -- Update or insert health state
    INSERT INTO webhook_health_state (webhook_id, consecutive_failures, last_success_at, last_failure_at, last_event_at, updated_at)
    VALUES (
        NEW.webhook_id,
        CASE WHEN NEW.success THEN 0 ELSE 1 END,
        CASE WHEN NEW.success THEN NEW.timestamp ELSE NULL END,
        CASE WHEN NOT NEW.success THEN NEW.timestamp ELSE NULL END,
        NEW.timestamp,
        NOW()
    )
    ON CONFLICT (webhook_id) DO UPDATE SET
        consecutive_failures = CASE 
            WHEN NEW.success THEN 0 
            ELSE webhook_health_state.consecutive_failures + 1 
        END,
        last_success_at = CASE 
            WHEN NEW.success THEN NEW.timestamp 
            ELSE webhook_health_state.last_success_at 
        END,
        last_failure_at = CASE 
            WHEN NOT NEW.success THEN NEW.timestamp 
            ELSE webhook_health_state.last_failure_at 
        END,
        last_event_at = NEW.timestamp,
        updated_at = NOW();
    
    -- Update webhook health status
    UPDATE webhook_registrations 
    SET health = calculate_webhook_health(NEW.webhook_id),
        updated_at = NOW()
    WHERE id = NEW.webhook_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for health state updates
CREATE TRIGGER webhook_health_state_update_trigger
    AFTER INSERT ON webhook_health_events
    FOR EACH ROW
    EXECUTE FUNCTION update_webhook_health_state();

-- Function to aggregate health events into hourly summaries
CREATE OR REPLACE FUNCTION aggregate_webhook_health_hourly()
RETURNS INTEGER AS $$
DECLARE
    processed_count INTEGER := 0;
    webhook_record RECORD;
    hour_record RECORD;
BEGIN
    -- Process each webhook
    FOR webhook_record IN 
        SELECT DISTINCT webhook_id FROM webhook_health_events 
        WHERE timestamp >= NOW() - INTERVAL '2 hours'
    LOOP
        -- Process each hour for the webhook
        FOR hour_record IN
            SELECT 
                date_trunc('hour', timestamp) as hour_start,
                date_trunc('hour', timestamp) + INTERVAL '1 hour' as hour_end
            FROM webhook_health_events 
            WHERE webhook_id = webhook_record.webhook_id
              AND timestamp >= NOW() - INTERVAL '2 hours'
            GROUP BY date_trunc('hour', timestamp)
        LOOP
            -- Insert or update hourly summary
            INSERT INTO webhook_health_summaries (
                webhook_id, window_start, window_end,
                total_deliveries, successful_deliveries, failed_deliveries,
                success_rate, avg_response_time, min_response_time, 
                max_response_time, p95_response_time, updated_at
            )
            SELECT 
                webhook_record.webhook_id,
                hour_record.hour_start,
                hour_record.hour_end,
                COUNT(*),
                SUM(CASE WHEN success THEN 1 ELSE 0 END),
                SUM(CASE WHEN success THEN 0 ELSE 1 END),
                AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END),
                AVG(response_time)::INTEGER,
                MIN(response_time),
                MAX(response_time),
                PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY response_time)::INTEGER,
                NOW()
            FROM webhook_health_events
            WHERE webhook_id = webhook_record.webhook_id
              AND timestamp >= hour_record.hour_start
              AND timestamp < hour_record.hour_end
            ON CONFLICT (webhook_id, window_start, window_end) DO UPDATE SET
                total_deliveries = EXCLUDED.total_deliveries,
                successful_deliveries = EXCLUDED.successful_deliveries,
                failed_deliveries = EXCLUDED.failed_deliveries,
                success_rate = EXCLUDED.success_rate,
                avg_response_time = EXCLUDED.avg_response_time,
                min_response_time = EXCLUDED.min_response_time,
                max_response_time = EXCLUDED.max_response_time,
                p95_response_time = EXCLUDED.p95_response_time,
                updated_at = NOW();
            
            processed_count := processed_count + 1;
        END LOOP;
    END LOOP;
    
    RETURN processed_count;
END;
$$ LANGUAGE plpgsql;

-- Initialize health state for existing webhooks
INSERT INTO webhook_health_state (webhook_id)
SELECT id FROM webhook_registrations
ON CONFLICT (webhook_id) DO NOTHING;