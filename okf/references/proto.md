---
type: Concept
title: Proto Definitions
description: Single webhook.proto file defining 5 services, ~36 RPCs, and generated code for Go, JS/TS, and Python
tags: [proto, protobuf, codegen]
timestamp: 2026-06-22T00:00:00Z
---

# Proto Definitions

Single proto file (`proto/webhook.proto`, ~1950 lines) defines the complete API surface.

## Services (5)

| Service | RPCs | Package |
|---------|------|---------|
| [WebhookService](/services/webhook-service.md) | 8 | `webhook.WebhookService` |
| [EventService](/services/event-service.md) | 12 | `webhook.EventService` |
| [SubscriptionService](/services/subscription-service.md) | 6 | `webhook.SubscriptionService` |
| [DeliveryService](/services/delivery-service.md) | 7 | `webhook.DeliveryService` |
| [HealthService](/services/health-service.md) | 3 | `webhook.HealthService` |

## Generated Code

| Output | Plugin | Location |
|--------|--------|----------|
| Go protobuf types | `protoc-gen-go` | `proto/webhook.pb.go` |
| Go gRPC server/client | `protoc-gen-go-grpc` | `proto/webhook_grpc.pb.go` |
| Go Connect-RPC | `protoc-gen-connect-go` | `proto/protoconnect/` |
| JS/TS (web UI) | `protoc-gen-es` | `proto/webhook_pb.js`, `.d.ts` |
| Go client library | buf generate | `client/go/` |
| JS/TS client | buf generate | `client/js/` |
| Python client | buf generate | `client/python/` |
| API docs | protoc-gen-doc | `docs/` |

## Config

- `buf.yaml` — v1 module, FILE-level breaking detection, DEFAULT lint
- `buf.gen.yaml` — 7 plugin groups

## Citations

- `proto/webhook.proto`
- `buf.gen.yaml`, `buf.yaml`
