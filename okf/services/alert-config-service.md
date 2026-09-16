---
type: REST Resource
title: Alert Configs
description: Manage per-consumer email alert recipients that receive system-event notifications
tags: [rest, alerts]
timestamp: 2026-09-16T00:00:00Z
---

# Alert Configs

Per-consumer email alert recipients. List, register, and delete the addresses that receive Sparrow system events — webhook health changes and delivery failures — delivered through the bootstrapped SendGrid alert webhook (see [alert configuration](/config/env-vars.md)). Registered under the `Alert Configs` tag.

The authoritative endpoint list — paths, methods, and schemas — is the OpenAPI spec at `api/openapi.yaml` (browse at `/docs`).

## Citations

- `internal/rest/` — endpoint registration + handlers
- `api/openapi.yaml` — canonical endpoint definitions
