---
title: API Reference
description: Complete API reference for all Sparrow services
---

Sparrow exposes its API through two protocols from the same service definitions:

- **Connect-RPC (HTTP/JSON)** on port `8080` — use with `curl`, browsers, or any HTTP client
- **gRPC** on port `50051` — use with generated gRPC clients for high-performance access

All Connect-RPC endpoints use `POST` with JSON request/response bodies. The URL pattern is:

```
POST http://localhost:8080/webhook.{ServiceName}/{RpcName}
```

## Services

| Service | Description | RPCs |
|---------|-------------|------|
| [WebhookService](/sparrow/reference/api/webhook-service/) | Register, configure, pause/resume webhooks | 8 |
| [EventService](/sparrow/reference/api/event-service/) | Define event types and push events | 7 |
| [SubscriptionService](/sparrow/reference/api/subscription-service/) | Link webhooks to events with optional transforms | 6 |
| [DeliveryService](/sparrow/reference/api/delivery-service/) | View delivery history and retry failed deliveries | 4 |
| [HealthService](/sparrow/reference/api/health-service/) | Monitor webhook health metrics | 3 |

## Authentication

Sparrow has **no authentication**. All endpoints are open. It is designed for self-hosted deployments behind your own network boundary.

## Pagination

All list endpoints accept optional pagination parameters:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `limit` | `int32` | `50` | Max items to return (clamped to server max) |
| `offset` | `int32` | `0` | Items to skip |

Responses include a `pagination` object with `total_count`, `limit`, and `offset`.

## Error Handling

Connect-RPC errors are returned as JSON with a `code` field matching gRPC status codes:

```json
{
  "code": "not_found",
  "message": "webhook not found"
}
```

Common error codes: `not_found`, `already_exists`, `invalid_argument`, `internal`.
