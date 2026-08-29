---
type: REST Resource
title: Events
description: Event type definitions, pushing, listing, re-push, and batch operations — 12 endpoints
tags: [rest, events]
timestamp: 2026-08-29T00:00:00Z
---

# Events

Registered under the `Event Types` and `Events` tags in the Huma-generated OpenAPI spec. Implemented in `internal/rest/event.go`.

## Endpoints

| Method | Path | OperationID | Description |
|--------|------|-------------|-------------|
| POST | `/v1/event-types` | `registerEventType` | Register an event type with optional JSON schema |
| GET | `/v1/event-types` | `listEventTypes` | List registered event types |
| GET | `/v1/event-types/{name}` | `getEventType` | Get a single event type |
| PATCH | `/v1/event-types/{name}` | `updateEventType` | Update event type definition |
| DELETE | `/v1/event-types/{name}` | `deleteEventType` | Delete an event type |
| POST | `/v1/namespaces/{namespace}/events` | `pushEvent` | Push a new event — triggers fan-out to subscriptions |
| GET | `/v1/namespaces/{namespace}/events` | `listEventOccurrences` | Filterable event occurrence listing (supports `prepare_repush`) |
| GET | `/v1/events/{event_id}` | `getEventOccurrence` | Get a single event occurrence |
| POST | `/v1/events/{event_id}:repush` | `repushEvent` | Re-push a single event through current subscriptions |
| POST | `/v1/namespaces/{namespace}/events:rePush` | `startEventRepushJob` | Batch re-push via snapshot — uses batch_jobs |
| GET | `/v1/namespaces/{namespace}/repush-jobs/{job_id}` | `getEventRepushJob` | Poll batch re-push progress |
| POST | `/v1/namespaces/{namespace}/repush-jobs/{job_id}:cancel` | `cancelEventRepushJob` | Cancel a batch re-push operation |

## Citations

- `internal/rest/event.go` — endpoint registration + handlers
