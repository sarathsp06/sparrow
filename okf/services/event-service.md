---
type: gRPC Service
title: EventService
description: Event registration, pushing, listing, re-push, and batch operations — 12 RPCs
tags: [grpc, events]
timestamp: 2026-06-22T00:00:00Z
---

# EventService

Defined in [`webhook.proto`](/references/proto.md). The largest service with 12 RPCs.

## RPCs

| RPC | Description |
|-----|-------------|
| `RegisterEvent` | Register an event type with optional JSON schema |
| `ListEvents` | List registered event types |
| `UpdateEvent` | Update event type definition |
| `DeleteEvent` | Delete an event type |
| `GetEvent` | Get a single event type |
| `PushEvent` | Push a new event — triggers fan-out to subscriptions |
| `ListEventReports` | Filterable event delivery reports (supports `prepare_repush`) |
| `GetEventRecord` | Get a single event instance |
| `RePushEvent` | Re-push a single event through current subscriptions |
| `RePushEvents` | Batch re-push via snapshot — uses batch_jobs |
| `GetRepushStatus` | Poll batch re-push progress |
| `CancelRepush` | Cancel a batch re-push operation |

## Citations

- `proto/webhook.proto` — service definition
- `internal/grpc/event_handlers.go` — implementation
