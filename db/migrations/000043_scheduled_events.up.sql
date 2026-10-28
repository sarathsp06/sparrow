-- Scheduled events: optional fan-out time on an event occurrence.
-- NULL means immediate (every pre-existing row). The delay itself lives in
-- the River job's scheduled_at; this column is for visibility and retention.
ALTER TABLE event_records ADD COLUMN scheduled_at TIMESTAMPTZ;
