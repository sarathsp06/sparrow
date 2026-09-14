---
type: Concept
title: Consumer
description: Scoping mechanism for webhook registrations and events within a tenant
tags: [consumer, scoping]
timestamp: 2026-06-22T00:00:00Z
---

# Consumer

Consumers provide logical scoping for webhooks and events within a tenant. All resources are consumer-scoped: webhook registrations, event registrations, event records, subscriptions.

## Table

`consumers` table with `(tenant_id, name)` UNIQUE constraint.

[Webhook registrations](/concepts/webhook-registration.md) reference consumers via FK `(tenant_id, consumer)`.

## Citations

- `internal/consumer/`
- Migration 000008 (create consumers), 000009 (FK from webhooks)
