---
type: REST Resource
title: Deliveries
description: Inspect delivery status and attempt history; retry single, bulk, or batched deliveries
tags: [rest, deliveries]
timestamp: 2026-09-16T00:00:00Z
---

# Deliveries

Read and retry webhook [deliveries](/concepts/delivery.md). Exposes delivery status, per-attempt history, single and bulk (per-webhook) retries, and snapshot-based batch retry jobs with progress polling and cancellation — offered both consumer-scoped and in global (cross-consumer) variants. Registered under the `Deliveries` tag.

The authoritative endpoint list — paths, methods, and schemas — is the OpenAPI spec at `api/openapi.yaml` (browse at `/docs`).

## Citations

- `internal/rest/delivery.go` — endpoint registration + handlers
- `api/openapi.yaml` — canonical endpoint definitions
