---
type: gRPC Service
title: SubscriptionService
description: Subscription CRUD with Go template transforms — 6 RPCs
tags: [grpc, subscriptions]
timestamp: 2026-06-22T00:00:00Z
---

# SubscriptionService

Defined in [`webhook.proto`](/references/proto.md).

## RPCs

| RPC | Description |
|-----|-------------|
| `CreateSubscription` | Subscribe a webhook to an event |
| `GetSubscription` | Get subscription details |
| `ListSubscriptions` | List subscriptions for a webhook or event |
| `UpdateSubscription` | Update subscription config, transform template |
| `DeleteSubscription` | Delete subscription |
| `TestSubscriptionTemplate` | Preview Go template transform output |

## Citations

- `proto/webhook.proto` — service definition
- `internal/grpc/subscription_handlers.go` — implementation
