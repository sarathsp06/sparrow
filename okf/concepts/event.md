---
type: Concept
title: Event
description: Event types with optional JSON schema validation and event record instances
tags: [event, schema, validation]
timestamp: 2026-06-22T00:00:00Z
---

# Event

Sparrow has two event-related entities:

## Event Registration

Defines an event type with an optional JSON schema. Schema validation is **soft** — mismatches produce warnings (`schema_valid=false`), events are always stored.

- Composite PK: `(tenant_id, name)`
- Fields: name, description, schema JSONB, active

## Event Record

An instance of a pushed event. Created by `PushEvent` RPC.

- Deduplication via optional `idempotency_key` (partial unique index)
- `schema_valid` flag for soft validation
- TTL-based expiry

## Citations

- `db/migrations/000016.up.sql` — schema_valid
- `db/migrations/000020.up.sql` — idempotency_key
