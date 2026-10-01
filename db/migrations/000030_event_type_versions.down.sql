ALTER TABLE event_records DROP COLUMN IF EXISTS event_version;
DROP TABLE IF EXISTS event_registration_versions;
ALTER TABLE event_registrations DROP COLUMN IF EXISTS version;
