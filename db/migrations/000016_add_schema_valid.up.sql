-- Add schema_valid column to event_records for soft schema validation.
-- Events are always accepted; schema_valid=false indicates the payload did not match the registered schema.
ALTER TABLE event_records ADD COLUMN schema_valid BOOLEAN NOT NULL DEFAULT true;
