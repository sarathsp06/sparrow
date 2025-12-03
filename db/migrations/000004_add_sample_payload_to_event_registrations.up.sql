BEGIN;

-- Add sample_payload column to event_registrations table
ALTER TABLE event_registrations
ADD COLUMN sample_payload JSONB DEFAULT '{}'::jsonb;

-- Add comment
COMMENT ON COLUMN event_registrations.sample_payload IS 'Auto-generated sample payload based on the event schema';

COMMIT;
