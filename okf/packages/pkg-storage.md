---
type: Go Package
title: pkg/storage
description: Database abstraction — DBTX/DB interfaces, WithTransaction helper, PG error translation
tags: [database, abstraction, leaf]
timestamp: 2026-06-22T00:00:00Z
---

# pkg/storage

Abstracts database access with DBTX/DB interfaces and PostgreSQL error translation.

## Interfaces

```go
type DBTX interface {
    GetContext(ctx, dest, query, args...) error
    SelectContext(ctx, dest, query, args...) error
    NamedExecContext(ctx, query, arg) (sql.Result, error)
    ExecContext(ctx, query, args...) (sql.Result, error)
}

type DB interface {
    DBTX
    Ping() error
    Close() error
    Beginx() (*sqlx.Tx, error)
}
```

## PG Error Translation

| PG Error | Semantic Error |
|----------|---------------|
| `sql.ErrNoRows` | `ErrNotFound` |
| 23505 (unique_violation) | `ErrAlreadyExists` |
| 23502 (not_null_violation) | `ErrInvalidInput` |
| 23503 (foreign_key_violation) | `ErrForeignKeyViolation` |

## Helper

- `WithTransaction(db, func(DBTX) error) error`

## Citations

- `pkg/storage/storage.go`, `pkg/storage/errors.go`
