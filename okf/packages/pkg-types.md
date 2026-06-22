---
type: Go Package
title: pkg/types
description: Generic Map[K, V] type with PostgreSQL JSONB-compatible Scan/Value implementations
tags: [types, db, leaf]
timestamp: 2026-06-22T00:00:00Z
---

# pkg/types

Generic `Map[K comparable, V any]` type with `database/sql` `Scan` and `driver.Valuer` implementations for JSONB columns.

## Methods

- `Keys() []K`
- `Values() []V`

## Citations

- `pkg/types/map.go`
