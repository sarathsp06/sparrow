-- Versioned event type definitions.
--
-- event_registrations keeps exactly one row per (tenant_id, name): the current
-- definition. `version` says which version that row is. Every version ever
-- created, including the current one, is kept in event_registration_versions,
-- which is append-only apart from the one-time "fill in" of a schema-less
-- version (see schema_defined_at).

ALTER TABLE event_registrations ADD COLUMN version INT NOT NULL DEFAULT 1;

CREATE TABLE event_registration_versions (
    tenant_id         UUID         NOT NULL,
    name              VARCHAR(255) NOT NULL,
    version           INT          NOT NULL,
    description       TEXT,
    schema            TEXT,
    sample_payload    JSONB,
    schema_defined_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, name, version),
    -- Event types are never deleted: a definition with history cannot be removed.
    FOREIGN KEY (tenant_id, name)
        REFERENCES event_registrations (tenant_id, name) ON DELETE RESTRICT
);

COMMENT ON TABLE event_registration_versions IS 'Every version of every event type definition, including the current one';
COMMENT ON COLUMN event_registration_versions.schema_defined_at IS 'Set when a schema was added to a version that had none, which updates that version in place';

INSERT INTO event_registration_versions
    (tenant_id, name, version, description, schema, sample_payload, created_at)
SELECT tenant_id, name, 1, description, schema, sample_payload, COALESCE(created_at, NOW())
FROM event_registrations;

-- The version of the event type an occurrence was accepted under.
-- NULL means version 1: no backfill, so the large table is not rewritten.
ALTER TABLE event_records ADD COLUMN event_version INT;
