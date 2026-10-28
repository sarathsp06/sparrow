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
| Portal behind a proxy | `cd deploy/portal && docker compose up -d && ./try.sh` | Private network + Caddy forwarding only `/portal*` and `/_app/*`, like a real portal deployment; `SPARROW_IMAGE=sparrow:dev` tests a local `docker build -t sparrow:dev .` |
| Start dev Postgres | `make dev-db` | Throwaway `postgres:15-alpine` on localhost:5432 matching the default `DATABASE_URL` |
| Run UI dev server | `make run-web` | Hot-reload Svelte at localhost (:5173 default) |
| Run all tests | `make test` | `go test -v ./...` |
| Single package test | `go test -v ./internal/webhooks/...` | |
| Integration tests | `make test-integration` | Needs Docker (testcontainers) |
| Core coverage | `make cover` | Unit + integration tests, merged by `scripts/coverage.sh`; CI fails below 80% of `internal/...` + `pkg/...` (generated `*_otel.go` excluded). Needs `DATABASE_URL` and Docker |
| E2E tests | `make test-e2e` | Gauge + Python, needs Docker |
| SQL benchmark | `make bench-sql SCALE=0.1` | `scripts/sqlbench/run.sh`: fresh DB + `seed.sql` (scale 1 = 6M deliveries, ~9 GB, ~20 min), then `sqlbench_test.go` times the heavy repository methods; `--e2e` adds the pipeline benchmark, `--skip-seed` reuses the DB |
| Lint | `make lint` | golangci-lint, 15m timeout |
| Format | `make fmt` | `goimports -local github.com/sarathsp06/sparrow/ -w .` |
| Run migrations | `make migrate` | Also runs automatically on server startup |
| Generate spec | `make generate` | Exports OpenAPI from Go, then `go generate ./...`. Also regenerates `e2e/client/` (`make client-python`, pinned openapi-python-client): the Python client the e2e suite imports, checked in for the tests only. Users generate their own clients from `api/openapi.yaml` |
| OTel wrapper codegen | `go generate ./...` | Uses gowrap for OTel tracing wrappers |
| Fuzz one target | `cd pkg/template && go test -run '^$' -fuzz '^FuzzExecute$' -fuzztime 1m .` | One `-fuzz` target per run; targets are listed in `.github/workflows/fuzz.yml` (nightly, 5m each). `go test` alone replays the seeds. Commit a crasher from `testdata/fuzz/<Name>/` as a regression seed |

## Architecture

- **Entrypoint**: `server/` (public package) wires chi router, Huma REST API, River queue, OTel, bootstraps default tenant: `server.New` builds, `Start`/`Drain`/`Shutdown` run it, `Handler()` is the full HTTP handler, `Migrate` runs migrations only. `cmd/server/main.go` only adds flags, `.env`, the listener and signal handling, so another Go program can embed Sparrow the same way. Configuration stays env-only: never add a config struct or option for something an env var already covers.
- **REST/OpenAPI**: single HTTP transport on `:8080`. [Huma](https://github.com/danielgtaylor/huma) generates the OpenAPI 3.1 spec from Go handler structs; served interactively at `/docs` (Scalar), spec at `/openapi.yaml`/`/openapi.json`.
- **Queue**: River (Postgres-backed). Default 20 event fan-out workers + 20 webhook delivery workers + 5 default (`SPARROW_EVENT_WORKERS`, `SPARROW_WEBHOOK_WORKERS`); `SPARROW_QUEUE_FETCH_COOLDOWN` (default `20ms`) bounds each queue at `workers / cooldown` jobs/s — River's default `100ms` capped delivery at ~200/s, which is how `cmd/benchmark -mode e2e` found it.
- **Per-tenant fairness (embedders only)**: `server.WithTenantLimits` declares extra queues and a per-tenant `TenantLimits{MaxInFlight, Queue}`. The job inserter routes a tenant's event and delivery jobs to its queue (`jobInserter.tenantQueue`), and both workers check an in-process gate (`queue/tenant_limits.go`) right after parsing the tenant: at the cap they return `river.JobSnooze`, which River does not count as an attempt and which records no health event. Without the option there is no gate and no routing; self-hosted Sparrow never sets it. `internal/integration/tenant_fairness_test.go` covers it.
- **DB**: pgxpool (50 conns, 10 min) for River + sqlx (25 conns) for app queries.
- **Config**: env vars via `kelseyhightower/envconfig` — see `internal/config/config.go`. A value of `""` or `-` means unset (`config.Load` unsets it before parsing, so defaults apply and the OTel exporters agree); never give an empty value its own meaning in a new variable.
- **Logging**: `slog` JSON on stdout, bridged to OTel logs. `SPARROW_LOG_LEVEL` (`debug`/`info`/`warn`/`error`, default `info`) sets the level in `server/server.go` via `Config.SlogLevel`. `info` logs every event and delivery; keep per-request detail at `Debug` so busy instances can run at `warn`.

## Philosophy

Code, docs and reviews in this repository follow the Zen of Python
([PEP 20](https://peps.python.org/pep-0020/)), whatever the language:

```
Beautiful is better than ugly.
Explicit is better than implicit.
Simple is better than complex.
Complex is better than complicated.
Flat is better than nested.
Sparse is better than dense.
Readability counts.
Special cases aren't special enough to break the rules.
Although practicality beats purity.
Errors should never pass silently.
Unless explicitly silenced.
In the face of ambiguity, refuse the temptation to guess.
There should be one-- and preferably only one --obvious way to do it.
Although that way may not be obvious at first unless you're Dutch.
Now is better than never.
Although never is often better than *right* now.
If the implementation is hard to explain, it's a bad idea.
If the implementation is easy to explain, it may be a good idea.
Namespaces are one honking great idea -- let's do more of those!
```

The design principles and conventions below are how it plays out here. A
change that needs a long explanation, a second way to do something that
already has one, or an error swallowed without a comment saying why, is the
signal to stop and simplify. Docs say only what we have run or tested.

## Design Principles

1. **Deterministic bulk operations** — Batch actions snapshot matching IDs into `batch_jobs` at query time. The bulk action operates on that snapshot, NOT a live re-query.
2. **Soft validation over hard rejection** — Schema validation produces warnings, not errors. Events tagged `schema_valid=false`, never discarded.
3. **Fail visibly, never silently** — A step that cannot do what the subscription asked (e.g., a Go template transform that errors) fails the delivery with a permanent, non-retryable error recorded on the delivery row. Fallbacks are opt-in per subscription and still record the error. Faults on the sending side never count against a receiver's health.
4. **Generic infrastructure over per-feature tables** — Shared concerns use generic tables with `job_type` + JSONB `data` columns.
5. **Implicit infrastructure, explicit actions** — Batch jobs are an implementation detail. Users see "re-push ID" and "retry ID", not "batch job IDs".
6. **Postgres-only, no Redis** — All queuing, state, and caching uses PostgreSQL. Don't introduce Redis or other external dependencies without strong justification.
7. **Self-hosted first** — Sparrow targets teams running it behind a VPN for internal webhook delivery, not multi-tenant SaaS.
8. **Principle of least astonishment** — Software and interfaces should behave in a way that least surprises users and other developers. Names, endpoints, config and error messages follow existing conventions in the codebase. When a choice is between a clever behaviour and a predictable one, pick the predictable one and document the rest. A surprise a user has to discover in production is a defect even when the code is correct.

## Key packages

| Path | Purpose |
|------|---------|
| `server/` | Public package: all server wiring (`New`, `Start`, `Handler`, `Shutdown`), embeddable |
| `cmd/server/` | Binary: flags, `.env`, listener, signals around `server` |
| `cmd/migrate/` | Standalone migration runner |
| `internal/rest/` | Huma REST handler layer (thin, calls webhook service) — one file per resource (`webhook.go`, `event.go`, `subscription.go`, `delivery.go`, `health.go`, `ai.go`) |
| `internal/webhooks/` | Business logic + store + queue workers |
| `internal/webhooks/store/` | DB repository (sqlx, WithConn transaction pattern) |
| `internal/webhooks/queue/` | River job types + workers |
| `internal/middleware/` | Auth (master key + access tokens), CORS, portal gateway, security headers |
| `internal/ai/` | AI-assisted transform template drafting behind a `completer` seam: `provider_anthropic.go` (SDK) and `provider_openai.go` (plain HTTP, any OpenAI-compatible server incl. Ollama/vLLM). Off unless `SPARROW_AI_ENABLED=true` (the only switch); the provider vars (`SPARROW_AI_PROVIDER`, `SPARROW_AI_API_KEY`, `SPARROW_AI_BASE_URL`, `SPARROW_AI_MODEL`) are then required, but a missing one never stops startup: `Config.AIConfigError` goes into `rest.AIDeps.ConfigErr`, `GET /v1/capabilities` reports it as `ai_drafting.problem`, and the draft endpoint answers 503 with it. The UI always offers Copy prompt; exposed as `POST /v1/subscriptions:draftTemplate`, `POST /v1/subscriptions:draftTemplatePrompt` (prompt only, works with no provider: the UI's "Copy prompt for AI") + `GET /v1/capabilities` in `internal/rest/ai.go`. Drafts are verified through `TestSubscriptionTemplate` before being returned. The prompt (`prompt.go`) has three parts: a static template reference (system prompt, prefix-cached), the destination recipe (URL, content type, `guidance` on what a good message looks like, params, reference template), then the request (payload field outline, schema, sample, instructions) |
| `internal/accessauth/` | Sparrow adapter for `pkg/access` (realm, scope, prefixes, lifetime rules) |
| `pkg/safetransport/` | Separate Go module, stdlib-only — `New()`/`TLSConfig()` with a `VerifyConnection` hook that refuses known-compromised CA keys (Mozilla bug 2083449), since Go checks no revocation. Every outbound client that verifies TLS uses it; set it on any new one. `NewRevocationChecker` adds intermediate CRL checks (in-memory cache, fetched on first use, refreshed by use past a max age or by `Refresh`), used for deliveries via `client.Config.Revocation` |
| `internal/tlsrevocation/` | Ties the delivery CRL checker to the cluster: `POST /v1/tls/crls:refresh` sends `pg_notify('sparrow_crl_refresh')` and each instance's `Listen` (a hijacked pgx connection) refreshes its own cache. No table, no periodic job. `SPARROW_TLS_REVOCATION` (`soft`/`hard`/`off`), `SPARROW_CRL_MAX_AGE` |
| `pkg/access/` | Separate Go module — reusable token/invite library (memstore, pgstore, storetest, httpauth) |
| `internal/storage/` | DB abstractions, transaction helpers, error sentinels |
| `internal/crypto/` | Envelope encryption (AES-256-GCM) |
| `internal/errors/` | Error categories, service errors, retryability |

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
| `GET /v1/tls/crls`, `POST /v1/tls/crls:refresh` | CRL cache status and refresh (`internal/rest/tls.go`) | Yes | Status is per instance; refresh reaches all instances via NOTIFY, one per minute per server |
| `/v1/tokens`, `/v1/invites`, `/v1/whoami` | Access token/invite endpoints (`internal/rest/access.go`) | Yes | Under `/v1` — require master key or tenant-wide token |
| `/portal/api/*` | Portal gateway (`internal/middleware/portal_gateway.go`) | Portal bearer or consumer token | Re-dispatches to `/v1` scoped to the token's consumer. `SPARROW_PORTAL_RETRY=false` refuses the delivery retry routes and `prepare_retry` (403; `portal.retry` in capabilities, portal disables Retry). Re-push (`events:rePush`, `repush-jobs`, `prepare_repush`) is always refused, like `POST events` |
| `POST /invite/redeem` | Invite redemption (`pkg/access/httpauth`) | No (invite is the credential) | Body `{"invite": "..."}` → token; `400 invalid_invite` for bad/used/expired/cancelled |
| `* (NotFound)` | UI SPA | No | GET/HEAD → HTML; others → JSON 404 |

Route-group middleware (API key auth) wraps only the `/v1/*` group (`r.Group` in `cmd/server/main.go`), not health, docs, or the UI.

## Code conventions

- **Error order**: All `if err != nil { return ... }` before happy path.
- **WithConn**: `repo.WithConn(tx)` inside `storage.WithTransaction()` for repo-level txns.
- **Tenant is the second argument**: every method of every interface in `internal/webhooks/store` is `(ctx context.Context, tenantID uuid.UUID, ...)` and scopes its SQL by that tenant; tables without a `tenant_id` column (deliveries, health, rate limit state) scope through their webhook (`EXISTS (SELECT 1 FROM webhook_registrations wr WHERE wr.id = <table>.webhook_id AND wr.tenant_id = $n)`). A method that genuinely spans every tenant (retention, health evaluation, batch cleanup) carries `//sparrow:no-tenant <reason>` directly above it. `tenant_signature_test.go` enforces both in `go test`.
- **No direct SQL in handlers** — all DB access through RepositoryInterface methods.
- **REST errors**: use `mapError(ctx, err, msg)` from `internal/rest/errors.go`.
- **Tenant scoping**: service code takes the tenant from `tenant.FromContext(ctx)` (the default tenant unless an embedder set one with `server.WithTenant`) and passes it to every storage call; workers use the tenant in their job args. Never read `tenant.DefaultTenantID` in request paths: only bootstrap, `accessauth.Realm()` (Sparrow's own tokens) and the self-hosted state gauge use it. Sparrow never sets a tenant itself, so self-hosted behaviour is one tenant. `internal/integration/multi_tenant_test.go` runs two tenants through the public package and must stay green.
- **Event type writes**: go through `saveEventType` (`internal/webhooks/event_type_save.go`). The only other writer is `store.RegisterEvent` for creation (auto-register, system events), which inserts the version-history row in the same statement. Never update `event_registrations` directly, or the history drifts. Event types are never deleted, so a new name must match `eventNamePattern` (lowercase dotted parts of `a-z0-9_-`, checked only on creation so older names keep working).
- **Templates parse on save**: subscription create/update and webhook create reject a `transform_template` that does not parse (`validateTransformTemplate`, 400). Save never renders; render failures stay delivery-time `template_error`s.
- **Naming**: files `snake_case.go`, packages lowercase single word, REST OperationIDs `camelCase` verb-first (e.g. `registerWebhook`, `listDeliveries`).
- **Requires transform**: `webhook_registrations.requires_transform` (migration 000035) marks a receiver that only accepts a transformed payload; every shipped recipe sets `webhook.requires_transform: true`. Enforced in the service, not the UI, serialized on a row lock of the webhook (`LockWebhook`: exclusive to turn the flag on, shared for subscription writes): `CreateWebhook` needs `transform_template` when events are given (applied to every created subscription), `Create/UpdateSubscription` reject a subscription without an enabled template (`checkRequiredTransform`), PATCH can only turn the flag on once every subscription has one and cannot bulk-replace events while it is on, and the worker fails such a delivery with `template_error` instead of sending the envelope.
- **Delivery health bookkeeping**: the delivery worker writes exactly one transaction per attempt (`DeliveryRepository.RecordDeliveryAttempt`: delivery row incl. `request_body`, plus an append-only `webhook_health_events` row) and never touches webhook-owned rows. `HealthEvaluatorWorker` (River periodic job `health_evaluate`, `SPARROW_HEALTH_EVAL_INTERVAL`, default 1m) calls `EvaluateHealth`, which in one transaction folds events after the watermark (`webhook_health_evaluator`, migration 000037) into per-minute `webhook_health_buckets`, derives `consecutive_failures`/`failing_since` in `webhook_health_state`, rewrites `webhook_registrations.health` only when the label changed, and returns the changes; the worker then emits `health_changed`, and runs the auto-disable check for webhooks in a failure run. The label's 24h success rate is over attempts, not distinct deliveries, and comes from `webhook_metrics.window_attempts/window_failures` (migration 000040): each pass adds its in-window outcomes and subtracts the buckets that slid out of the window, so a pass never re-sums 24h of buckets. The same pass keeps all-time `total_attempts/total_failures` there (000038), which `GetConsumerStats` sums instead of counting deliveries. `webhook_metrics` holds only running counters, one row per webhook; health state (failure runs, timestamps) stays in `webhook_health_state`. Don't reintroduce per-delivery writes to webhook-owned rows: they serialize all workers delivering to one busy webhook (measured: ~440/s cap regardless of worker count).
- **River periodic jobs**: never `UniqueOpts{ByArgs: true}` alone. River's default unique states include `completed` for ~24h, which silently drops every periodic insert after the first run. Either add `ByPeriod: <interval>` (the cleanup/retention/token-purge jobs) or scope `ByState` to the active states (`HealthEvaluateArgs.InsertOpts`).
- **Webhook deletes are soft** (migration 000041): `UnregisterWebhook` sets `deleted_at` and `active=false`; nothing cascades. Every webhook read (lookups, lists, stats, health counts, fan-out, subscription reads) filters `deleted_at IS NULL`; the URL unique index is partial on it. Deliveries, health history and subscriptions of a deleted webhook stay, so delivery lists still show its history. New webhook queries must filter it too.
- **Delivery status binds as the enum**: filters compare `wd.status = $n` with `$n::webhook_delivery_status`, never `status::text` or a text parameter cast to the enum. Either hides the column from `idx_webhook_deliveries_unsuccessful` (the partial index on non-success rows that serves retry snapshots, paused counts and in-flight counts). `webhook_deliveries` keeps only the indexes in migration 000039; each extra one is written on every delivery attempt.
- **Consumers are implicit**: a name on webhooks and events, no table (the `consumers` table was dropped in 000013). `GET /v1/consumers?q=` (`store.ListConsumers`) finds them with a loose index scan; use it for any consumer picker, never a preloaded list (tenants can have hundreds).
- **`_sparrow` is reserved** (`tenant.SystemConsumer`): Sparrow pushes its `sparrow.webhook.*` events there and operators register the alert mailer there. Enforced server-side: `checkEventForConsumer` rejects subscriptions across the line (`sparrow.*` only under `_sparrow`; `_sparrow` only `sparrow.*` or `*`), `GET /v1/event-types` lists `sparrow.*` only with `consumer=_sparrow`, and `accessauth.ValidateConsumer` refuses consumer-scoped tokens for it. Otherwise it is listed and counted like any other consumer.
- **UI consumer scoping**: no global switcher. Each list page has its own `ConsumerPicker` filter backed by the URL (`consumerFilter` in `web/src/lib/consumer.svelte.ts`, `?consumer=`; empty = all). Forms take `?consumer=` or a picked value and never default to `default`.
- **Auto-disable**: `queue.AutoDisablePolicy` (`SPARROW_AUTO_DISABLE_AFTER`/`_MIN_FAILURES`). `webhook_health_state.failing_since` tracks the current failure run (migration 000036); each health evaluation pass (`HealthEvaluatorWorker`) calls `AutoDisableWebhook` for webhooks in a failure run, one UPDATE that pauses the webhook (`active=false`, `auto_disabled_at/_reason`) and emits `sparrow.webhook.disabled`. `UpdateWebhook` with `active=true` clears the marker and restarts the run. `_sparrow` webhooks are exempt.
- **Install script**: `scripts/install.sh` (`curl ... | sh`) resolves the latest tag from the `releases/latest` redirect, downloads the goreleaser archive for the OS/arch, verifies it against `checksums.txt`, and installs to `/usr/local/bin` or `~/.local/bin`. Archive names come from `.goreleaser.yml` (`sparrow-cli-<ver>-<os>-<arch>.tar.gz`): keep them in sync.
- **Pause holds, never drops**: a paused webhook (manual or auto-disabled) or paused subscription gets its fan-out deliveries recorded as `paused`, and the worker turns any queued/retrying delivery for it into `paused` (`HoldDelivery`, no attempt, no health impact). Resume (webhook or subscription) returns `paused_deliveries`; held deliveries are only sent when retried.
- **Template history**: `subscription_template_versions` (migration 000034). `UpdateSubscription` records a version in the same transaction when the template changes, with `TemplateSaveMeta{Source: manual|ai_draft, Notes, SavedBy}`; `CreateSubscription` records the first. No restore endpoint by design: the UI loads a version into the editor and saves normally.
- **Subscription UI**: create/edit are pages (`/webhooks/{id}/subscriptions/new`, `/{sub}/edit`, `SubscriptionForm.svelte`); the template is edited in `TemplateEditor.svelte` (full-viewport modal: live strict render, AI drawer or copy-prompt mode, history, helper reference). `SubscriptionManager.svelte` is the list only.
- **High-volume lists are cursor-paged, without a count**: deliveries and event occurrences return `pagination: {limit, has_more, next_cursor}`; pass `next_cursor` back as `cursor` (keyset on `(created_at, id)`, fetched as `limit + 1`). There is no offset: these tables grow without bound and an offset reads every skipped row (91ms/241ms at 100k deep on 6M rows), so a non-zero `offset` is a 400. Don't add a `total_count` back: on 300k deliveries the count cost more than the page on every load. Filters on another table go in an `EXISTS` behind the `($N IS NULL OR ...)` guard, never a join (pgx caches generic plans, where a join slowed every count 5-20x); cut the page in a subquery first, then join what is displayed. Small lists (webhooks, event types) keep `total_count`. UI: `CursorPages` + `CursorPager` (`web/src/lib/cursor-pages.svelte.ts`).
- **Satellites are libraries**: `satellites/sparrow-sinks` and `satellites/sparrow-sources` are their own modules. The binary stays `package main` at the satellite root; the library mirrors the command in two layers: a core package (`sinks`, `sources`: interfaces, `Handle`, `Verifier`/`Pusher`, envelope) and one package per sink or source (`sinks/{email,otlp,s3}`, `sources/{stripe,github,cron}`), each with its YAML section as `Config` and a `Handler` (or `New`+`Run` for pull/background sources), so every implementation keeps its own dependencies (aws-sdk only in `sinks/s3`). No file/env/flag reading, never a server; the binary composes the packages. A new sink or source is a new package. Listen address, YAML loading and `SPARROW_*` env overrides live in `main.go`. See ADR 0002's 2026-10 amendment.
- **OTel wrappers**: generated via `//go:generate gowrap gen -i InterfaceName ...` — do not hand-edit `*_otel.go` files.
- **Docs diagrams**: write them as ```mermaid blocks in the page (`astro-mermaid`, registered before Starlight in `docs/astro.config.mjs`); they render in the browser, follow the light/dark theme, and open in a zoomable viewer on click (`docs/public/diagram-viewer.js`). Add `accTitle`/`accDescr` lines for screen readers. No pre-rendered SVGs.
- **OpenAPI spec**: exported from Go via `cmd/openapi-export`, committed at `api/openapi.{yaml,json}` — regenerate with `make generate` after any handler change; `internal/rest/openapi_drift_test.go` fails CI if it's stale.

### Repository / Storage pattern

```go
type DBTX interface { GetContext, SelectContext, ExecContext }
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
- **Every tenant-owned table must be erasable by `store.PurgeTenant`** (behind `server.PurgeTenant`): give it a `tenant_id` that cascades from `tenants`, or a `webhook_id` that cascades from `webhook_registrations`, or add an explicit delete there. `internal/integration/tenant_purge_test.go` finds tenant- and webhook-keyed tables from the schema and fails when a new table keeps a purged tenant's rows or is left empty by its setup.

## Test nuances

- Unit tests use `DATABASE_URL` from environment (CI provides a postgres service).
- Integration tests (`-tags integration`) use testcontainers — need Docker.
- E2E tests (Gauge + Python) in `e2e/` — `uv run gauge run specs/`.
- `make fmt` uses `goimports` with local module grouping — install `go install golang.org/x/tools/cmd/goimports@latest`.
- The repo is a Go **workspace** (`go.work`): the CLI, `satellites/{recipes,sparrow-sinks,sparrow-sources}`, `pkg/{signature,template}`, and `pkg/access` are separate modules. Root `./...` only tests the root module — `make test` lists the nested modules explicitly (`MODULE_TEST_PATHS`).

## Release

- A release is one tag push: `git tag vX.Y.Z && git push origin vX.Y.Z`. `.github/workflows/release.yml` runs on every `v*` tag and does the rest: GoReleaser (server, `sinks`, `sources`, CLI binaries and the GitHub release), the split-module tags, then the Docker image. There is no second manual step.
- The CLI (`satellites/sparrow`), `satellites/recipes`, `satellites/sparrow-sinks`, `satellites/sparrow-sources`, `pkg/signature`, `pkg/template`, `pkg/access`, and `pkg/safetransport` are **separate modules** (see `docs/adr/0002-cli-module-split.md`) so `go install` of the CLI doesn't pull the server's dependency tree. They are versioned in lockstep with Sparrow under path-prefixed tags (`pkg/signature/vX.Y.Z`, `satellites/sparrow/vX.Y.Z`, ...), which the workflow's "Release Split Modules" job creates by running `scripts/release-submodules.sh vX.Y.Z --push` (strips each module's `replace` lines on a throwaway commit, never on `main`). Don't run it with `--push` yourself after a release: the tags already exist on the remote. To check what a release tagged, query the remote (`git ls-remote --tags origin 'refs/tags/satellites/*'`) or `git fetch --tags` first; a stale local tag list makes the modules look behind.
- GoReleaser config at `.goreleaser.yml`; it builds every binary from the checkout via `replace`, so binary releases don't need the tag dance. Release notes come from `CHANGELOG.md`: every user-visible change adds a line under `## [Unreleased]`, and before tagging that section is renamed `## [vX.Y.Z] - date` (with a compare link at the bottom). The release workflow publishes the tag's section as the GitHub release note and fails if it is missing; release candidates use `Unreleased`. Write for users deciding whether to upgrade, and put anything they must do under `### Upgrading`.
- Conventional Commits (`feat:`, `fix:`, etc.) for clean release-note grouping.

## Known Gaps

- No published client SDKs, by design: clients are generated from `api/openapi.yaml` (also served at `/openapi.yaml` and attached to each release). Only `pkg/signature` and the `client/verify/` samples are hand-written; `e2e/client/` is generated and checked in only because the e2e suite imports it
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
