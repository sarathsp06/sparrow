-- Restore the UUID primary key on event_registrations.

-- Drop the composite primary key
ALTER TABLE event_registrations DROP CONSTRAINT event_registrations_pkey;

-- Re-add the id column with UUIDs
ALTER TABLE event_registrations ADD COLUMN id UUID DEFAULT gen_random_uuid();

-- Backfill any rows that might have NULL ids
UPDATE event_registrations SET id = gen_random_uuid() WHERE id IS NULL;

-- Make id NOT NULL and set as primary key
ALTER TABLE event_registrations ALTER COLUMN id SET NOT NULL;
ALTER TABLE event_registrations ADD PRIMARY KEY (id);

-- Re-create the unique index on (tenant_id, name)
CREATE UNIQUE INDEX idx_event_registrations_tenant_name ON event_registrations(tenant_id, name);
