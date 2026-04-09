DROP INDEX IF EXISTS idx_event_records_idempotency_key;
ALTER TABLE event_records DROP COLUMN IF EXISTS idempotency_key;
