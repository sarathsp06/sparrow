---
type: Go Package
title: internal/logger
description: Structured slog-based logging with OTel bridge
tags: [logging, leaf]
timestamp: 2026-06-22T00:00:00Z
---

# internal/logger

Provides structured logging via `slog` with OTel integration.

## Key Exports

- `Logger *slog.Logger` — package-level global
- `NewLogger(name string) *slog.Logger` — creates named logger with `component` attribute
- `SetLevel(level slog.Level)` — configures JSON handler with OTel bridge

## Citations

- `internal/logger/logger.go`
