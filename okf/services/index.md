# Services

5 protobuf-defined gRPC/Connect-RPC services + 1 Go-only namespace service.

* [WebhookService](webhook-service.md) — 8 RPCs: register, list, update, pause/resume webhooks
* [EventService](event-service.md) — 12 RPCs: register, push, list, re-push events
* [SubscriptionService](subscription-service.md) — 6 RPCs: CRUD + template testing
* [DeliveryService](delivery-service.md) — 7 RPCs: status, list, retry deliveries
* [HealthService](health-service.md) — 3 RPCs: webhook health queries
* [NamespaceService](namespace-service.md) — 5 RPCs: CRUD (Go-only, not in proto)
