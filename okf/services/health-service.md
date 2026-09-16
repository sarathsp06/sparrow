---
type: REST Resource
title: Health
description: Query per-webhook health status and metrics, and aggregate health counts across consumers
tags: [rest, health]
timestamp: 2026-09-16T00:00:00Z
---

# Health

Read-only views over event-sourced [webhook health](/concepts/webhook-health.md): the status and metrics for a single webhook, an aggregate health-count summary, and a listing of webhooks by health state across all consumers. Registered under the `Health` tag.

The authoritative endpoint list — paths, methods, and schemas — is the OpenAPI spec at `api/openapi.yaml` (browse at `/docs`).

## Citations

- `internal/rest/health.go` — endpoint registration + handlers
- `api/openapi.yaml` — canonical endpoint definitions
