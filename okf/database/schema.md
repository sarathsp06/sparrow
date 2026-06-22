---
type: Concept
title: Database Schema
description: 11 tables, 23 migrations, comprehensive FK relationships and composite indexes
tags: [database, schema, postgresql, migrations]
timestamp: 2026-06-22T00:00:00Z
---

# Database Schema

## Entity Relationship

```
tenants
  ├── namespaces (tenant_id FK)
  ├── event_registrations (tenant_id FK, composite PK)
  ├── webhook_registrations (tenant_id + namespace FK to namespaces)
  ├── event_subscriptions (tenant_id FK)
  └── event_records (tenant_id FK)

webhook_registrations
  ├── event_subscriptions (webhook_id FK)
  ├── webhook_deliveries (webhook_id FK)
  ├── webhook_health_events (webhook_id FK)
  ├── webhook_health_summaries (webhook_id FK)
  ├── webhook_health_state (webhook_id FK, UNIQUE)
  └── webhook_rate_limit_state (webhook_id PK)

event_records → webhook_deliveries (event_id FK)
event_subscriptions → webhook_deliveries (subscription_id FK, SET NULL)
```

## Table Details

### tenants
Basic tenant info. Default tenant auto-created on startup.

### namespaces
`(tenant_id, name)` UNIQUE. Scopes webhooks and events.

### event_registrations
Composite PK `(tenant_id, name)`. Includes optional JSON schema, active flag.

### webhook_registrations
23 columns — URL, HTTP config, secrets, health, rate limit, Ed25519 key, signature type. FK to namespaces via `(tenant_id, namespace)`.

### event_subscriptions
11 columns — binds webhook to event. Includes `transform_template` (Go template) and `label_filters`.

### event_records
10 columns — pushed event instance with payload, optional `idempotency_key` (partial unique index), `schema_valid` flag, TTL expiry.

### webhook_deliveries
16 columns — status, attempts, error category, response capture. FKs to webhook, event, subscription.

### webhook_health_events
9 columns — per-delivery health measurement.

### webhook_health_summaries
17 columns — rolling window summaries with p95, success rate, error breakdown. `(webhook_id, window_start, window_end)` UNIQUE.

### webhook_health_state
8 columns — current state per webhook, `webhook_id` UNIQUE.

### batch_jobs
11 columns — generic batch infrastructure, `job_type` + JSONB `data`.

### webhook_rate_limit_state
5 columns — leaky bucket state, `webhook_id` PK.

## Composite Indexes (migration 000010)

| Index | Columns | Purpose |
|-------|---------|---------|
| `idx_event_subscriptions_tenant_ns_event` | `(tenant_id, namespace, event_name)` | Fan-out query |
| `idx_webhook_deliveries_webhook_created` | `(webhook_id, created_at DESC)` | Delivery listing |
| `idx_webhook_deliveries_event_created` | `(event_id, created_at DESC)` | Delivery-by-event |
| `idx_event_records_tenant_ns_created` | `(tenant_id, namespace, created_at DESC)` | Event listing |
| `idx_event_records_tenant_event_created` | `(tenant_id, event, created_at DESC)` | Event name filter |
| `idx_webhook_registrations_tenant_ns_active` | `(tenant_id, namespace, active)` | Filtered listing |

## FK Cascade Map

- All `tenant_id` FKs → `tenants.id` with CASCADE
- `webhook_registrations(tenant_id, ns)` → `namespaces(tenant_id, name)` CASCADE
- `webhook_deliveries.subscription_id` → `event_subscriptions.id` SET NULL
- All other entity FKs → parent with CASCADE

## Migrations (23)

`db/migrations/` — 23 pairs of `.up.sql` / `.down.sql`.

| # | Description |
|---|-------------|
| 000001 | Consolidated initial schema |
| 000002 | Error categories |
| 000003 | error_category on deliveries |
| 000004 | Tenants table + FK migration |
| 000005 | Tenant external_id (no-op) |
| 000006 | Tenant created_by |
| 000007 | Remove event registration UUID |
| 000008 | Namespaces table |
| 000009 | Webhook namespace FK |
| 000010 | Composite indexes + drop unused |
| 000011 | Remove namespace_memberships |
| 000012 | Labels on events/subscriptions |
| 000013 | Drop unused namespace entities |
| 000014 | Secret headers |
| 000015 | Envelope encryption migration |
| 000016 | schema_valid flag on event_records |
| 000017 | batch_jobs table |
| 000018 | Unexpected status errors in summaries |
| 000019 | Drop system_settings table |
| 000020 | Idempotency key + partial unique index |
| 000021 | Rate limiting columns + state table |
| 000022 | Ed25519 private key column |
| 000023 | Signature type column |

## Citations

- `db/migrations/` — all migration files
- `opencode.md` — ER diagram and FK cascade map
