---
type: Go Package
title: internal/tenant
description: Tenant model, repository, service, and bootstrap — default tenant auto-created on startup
tags: [tenant, multi-tenancy]
timestamp: 2026-06-22T00:00:00Z
---

# internal/tenant

Provides the tenant model and infrastructure. A **default tenant** (`00000000-0000-0000-0000-000000000001`) is auto-created on startup via `Bootstrap()`. All operations use this tenant ID.

## Key Types

- `Tenant` — ID, Name, Slug, Status, Settings, CreatedAt, UpdatedAt
- `Repository` interface — CRUD operations
- `Service` — wraps repository

## Key Symbols

- `DefaultTenantID` — well-known UUID for all operations
- `StatusActive`, `StatusSuspended`, `StatusArchived`
- `Bootstrap(ctx, svc, cfg) error`

## Citations

- `internal/tenant/` — 4 files
