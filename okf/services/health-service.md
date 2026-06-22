---
type: gRPC Service
title: HealthService
description: Webhook health queries — status, filtering, summary — 3 RPCs
tags: [grpc, health]
timestamp: 2026-06-22T00:00:00Z
---

# HealthService

Defined in [`webhook.proto`](/references/proto.md).

## RPCs

| RPC | Description |
|-----|-------------|
| `GetWebhookHealth` | Health state for a single webhook |
| `ListWebhooksByHealth` | List webhooks filtered by health status |
| `GetHealthSummary` | Aggregate health summary across webhooks |

## Citations

- `proto/webhook.proto` — service definition
- `internal/grpc/health_handlers.go` — implementation
