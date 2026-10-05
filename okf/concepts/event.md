---
type: Concept
title: Event
description: Versioned event types with optional JSON schema validation, and event record instances pinned to a version
tags: [event, schema, validation, versioning, import, export]
timestamp: 2026-06-22T00:00:00Z
---

# Event

Sparrow has two event-related entities:

## Event Registration

Defines an event type with an optional JSON schema. Schema validation is **soft** — mismatches produce warnings (`schema_valid=false`), events are always stored.

- Composite PK: `(tenant_id, name)`
- Fields: name, description, schema, metadata, active, `version`
- Versioned: every schema is kept in `event_registration_versions` (PK
  `(tenant_id, name, version)`, FK to the head row with `ON DELETE RESTRICT`).
  Only a schema change bumps the version; adding a first schema fills in the
  current version (`schema_defined_at`). Other fields change in place.
- Never deleted: there is no delete API; retire with `active=false`.
- One save path (`internal/webhooks/event_type_save.go`) handles register,
  PATCH, import and auto-register, under a row lock.
- A breaking schema change (subscriber-side rules in `schema_compat.go`) to a
  type any subscription receives needs `allow_breaking`.
- `sparrow.*` names are reserved for system events.
- Unknown names on push return 404 unless `SPARROW_AUTO_REGISTER_EVENTS=true`.
- Schema inference (UI only, no API): the schema editor generates a schema
  from stored events of the type (`GET /v1/events?event=`), merging several
  into required/optional, unioned types and strict `format` detection
  (`web/src/lib/schema-infer.ts`, `SchemaFromSamples.svelte`). The suggested
  dev workflow: auto-register, push real events, infer, review, export.
- Bundles: `POST /v1/event-types:export` / `:import` move definitions between
  environments (`internal/webhooks/event_type_bundle.go`), all-or-nothing, with
  dry run, a version stamp, and a strict template pre-check.

## Event Record

An instance of a pushed event. Created by the `PushEvent` API.

- Deduplication via optional `idempotency_key` (partial unique index)
- `schema_valid` flag for soft validation
- `event_version`: the event type version it was accepted under (NULL reads as 1)
- TTL-based expiry

## Citations

- `db/migrations/000016.up.sql` — schema_valid
- `db/migrations/000020.up.sql` — idempotency_key
- `db/migrations/000030_event_type_versions.up.sql` — versions, event_version
