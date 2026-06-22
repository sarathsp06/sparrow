---
type: Concept
title: Namespace
description: Scoping mechanism for webhook registrations and events within a tenant
tags: [namespace, scoping]
timestamp: 2026-06-22T00:00:00Z
---

# Namespace

Namespaces provide logical scoping for webhooks and events within a tenant. All resources are namespace-scoped: webhook registrations, event registrations, event records, subscriptions.

## Table

`namespaces` table with `(tenant_id, name)` UNIQUE constraint.

[Webhook registrations](/concepts/webhook-registration.md) reference namespaces via FK `(tenant_id, namespace)`.

## Citations

- `internal/namespace/`
- Migration 000008 (create namespaces), 000009 (FK from webhooks)
