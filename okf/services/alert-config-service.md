---
type: REST Resource
title: Alert Configs
description: Manage per-consumer email alert recipients that receive system-event notifications
tags: [rest, alerts]
timestamp: 2026-10-08T00:00:00Z
---

# Alert Configs

Per-consumer email alert recipients. List, register, and delete the addresses that receive Sparrow system events — [webhook health](/concepts/webhook-health.md) changes, permanent delivery failures and auto-disables — pushed under the internal `_sparrow` consumer. Delivery to email is wired by applying the SendGrid recipe as a normal webhook under `_sparrow` (not by any bootstrap or env var); `GET /v1/capabilities` reports `alert_delivery.configured` once one exists. The webhook detail page lists, adds and removes a webhook's alert configs; the [webhook health alerts guide](https://sarathsp06.github.io/sparrow/guides/webhook-health-alerts/) is the how-to. Registered under the `Alert Configs` tag.

The authoritative endpoint list — paths, methods, and schemas — is the OpenAPI spec at `api/openapi.yaml` (browse at `/docs`).

## Citations

- `internal/rest/` — endpoint registration + handlers
- `api/openapi.yaml` — canonical endpoint definitions
