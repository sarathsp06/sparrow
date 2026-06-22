---
type: gRPC Service
title: DeliveryService
description: Delivery status, listing, retry, and batch retry — 7 RPCs
tags: [grpc, deliveries]
timestamp: 2026-06-22T00:00:00Z
---

# DeliveryService

Defined in [`webhook.proto`](/references/proto.md).

## RPCs

| RPC | Description |
|-----|-------------|
| `GetDeliveryStatus` | Get a single delivery status |
| `ListDeliveries` | Filterable delivery listing (supports `prepare_retry`) |
| `RetryDelivery` | Retry a single failed delivery |
| `GetDeliveryAttempts` | List HTTP attempt records for a delivery |
| `RetryDeliveries` | Batch retry via snapshot — uses batch_jobs |
| `GetRetryStatus` | Poll batch retry progress |
| `CancelRetry` | Cancel a batch retry operation |

## Citations

- `proto/webhook.proto` — service definition
- `internal/grpc/delivery_handlers.go` — implementation
