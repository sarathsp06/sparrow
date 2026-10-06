---
type: Go Package
title: internal/observability
description: OpenTelemetry setup — traces, metrics, logs via OTLP export (HTTP or gRPC)
tags: [observability, tracing, metrics, otel]
timestamp: 2026-06-22T00:00:00Z
---

# internal/observability

Initializes OpenTelemetry tracing, metrics, and logging with OTLP export over HTTP (default) or gRPC, selected by `Config.OTLPProtocol`. Endpoint, TLS and headers come from the standard `OTEL_EXPORTER_OTLP_*` env vars, read by the exporters themselves; a bare `host:port` endpoint is passed explicitly as plaintext for backward compatibility. `Config.OTLPSignals` (`SignalTraces`, `SignalMetrics`, `SignalLogs`; empty = all) selects which exporters are built, so a traces-only backend such as Jaeger or Tempo never receives metric or log uploads it would reject. The Prometheus reader behind `MetricsHandler` is independent of this list.

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
