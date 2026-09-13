---
type: Go Package
title: pkg/template
description: Cached Go template engine for payload transforms, shared by the delivery worker and satellites
tags: [template, transform, pkg]
timestamp: 2026-09-13T00:00:00Z
---

# pkg/template

Payload-transform engine extracted from `internal/webhooks/client` so both the
core delivery worker and the out-of-tree satellites (`satellites/recipes`,
`satellites/sparrow` CLI) can render templates without importing `internal/`.

## Key Types

- `TemplateEngine` — cached Go `text/template` execution; 1 MB output limit, 5s CPU timeout, per-execution buffer pool
- `TemplateCache` — LRU of parsed templates keyed by SHA-256 of the source (`hashicorp/golang-lru/v2`)
- `WebhookTemplateContext` — snake_case fields (`event_id`, `event_name`, `timestamp`, `attempt`, `payload`) passed to every transform
- `GetTemplateFunctions()` / `GetFunctionMap()` — utility funcs available in templates (`upper`, `json`, …)

## Consumers

- `internal/webhooks/client` — `WebhookClient.TransformPayload` delegates here
- `internal/webhooks/queue/webhook_worker.go` — applies the subscription transform at delivery time
- `internal/webhooks/webhook_service_subscription.go` — `TestSubscriptionTemplate` dry-run
- `satellites/sparrow/template_cmd.go` and `satellites/recipes` — render recipes/CLI previews

## Citations

- `pkg/template/` — template.go, cache.go, functions.go, pool.go
