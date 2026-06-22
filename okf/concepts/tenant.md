---
type: Concept
title: Tenant
description: Multi-tenant model — default tenant auto-created on startup, all operations use it
tags: [tenant, multi-tenancy]
timestamp: 2026-06-22T00:00:00Z
---

# Tenant

Infrastructure for multi-tenancy exists in the database (`tenants` table with FK references from all entities), but Sparrow currently operates with a **single default tenant** (`00000000-0000-0000-0000-000000000001`). All API operations use this tenant ID.

Tenant infrastructure is retained for future multi-tenant SaaS mode but is not currently active.

## Key Fields

- `id` UUID (PK)
- `name`, `slug` (unique), `status` (active/suspended/archived)

## Citations

- `internal/tenant/` — bootstrap, model, repository
