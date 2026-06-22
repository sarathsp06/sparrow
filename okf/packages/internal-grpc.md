---
type: Go Package
title: internal/grpc
description: gRPC handler implementations for all 5 protobuf-defined services
tags: [grpc, handlers, api]
timestamp: 2026-06-22T00:00:00Z
---

# internal/grpc

Implements all 5 protobuf-defined gRPC services as handler methods across 9 files.

## WebhookServer

A single struct embeds all 5 `Unimplemented*Server` types:

```go
type WebhookServer struct {
    pb.UnimplementedWebhookServiceServer
    pb.UnimplementedEventServiceServer
    pb.UnimplementedSubscriptionServiceServer
    pb.UnimplementedDeliveryServiceServer
    pb.UnimplementedHealthServiceServer
    svc webhooks.WebhookServiceInterface
}
```

## Handler Files

| File | RPCs |
|------|------|
| `webhook_handlers.go` | 8 (Register, Unregister, List, Update, Pause, Resume, GetNamespaceStats, GetTemplateFunctions) |
| `event_handlers.go` | 12 (Register, List, Update, Delete, Get, Push, ListReports, GetRecord, RePush, RePushEvents, GetRepushStatus, CancelRepush) |
| `subscription_handlers.go` | 6 (Create, Get, List, Update, Delete, TestTemplate) |
| `delivery_handlers.go` | 7 (GetStatus, List, Retry, GetAttempts, RetryDeliveries, GetRetryStatus, CancelRetry) |
| `health_handlers.go` | 3 (GetHealth, ListByHealth, GetSummary) |

## Pattern

Each handler: uses default tenant → validates input → calls service → translates errors to gRPC status codes.

## Citations

- `internal/grpc/` — 9 files
