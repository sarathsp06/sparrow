# AGENTS.md — Sparrow

## First reads

- `opencode.json` loads `plan.md` as instructions — keep it accurate.
- `okf/index.md` has the full architecture overview (knowledge bundle at `okf/`).
- `opencode.md` is a condensed reference with design principles, code patterns, handler patterns, and naming conventions.

## Quick commands

| Action | Command | Notes |
|--------|---------|-------|
| Build server | `make build` | Output: `build/server-$(GOOS)-$(GOARCH)` |
| Build UI | `make build-ui` | Svelte 5 → `internal/ui/dist/` (embedded via `go:embed`) |
| Build both | `make build-with-ui` | |
| Run (dev) | `make run` | Needs `DATABASE_URL`, `SPARROW_ENCRYPTION_KEY` |
| Run UI dev server | `make run-web` | Hot-reload Svelte at localhost (:5173 default) |
| Run all tests | `make test` | `go test -v ./...` |
| Single package test | `go test -v ./internal/webhooks/...` | |
| Integration tests | `make test-integration` | Needs Docker (testcontainers) |
| E2E tests | `make test-e2e` | Gauge + Python, needs Docker |
| Lint | `make lint` | golangci-lint, 15m timeout |
| Format | `make fmt` | `goimports -local github.com/sarathsp06/sparrow/ -w .` |
| Run migrations | `make migrate` | Also runs automatically on server startup |
| Generate protos + clients | `make generate` | `buf generate` then `go generate ./...` |
| gRPC codegen | `go generate ./...` | Uses gowrap for OTel tracing wrappers |

## Architecture

- **Entrypoint**: `cmd/server/main.go` — wires chi router, gRPC, River queue, OTel, bootstraps default tenant.
- **Dual protocol**: gRPC on `:50051`, Connect-RPC (HTTP/JSON) on `:8080`, same handlers.
- **Queue**: River (Postgres-backed, 45 concurrent workers: 20 events + 20 webhooks + 5 default).
- **DB**: pgxpool (50 conns, 10 min) for River + sqlx (25 conns) for app queries.
- **Config**: env vars via `kelseyhightower/envconfig` — see `internal/config/config.go`.
- **Proto**: `proto/webhook.proto` → 5 services (Webhook, Event, Subscription, Delivery, Health).

## Key packages

| Path | Purpose |
|------|---------|
| `cmd/server/` | Server entrypoint + DI wiring |
| `cmd/migrate/` | Standalone migration runner |
| `internal/grpc/` | gRPC handler layer (thin, calls webhook service) |
| `internal/connect/` | Connect-RPC handler wrapper |
| `internal/webhooks/` | Business logic + store + queue workers |
| `internal/webhooks/store/` | DB repository (sqlx, WithConn transaction pattern) |
| `internal/webhooks/queue/` | River job types + workers |
| `internal/middleware/` | API key auth, security headers |
| `pkg/storage/` | DB abstractions, transaction helpers, error sentinels |
| `pkg/crypto/` | Envelope encryption (AES-256-GCM) |
| `pkg/errors/` | Error categories, service errors, retryability |

## Code conventions

- **Error order**: All `if err != nil { return ... }` before happy path.
- **WithConn**: `repo.WithConn(tx)` inside `storage.WithTransaction()` for repo-level txns.
- **No direct SQL in handlers** — all DB access through RepositoryInterface methods.
- **gRPC errors**: use `toGRPCError(ctx, err, msg)` from `internal/grpc/helpers.go`.
- **Tenant scoping**: always filter by `tenant.DefaultTenantID` in queries.
- **Naming**: files `snake_case.go`, packages lowercase single word, proto services `SomethingService`.
- **OTel wrappers**: generated via `//go:generate gowrap gen -i InterfaceName ...` — do not hand-edit `*_otel.go` files.
- **Proto path for Go imports**: `github.com/sarathsp06/sparrow/proto` (not `protoconnect/`).

## Frontend (Svelte 5)

- Static SPA via `@sveltejs/adapter-static` — no SSR.
- Output dir: `../internal/ui/dist` (embedded in Go binary).
- Connect-RPC client for API calls (`@connectrpc/connect-web`).
- Dev server: `npm run dev` from `web/`.

## DB / Migrations

- Migrations run **on server startup** automatically (before anything else). No separate step needed in prod.
- Also runnable standalone via `cmd/migrate`.
- Location: `db/migrations/000001_...up.sql` etc.
- Uses `golang-migrate` with PostgreSQL advisory locks (safe for concurrent instances).

## Test nuances

- Unit tests use `DATABASE_URL` from environment (CI provides a postgres service).
- Integration tests (`-tags integration`) use testcontainers — need Docker.
- E2E tests (Gauge + Python) in `e2e/` — `uv run gauge run specs/`.
- `make fmt` uses `goimports` with local module grouping — install `go install golang.org/x/tools/cmd/goimports@latest`.

## Release

- Tags trigger GoReleaser CI: `git tag vX.Y.Z && git push origin main --tags`.
- GoReleaser config at `.goreleaser.yml`.
- Conventional Commits (`feat:`, `fix:`, etc.) for clean changelog grouping.
