# Services

REST resources exposed over a single Huma-generated OpenAPI surface, versioned under `/v1`. The authoritative, always-current endpoint list — paths, methods, request/response schemas — lives in the OpenAPI spec (`api/openapi.yaml`, browsable at `/docs`). These files describe each resource at a high level so they stay valid as endpoints are added or changed.

* [Webhooks](webhook-service.md) — register, configure, pause/resume webhooks; delivery stats; portal-token minting
* [Events & Event Types](event-service.md) — define event types and push/replay event occurrences
* [Subscriptions](subscription-service.md) — bind webhooks to events with label filters and template transforms
* [Deliveries](delivery-service.md) — delivery status, per-attempt history, and retries (single, bulk, batch)
* [Health](health-service.md) — per-webhook and aggregate health queries
* [Alert Configs](alert-config-service.md) — per-consumer email alert recipients for system events
