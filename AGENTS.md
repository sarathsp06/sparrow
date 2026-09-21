# AGENTS.md — Sparrow

## First reads

- `okf/index.md` has the full architecture overview (knowledge bundle at `okf/`).
- This file has design principles, code patterns, handler patterns, and naming conventions.

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
| Generate spec + clients | `make generate` | Exports OpenAPI from Go, regenerates the Python client, then `go generate ./...` |
| OTel wrapper codegen | `go generate ./...` | Uses gowrap for OTel tracing wrappers |

## Architecture

- **Entrypoint**: `cmd/server/main.go` — wires chi router, Huma REST API, River queue, OTel, bootstraps default tenant.
- **REST/OpenAPI**: single HTTP transport on `:8080`. [Huma](https://github.com/danielgtaylor/huma) generates the OpenAPI 3.1 spec from Go handler structs; served interactively at `/docs` (Scalar), spec at `/openapi.yaml`/`/openapi.json`.
- **Queue**: River (Postgres-backed, 45 concurrent workers: 20 events + 20 webhooks + 5 default).
- **DB**: pgxpool (50 conns, 10 min) for River + sqlx (25 conns) for app queries.
- **Config**: env vars via `kelseyhightower/envconfig` — see `internal/config/config.go`.

## Design Principles

1. **Deterministic bulk operations** — Batch actions snapshot matching IDs into `batch_jobs` at query time. The bulk action operates on that snapshot, NOT a live re-query.
2. **Soft validation over hard rejection** — Schema validation produces warnings, not errors. Events tagged `schema_valid=false`, never discarded.
3. **Graceful degradation** — When a non-critical step fails (e.g., Go template transform), fall back to a safe default (envelope payload).
4. **Generic infrastructure over per-feature tables** — Shared concerns use generic tables with `job_type` + JSONB `data` columns.
5. **Implicit infrastructure, explicit actions** — Batch jobs are an implementation detail. Users see "re-push ID" and "retry ID", not "batch job IDs".
6. **Postgres-only, no Redis** — All queuing, state, and caching uses PostgreSQL. Don't introduce Redis or other external dependencies without strong justification.
7. **Self-hosted first** — Sparrow targets teams running it behind a VPN for internal webhook delivery, not multi-tenant SaaS.

## Key packages

| Path | Purpose |
|------|---------|
| `cmd/server/` | Server entrypoint + DI wiring |
| `cmd/migrate/` | Standalone migration runner |
| `internal/rest/` | Huma REST handler layer (thin, calls webhook service) — one file per resource (`webhook.go`, `event.go`, `subscription.go`, `delivery.go`, `health.go`) |
| `internal/webhooks/` | Business logic + store + queue workers |
| `internal/webhooks/store/` | DB repository (sqlx, WithConn transaction pattern) |
| `internal/webhooks/queue/` | River job types + workers |
| `internal/middleware/` | API key auth, security headers |
| `pkg/storage/` | DB abstractions, transaction helpers, error sentinels |
| `pkg/crypto/` | Envelope encryption (AES-256-GCM) |
| `pkg/errors/` | Error categories, service errors, retryability |

## API Key Authentication

Optional shared-secret auth via `SPARROW_API_KEY` env var. When set, every `/v1/*` REST request must include `X-API-Key: <key>`. When unset, all endpoints are open. Excluded paths: `/health`, `/ready`, `/docs`, `/openapi`, UI catch-all. Uses constant-time comparison (`crypto/subtle`). The embedded UI gets the key injected at runtime via `window.__SPARROW_CONFIG__`.

## HTTP Routing (chi)

| Pattern | Handler | Auth | Notes |
|---------|---------|------|-------|
| `/v1/*` | Huma REST API | Yes | See `internal/rest/` — one file per resource |
| `/docs`, `/openapi.*` | Huma-served Scalar UI + spec | No | Interactive API reference |
| `GET /health`, `/ready` | Health check | No | JSON status |
| `* (NotFound)` | UI SPA | No | GET/HEAD → HTML; others → JSON 404 |

Route-group middleware (API key auth) wraps only the `/v1/*` group (`r.Group` in `cmd/server/main.go`), not health, docs, or the UI.

## Code conventions

- **Error order**: All `if err != nil { return ... }` before happy path.
- **WithConn**: `repo.WithConn(tx)` inside `storage.WithTransaction()` for repo-level txns.
- **No direct SQL in handlers** — all DB access through RepositoryInterface methods.
- **REST errors**: use `mapError(ctx, err, msg)` from `internal/rest/errors.go`.
- **Tenant scoping**: always filter by `tenant.DefaultTenantID` in queries.
- **Naming**: files `snake_case.go`, packages lowercase single word, REST OperationIDs `camelCase` verb-first (e.g. `registerWebhook`, `listDeliveries`).
- **OTel wrappers**: generated via `//go:generate gowrap gen -i InterfaceName ...` — do not hand-edit `*_otel.go` files.
- **OpenAPI spec**: exported from Go via `cmd/openapi-export`, committed at `api/openapi.{yaml,json}` — regenerate with `make generate` after any handler change; `internal/rest/openapi_drift_test.go` fails CI if it's stale.

### Repository / Storage pattern

```go
type DBTX interface { GetContext, SelectContext, NamedExecContext, ExecContext }
type DB interface { DBTX + Ping, Close, Beginx }

// WithConn pattern (used by all repos)
type Repository struct { db storage.DB; conn storage.DBTX }
func (r *Repository) WithConn(conn storage.DBTX) *Repository
```

SQL error translation: `sql.ErrNoRows` → `ErrNotFound`, PG 23505 → `ErrAlreadyExists`, PG 23502 → `ErrInvalidInput`, PG 23503 → `ErrForeignKeyViolation`.

### Handler pattern

```go
huma.Register(api, huma.Operation{
    OperationID: "getWebhook",
    Method:      http.MethodGet,
    Path:        "/v1/consumers/{consumer}/webhooks/{webhook_id}",
    Summary:     "Get a webhook by id",
    Tags:        []string{"Webhooks"},
}, func(ctx context.Context, in *webhookIDInput) (*webhookOutput, error) {
    regs, _, err := d.Svc.ListWebhooks(ctx, in.Consumer, in.WebhookID, "", false, 1, 0)
    if err != nil {
        return nil, mapError(ctx, err, "failed to get webhook")
    }
    if len(regs) == 0 {
        return nil, huma.Error404NotFound("webhook not found")
    }
    return &webhookOutput{Body: toWebhookOut(regs[0], nil, d.Svc)}, nil
})
```

## Frontend (Svelte 5)

- Static SPA via `@sveltejs/adapter-static` — no SSR.
- Output dir: `../internal/ui/dist` (embedded in Go binary).
- REST client for API calls (`openapi-fetch`, typed from `web/src/lib/api-types.d.ts`, generated from the OpenAPI spec).
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
- The repo is a Go **workspace** (`go.work`): the CLI, `satellites/recipes`, and `pkg/{signature,template}` are separate modules. Root `./...` only tests the root module — `make test` lists the nested modules explicitly (`MODULE_TEST_PATHS`).

## Release

- Server + `sinks`/`sources` release under `vX.Y.Z`: `git tag vX.Y.Z && git push origin main --tags`.
- The CLI (`satellites/sparrow`), `satellites/recipes`, and `pkg/signature`, `pkg/template` are **separate modules** (see `docs/adr/0002-cli-module-split.md`). They use path-prefixed tags (`pkg/signature/vX.Y.Z`, `satellites/recipes/vX.Y.Z`, `satellites/sparrow/vX.Y.Z`) and their working-tree `replace` lines must be stripped before tagging — `scripts/release-submodules.sh` automates it, full recipe in the ADR.
- GoReleaser config at `.goreleaser.yml`; it builds every binary from the checkout via `replace`, so binary releases don't need the tag dance. Release notes are generated by GoReleaser from Conventional Commits (`feat:`/`fix:`/etc.), grouped in `.goreleaser.yml`'s `changelog:` block — no separate CHANGELOG file to maintain.
- Conventional Commits (`feat:`, `fix:`, etc.) for clean release-note grouping.

## Known Gaps

- No scheduled/delayed webhooks (not in Svix OSS either)
- Limited client SDKs (Python only; generate others from `api/openapi.yaml` on demand)
- **No inbound API rate limiting.** Sparrow does not throttle its own `/v1/*` API — put a reverse proxy in front of it if the API is reachable from untrusted networks. (Outbound *delivery* rate limiting per webhook — `rate_limit_rps` / `webhook_rate_limit_state` — is unrelated and unaffected; it protects receivers from being hammered, not Sparrow's inbound API.)

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `/graphify`, use the installed graphify skill or instructions before doing anything else.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph output, or the user explicitly says not to use it.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
