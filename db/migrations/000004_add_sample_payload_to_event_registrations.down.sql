BEGIN;

-- Remove sample_payload column from event_registrations table
ALTER TABLE event_registrations
DROP COLUMN IF EXISTS sample_payload;

COMMIT;
