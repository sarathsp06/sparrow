---
type: REST Resource
title: Subscriptions
description: Bind webhooks to event types with label filters and Go-template payload transforms; CRUD plus template testing
tags: [rest, subscriptions]
timestamp: 2026-09-16T00:00:00Z
---

# Subscriptions

Links a webhook to an event type, optionally narrowed by [label filters](/concepts/subscription.md) and reshaped by a Go transform template. Supports full CRUD plus a template-testing endpoint that renders a transform against an event's sample payload. Registered under the `Subscriptions` tag.

The authoritative endpoint list — paths, methods, and schemas — is the OpenAPI spec at `api/openapi.yaml` (browse at `/docs`).

## Citations

- `internal/rest/subscription.go` — endpoint registration + handlers
- `api/openapi.yaml` — canonical endpoint definitions
