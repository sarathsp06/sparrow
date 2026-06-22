---
type: Go Package
title: internal/observability
description: OpenTelemetry setup — traces, metrics, logs via OTLP HTTP export
tags: [observability, tracing, metrics, otel]
timestamp: 2026-06-22T00:00:00Z
---

# internal/observability

Initializes OpenTelemetry tracing, metrics, and logging with OTLP HTTP export.

## SparrowMetrics

Application-level metrics:

| Metric | Type |
|--------|------|
| `webhook_registrations` | Counter |
| `events_pushed` | Counter |
| `webhook_deliveries` (by status) | Counter |
| `delivery_duration` | Histogram |
| `queue_depth` | Up/Down Counter |
| `active_webhooks` | Up/Down Counter |

## Citations

- `internal/observability/observability.go`
