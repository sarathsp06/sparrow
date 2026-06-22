# Go Packages

* [cmd/server](cmd-server.md) — main entry point, dependency injection
* [internal/config](internal-config.md) — configuration loading from env vars
* [internal/webhooks (service)](internal-webhooks-service.md) — core business logic
* [internal/webhooks/store](internal-webhooks-store.md) — database repository layer
* [internal/webhooks/queue](internal-webhooks-queue.md) — River job queue workers
* [internal/webhooks/client](internal-webhooks-client.md) — HTTP delivery client
* [internal/grpc](internal-grpc.md) — gRPC handler implementations
* [internal/connect](internal-connect.md) — Connect-RPC adapter
* [internal/middleware](internal-middleware.md) — API key auth + security headers
* [internal/tenant](internal-tenant.md) — tenant model and bootstrap
* [internal/namespace](internal-namespace.md) — namespace CRUD
* [internal/health](internal-health.md) — health check endpoints
* [internal/observability](internal-observability.md) — OTel setup
* [internal/logger](internal-logger.md) — structured logging
* [pkg/crypto](pkg-crypto.md) — envelope encryption
* [pkg/errors](pkg-errors.md) — error classification
* [pkg/storage](pkg-storage.md) — database abstraction
* [pkg/types](pkg-types.md) — generic Map type
