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
| Run (dev) | `make run` | Shell/`.env` values win; otherwise `scripts/dev-env.sh` defaults `DATABASE_URL` + an all-zeros keyring. Start Postgres first with `make dev-db` |
| Start dev Postgres | `make dev-db` | Throwaway `postgres:15-alpine` on localhost:5432 matching the default `DATABASE_URL` |
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
3. **Fail visibly, never silently** — A step that cannot do what the subscription asked (e.g., a Go template transform that errors) fails the delivery with a permanent, non-retryable error recorded on the delivery row. Fallbacks are opt-in per subscription and still record the error. Faults on the sending side never count against a receiver's health.
4. **Generic infrastructure over per-feature tables** — Shared concerns use generic tables with `job_type` + JSONB `data` columns.
5. **Implicit infrastructure, explicit actions** — Batch jobs are an implementation detail. Users see "re-push ID" and "retry ID", not "batch job IDs".
6. **Postgres-only, no Redis** — All queuing, state, and caching uses PostgreSQL. Don't introduce Redis or other external dependencies without strong justification.
7. **Self-hosted first** — Sparrow targets teams running it behind a VPN for internal webhook delivery, not multi-tenant SaaS.

## Key packages

| Path | Purpose |
|------|---------|
| `cmd/server/` | Server entrypoint + DI wiring |
| `cmd/migrate/` | Standalone migration runner |
| `internal/rest/` | Huma REST handler layer (thin, calls webhook service) — one file per resource (`webhook.go`, `event.go`, `subscription.go`, `delivery.go`, `health.go`, `ai.go`) |
| `internal/webhooks/` | Business logic + store + queue workers |
| `internal/webhooks/store/` | DB repository (sqlx, WithConn transaction pattern) |
| `internal/webhooks/queue/` | River job types + workers |
| `internal/middleware/` | Auth (master key + access tokens), CORS, portal gateway, security headers |
| `internal/ai/` | AI-assisted transform template drafting behind a `completer` seam: `provider_anthropic.go` (SDK) and `provider_openai.go` (plain HTTP, any OpenAI-compatible server incl. Ollama/vLLM). Off unless configured (`SPARROW_AI_PROVIDER`, `SPARROW_AI_API_KEY`, `SPARROW_AI_BASE_URL`); exposed as `POST /v1/subscriptions:draftTemplate`, `POST /v1/subscriptions:draftTemplatePrompt` (prompt only, works with no provider: the UI's "Copy prompt for AI") + `GET /v1/capabilities` in `internal/rest/ai.go`. Drafts are verified through `TestSubscriptionTemplate` before being returned |
| `internal/accessauth/` | Sparrow adapter for `pkg/access` (realm, scope, prefixes, lifetime rules) |
| `pkg/access/` | Separate Go module — reusable token/invite library (memstore, pgstore, storetest, httpauth) |
| `pkg/storage/` | DB abstractions, transaction helpers, error sentinels |
| `pkg/crypto/` | Envelope encryption (AES-256-GCM) |
| `pkg/errors/` | Error categories, service errors, retryability |

## Authentication

Optional auth via `SPARROW_API_KEY` env var. When set, every `/v1/*` request must include either the master key or a tenant-wide access token via `X-API-Key` or `Authorization: Bearer`. Consumer-scoped tokens are refused on `/v1` (403); they only work through the portal gateway `/portal/api/*`, pinned to their consumer. When `SPARROW_API_KEY` is unset, all endpoints are open. Excluded paths: `/health`, `/ready`, `/metrics`, `/docs`, `/openapi`, UI catch-all. Implementation: `internal/middleware/auth.go` (replaced `apikey.go`).

The embedded UI (`SPARROW_SERVE_UI=true`) is served exactly as built -- the server never writes `SPARROW_API_KEY` into the page. When `SPARROW_API_KEY` is set, the UI shows a sign-in prompt on the first 401: paste the master key (exchanged for a named browser token, never stored) or use an access token or invite link. A separately hosted UI reads `apiUrl`/`apiKey` from its static `/config.js` (`web/static/config.js`) or, if the server requires a key and none is configured, prompts for a credential on the first 401 and stores it in `localStorage`.

## HTTP Routing (chi)

| Pattern | Handler | Auth | Notes |
|---------|---------|------|-------|
| `/v1/*` | Huma REST API | Yes | See `internal/rest/` — one file per resource |
| `/docs`, `/openapi.*` | Huma-served Scalar UI + spec | No | Interactive API reference |
| `GET /health`, `/ready` | Health check | No | JSON status |
| `GET /metrics` | Prometheus scrape of all OTel metrics (`observability.MetricsHandler`) | No | Off with `SPARROW_METRICS_ENABLED=false` |
| `/v1/event-types:export`, `/v1/event-types:import` | Event type bundles (`internal/rest/event_bundle.go`) | Yes | Move definitions between environments; import is all-or-nothing with dry run. There is no event type delete |
| `/v1/consumers/{c}/subscriptions/{id}/templateVersions` | Saved template history (`internal/rest/subscription.go`) | Yes | Newest = current; last 20 kept; written when a save changes `transform_template` |
| `/v1/tokens`, `/v1/invites`, `/v1/whoami` | Access token/invite endpoints (`internal/rest/access.go`) | Yes | Under `/v1` — require master key or tenant-wide token |
| `/portal/api/*` | Portal gateway (`internal/middleware/portal_gateway.go`) | Portal bearer or consumer token | Re-dispatches to `/v1` scoped to the token's consumer |
| `POST /invite/redeem` | Invite redemption (`pkg/access/httpauth`) | No (invite is the credential) | Body `{"invite": "..."}` → token; `400 invalid_invite` for bad/used/expired/cancelled |
| `* (NotFound)` | UI SPA | No | GET/HEAD → HTML; others → JSON 404 |

Route-group middleware (API key auth) wraps only the `/v1/*` group (`r.Group` in `cmd/server/main.go`), not health, docs, or the UI.

## Code conventions

- **Error order**: All `if err != nil { return ... }` before happy path.
- **WithConn**: `repo.WithConn(tx)` inside `storage.WithTransaction()` for repo-level txns.
- **No direct SQL in handlers** — all DB access through RepositoryInterface methods.
- **REST errors**: use `mapError(ctx, err, msg)` from `internal/rest/errors.go`.
- **Tenant scoping**: always filter by `tenant.DefaultTenantID` in queries.
- **Event type writes**: go through `saveEventType` (`internal/webhooks/event_type_save.go`). The only other writer is `store.RegisterEvent` for creation (auto-register, system events), which inserts the version-history row in the same statement. Never update `event_registrations` directly, or the history drifts. Event types are never deleted.
- **Naming**: files `snake_case.go`, packages lowercase single word, REST OperationIDs `camelCase` verb-first (e.g. `registerWebhook`, `listDeliveries`).
- **Requires transform**: `webhook_registrations.requires_transform` (migration 000035) marks a receiver that only accepts a transformed payload; every shipped recipe sets `webhook.requires_transform: true`. Enforced in the service, not the UI, serialized on a row lock of the webhook (`LockWebhook`: exclusive to turn the flag on, shared for subscription writes): `CreateWebhook` needs `transform_template` when events are given (applied to every created subscription), `Create/UpdateSubscription` reject a subscription without an enabled template (`checkRequiredTransform`), PATCH can only turn the flag on once every subscription has one and cannot bulk-replace events while it is on, and the worker fails such a delivery with `template_error` instead of sending the envelope.
- **Auto-disable**: `queue.AutoDisablePolicy` (`SPARROW_AUTO_DISABLE_AFTER`/`_MIN_FAILURES`). `webhook_health_state.failing_since` tracks the current failure run (migration 000036); after each failed attempt the worker calls `AutoDisableWebhook`, one UPDATE that pauses the webhook (`active=false`, `auto_disabled_at/_reason`) and emits `sparrow.webhook.disabled`. `UpdateWebhook` with `active=true` clears the marker and restarts the run. `_sparrow` webhooks are exempt.
- **Pause holds, never drops**: a paused webhook (manual or auto-disabled) or paused subscription gets its fan-out deliveries recorded as `paused`, and the worker turns any queued/retrying delivery for it into `paused` (`HoldDelivery`, no attempt, no health impact). Resume (webhook or subscription) returns `paused_deliveries`; held deliveries are only sent when retried.
- **Template history**: `subscription_template_versions` (migration 000034). `UpdateSubscription` records a version in the same transaction when the template changes, with `TemplateSaveMeta{Source: manual|ai_draft, Notes, SavedBy}`; `CreateSubscription` records the first. No restore endpoint by design: the UI loads a version into the editor and saves normally.
- **Subscription UI**: create/edit are pages (`/webhooks/{id}/subscriptions/new`, `/{sub}/edit`, `SubscriptionForm.svelte`); the template is edited in `TemplateEditor.svelte` (full-viewport modal: live strict render, AI drawer or copy-prompt mode, history, helper reference). `SubscriptionManager.svelte` is the list only.
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
- The repo is a Go **workspace** (`go.work`): the CLI, `satellites/recipes`, `pkg/{signature,template}`, and `pkg/access` are separate modules. Root `./...` only tests the root module — `make test` lists the nested modules explicitly (`MODULE_TEST_PATHS`).

## Release

- Server + `sinks`/`sources` release under `vX.Y.Z`: `git tag vX.Y.Z && git push origin main --tags`.
- The CLI (`satellites/sparrow`), `satellites/recipes`, `pkg/signature`, `pkg/template`, and `pkg/access` are **separate modules** (see `docs/adr/0002-cli-module-split.md`). They use path-prefixed tags (`pkg/signature/vX.Y.Z`, `pkg/access/vX.Y.Z`, `satellites/recipes/vX.Y.Z`, `satellites/sparrow/vX.Y.Z`) and their working-tree `replace` lines must be stripped before tagging — `scripts/release-submodules.sh` automates it, full recipe in the ADR.
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
