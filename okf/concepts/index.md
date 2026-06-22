# Domain Concepts

* [Tenant](tenant.md) — multi-tenant model (default tenant active)
* [Namespace](namespace.md) — scoping for webhooks and events
* [Webhook Registration](webhook-registration.md) — webhook URL, config, secrets
* [Event](event.md) — event types, records, schema validation
* [Subscription](subscription.md) — event-to-webhook binding with transforms
* [Delivery](delivery.md) — webhook HTTP delivery lifecycle
* [Webhook Health](webhook-health.md) — health state machine and tracking
* [Batch Jobs](batch-jobs.md) — snapshot-based bulk operations
* [Rate Limiting](rate-limiting.md) — per-webhook leaky bucket
* [Envelope Encryption](envelope-encryption.md) — AES-256-GCM with per-record DEK
* [Payload Signing](payload-signing.md) — dual HMAC-SHA256 + Ed25519
* [Error Classification](error-classification.md) — 10 delivery error categories
