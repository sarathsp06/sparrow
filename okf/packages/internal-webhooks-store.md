---
type: Go Package
title: internal/webhooks/store
description: Database repository with ~80 methods, sqlx-based, WithConn pattern for transactions
tags: [database, repository, core]
timestamp: 2026-06-22T00:00:00Z
---

# internal/webhooks/store

The repository layer that [depends on](/packages/pkg-storage.md) for database access.

## Key Types

- `RepositoryInterface` — master storage interface (~80 methods)
- `Repository` — concrete impl using `storage.DB` + `storage.DBTX`
- Models for all 11+ tables

## Enums

- `WebhookHealth` — healthy, degraded, unhealthy, unknown
- `WebhookDeliveryStatus` — pending, sending, success, failed, retrying, expired
- `SignatureType` — hmac, ed25519
- `BatchJobStatus`, `BatchJobType`

## Constants

- `CatchAllEventName = "*"`
- `MaxBatchSize = 10000`
- `DefaultBatchTTLSeconds = 900`

## Pattern

```go
type Repository struct {
    db   storage.DB
    conn storage.DBTX
}

func (r *Repository) WithConn(conn storage.DBTX) *Repository {
    return &Repository{db: r.db, conn: conn}
}
```

Uses `storage.WithTransaction` for transactional operations.

## Citations

- `internal/webhooks/store/` — 11 files
