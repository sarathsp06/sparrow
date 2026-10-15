# Go Packages

* [cmd/server](cmd-server.md) — main entry point, dependency injection
* [internal/config](internal-config.md) — configuration loading from env vars
* [internal/webhooks (service)](internal-webhooks-service.md) — core business logic
* [internal/webhooks/store](internal-webhooks-store.md) — database repository layer
* [internal/webhooks/queue](internal-webhooks-queue.md) — River job queue workers
* [internal/webhooks/client](internal-webhooks-client.md) — HTTP delivery client
* [internal/middleware](internal-middleware.md) — auth (master key + access tokens), CORS, portal gateway, security headers
* [internal/accessauth](internal-accessauth.md) — Sparrow adapter for pkg/access (realm, scope, prefixes, lifetime rules)
* [internal/tenant](internal-tenant.md) — tenant model and bootstrap
* [internal/health](internal-health.md) — health check endpoints
* [internal/observability](internal-observability.md) — OTel setup
* [internal/crypto](internal-crypto.md) — envelope encryption
* [internal/errors](internal-errors.md) — error classification
* [internal/storage](internal-storage.md) — database abstraction
* [pkg/template](pkg-template.md) — cached Go template engine for payload transforms
* [pkg/access](pkg-access.md) — separate module: reusable token/invite library (memstore, pgstore, storetest, httpauth)
