---
type: REST Resource
title: Events & Event Types
description: Define event types (optional JSON Schema) and push, list, and replay event occurrences
tags: [rest, events]
timestamp: 2026-09-16T00:00:00Z
---

# Events & Event Types

Two related concerns under the `Event Types` and `Events` tags. Event types are the registered catalog of event names, each with an optional JSON Schema used for soft payload validation. Event occurrences are pushed against a consumer (triggering fan-out to matching subscriptions), then listable, individually replayable, and batch re-pushable from a prepared snapshot via [batch jobs](/concepts/batch-jobs.md).

The authoritative endpoint list — paths, methods, and schemas — is the OpenAPI spec at `api/openapi.yaml` (browse at `/docs`).

## Citations

- `internal/rest/event.go` — endpoint registration + handlers
- `api/openapi.yaml` — canonical endpoint definitions
