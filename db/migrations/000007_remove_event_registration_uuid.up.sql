-- Remove the UUID primary key from event_registrations.
-- The natural key (tenant_id, name) is already unique and used for every query.
-- No other table has a FK referencing event_registrations.id.

-- Drop the old UUID primary key
ALTER TABLE event_registrations DROP CONSTRAINT event_registrations_pkey;

-- Drop the id column
ALTER TABLE event_registrations DROP COLUMN id;

-- Make (tenant_id, name) the composite primary key.
-- The unique index idx_event_registrations_tenant_name already exists (from migration 4),
-- so drop it first to avoid a duplicate constraint, then create the PK.
DROP INDEX IF EXISTS idx_event_registrations_tenant_name;
ALTER TABLE event_registrations ADD PRIMARY KEY (tenant_id, name);
