---
type: gRPC Service
title: WebhookService
description: Webhook registration and configuration management — 8 RPCs
tags: [grpc, webhooks]
timestamp: 2026-06-22T00:00:00Z
---

# WebhookService

Defined in [`webhook.proto`](/references/proto.md). Implemented by `WebhookServer` in `internal/grpc/`.

## RPCs

| RPC | Description |
|-----|-------------|
| `RegisterWebhook` | Create webhook + auto-create subscriptions |
| `UnregisterWebhook` | Delete webhook (cascades subscriptions, deliveries) |
| `ListWebhooks` | Paginated listing with filters |
| `UpdateWebhookConfig` | Partial update with field mask |
| `PauseWebhook` | Set webhook inactive |
| `ResumeWebhook` | Set webhook active |
| `GetNamespaceStats` | Aggregate per-namespace delivery stats |
| `GetTemplateFunctions` | List available Go template functions |

## Citations

- `proto/webhook.proto` — service definition
- `internal/grpc/webhook_handlers.go` — implementation
