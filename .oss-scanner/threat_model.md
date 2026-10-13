# Sparrow threat model

Sparrow is a self-hosted webhook delivery service written in Go. Producers push
events to a REST API (`/v1/*`, Huma, OpenAPI 3.1); Sparrow fans each event out to
the subscriptions of a *consumer*, optionally reshapes the payload with a Go
template, signs it, and POSTs it to the consumer's registered webhook URLs with
retries, pausing, health tracking and auto-disable. PostgreSQL holds all state
and the job queue (River). An embedded Svelte SPA and a consumer-facing portal
are served by the same binary. Read `AGENTS.md` and `okf/index.md` in the scanned
revision for the current architecture; do not assume package names are fixed.

The typical deployment is one organisation running Sparrow behind a VPN, with
several internal teams (consumers) sharing it. It is not designed as public
multi-tenant SaaS, but the API and portal may be reachable from untrusted
networks and the consumer boundary is a real security boundary.

## Principals and trust

- **Operator**: controls environment variables, the keyring
  (`SPARROW_ENCRYPTION_KEYS`), the database and the host. Trusted.
- **Master key holder** (`SPARROW_API_KEY`): full API access. Trusted, but the
  key must not leak (logs, responses, the served UI).
- **Tenant-wide access token**: issued via `/v1/tokens` or invites; acts on all
  consumers. Must be revocable, expire as configured, and never exceed the
  scope it was minted with.
- **Consumer-scoped token / portal bearer**: lower trust. Refused on `/v1` (403);
  only valid through the portal gateway `/portal/api/*`, which re-dispatches to
  `/v1` pinned to that one consumer. **Any way for such a token to read or
  change another consumer's webhooks, subscriptions, events, deliveries,
  secrets or templates, or to reach `/v1` directly, is high or critical.**
- **Unauthenticated network clients**: may reach `/health`, `/ready`, `/metrics`,
  `/docs`, `/openapi.*`, the UI, and `POST /invite/redeem`.
- **Webhook receivers**: untrusted remote servers. Their responses, status
  codes, redirects, headers, timing and TLS behaviour are attacker-controlled.
- **Event producers**: event payloads, event type names, schemas and consumer
  names are untrusted input even from an authenticated caller.
- When `SPARROW_API_KEY` is unset all endpoints are open; that is documented and
  not a finding by itself.

## Where untrusted input enters

- REST handlers in `internal/rest/` and auth/portal/CORS middleware in
  `internal/middleware/` (`auth.go`, `portal_gateway.go`), the token/invite
  library in `pkg/access/` and its adapter `internal/accessauth/`.
- Webhook target URLs: SSRF protection in `internal/webhooks/urlvalidation.go`
  and `internal/webhooks/client/ssrf.go` (loopback, private ranges, cloud
  metadata blocked unless `SPARROW_ALLOW_PRIVATE_NETWORKS` or
  `SPARROW_ALLOWED_NETWORKS` allow them). Check DNS rebinding, redirects, IPv6
  and IPv4-mapped forms, alternate IP encodings, and that the check applies at
  connect time, not only at registration.
- Transform templates (`pkg/template/`, a custom `text/template` FuncMap) are
  authored by API callers, including portal users for their own consumer.
  Template execution must not read files, env vars or secrets, reach the
  network, or exhaust the server (CPU, memory, unbounded output).
- Event type schemas and bundles (`/v1/event-types:import`), JSON payloads,
  cursor tokens, filters and query parameters that reach SQL in
  `internal/webhooks/store/`.
- AI template drafting (`internal/ai/`, off unless configured): payload samples
  are sent to the configured provider; returned drafts are verified before use.
- Satellites (`satellites/sparrow-sources`, `satellites/sparrow-sinks`): inbound
  Stripe/GitHub webhook signature verification, and sinks that forward Sparrow
  deliveries to email/OTLP/S3. Signature bypass or replay is in scope.
- The embedded UI and portal (`web/`): stored XSS via consumer names, event
  payloads, templates, delivery bodies or receiver responses rendered in the UI.

## Security properties

- Consumer isolation through the portal gateway and consumer-scoped tokens; the
  reserved `_sparrow` system consumer cannot be targeted by consumer tokens or
  receive non-`sparrow.*` subscriptions.
- Webhook signing secrets and other sensitive fields are envelope-encrypted
  (`pkg/crypto/`, AES-256-GCM) at rest and never returned in plaintext beyond
  what the API documents. Outbound signatures (`pkg/signature/`) must be
  correct and verifiable; verification helpers must be constant-time.
- Access tokens and invites: single use where documented, expiry, revocation,
  and prefix/realm checks must hold; hashes, not tokens, are stored.
- Deliveries never go to addresses the SSRF policy forbids.
- Soft-deleted webhooks (`deleted_at`) must not receive deliveries or be
  readable as live resources.

## How to exercise it

The checkout is at `/src`. Binaries are in `/usr/local/bin`: `sparrow-server`,
`sparrow-migrate`, `sparrow` (CLI), `sparrow-sinks`, `sparrow-sources`.
PostgreSQL is installed; `DATABASE_URL` and a dev-only all-zeros keyring are
set in the environment. Go module caches are populated and
`GOPROXY=off`/`GOSUMDB=off` keep builds offline.

```sh
service postgresql start
cd /src
make test                              # all workspace modules
go test ./internal/middleware/... ./pkg/access/...
SPARROW_API_KEY=master SPARROW_HTTP_PORT=8080 sparrow-server &   # migrates on start
curl -s -H 'X-API-Key: master' localhost:8080/v1/whoami
cd web && npm run test:unit
```

Use local HTTP servers as webhook receivers; with the default SSRF policy you
need `SPARROW_ALLOWED_NETWORKS=127.0.0.1/32` (or
`SPARROW_ALLOW_PRIVATE_NETWORKS=true`) to deliver to localhost. Integration
(`-tags integration`, testcontainers), Gauge e2e and Playwright tests need
Docker or browsers and are not provisioned.

## Severity

- Critical: unauthenticated access to `/v1` when a key is configured; RCE;
  a consumer-scoped token gaining tenant-wide access; decryption of stored
  secrets without the keyring.
- High: cross-consumer read or write through the portal; SSRF bypass reaching
  loopback, private ranges or cloud metadata under the default policy; SQL
  injection reachable by any authenticated caller; stored XSS in the UI or
  portal triggered by consumer- or receiver-controlled data; token or invite
  reuse/expiry bypass; signature forgery.
- Medium: secret or token disclosure in logs or error messages; template
  execution that exhausts the server; denial of service by one authenticated
  consumer that stalls deliveries for others.
- Low: issues needing the master key or operator access; DoS that requires
  unauthenticated volume (there is no inbound rate limiting by design; it is
  documented in `AGENTS.md` under Known Gaps).

## Out of scope / leave alone

- No inbound rate limiting on `/v1` (documented; operators front it with a proxy).
- Endpoints being open when `SPARROW_API_KEY` is unset.
- Deliveries to private addresses when the operator explicitly allowed them.
- `verify_ssl: false` disabling TLS verification for that webhook (documented).
- `cmd/benchmark`, `scripts/`, `e2e/`, `docs/`, `book/`, generated clients in
  `client/`, and `*_otel.go` generated wrappers (report the generator instead).

## Reports

Include the attacker's starting credential (none, consumer token, tenant token,
master key), the request sequence, and a reproducer that runs against the local
PostgreSQL in this image, ideally as a Go test. Proposed patches should follow
`AGENTS.md` conventions and come with a regression test.
