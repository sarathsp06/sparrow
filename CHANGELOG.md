# Changelog

What changed in each Sparrow release, written for people deciding whether to
upgrade. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and versions follow [Semantic Versioning](https://semver.org/). What counts as a
breaking change is defined in the
[compatibility policy](https://sarathsp06.github.io/sparrow/reference/compatibility/).

From v1.0.0 on, each release's section is also its release note on
[GitHub Releases](https://github.com/sarathsp06/sparrow/releases). The full
commit list for a release is linked from its heading.

**Upgrading** lists what you must do when moving to that version. Releases
without it upgrade in place: replace the binary or image and restart; database
migrations run on startup.

## [Unreleased]

### Added

- Every delivery carries `X-Sparrow-Event-Type`, `X-Sparrow-Event-Version` and `X-Sparrow-Subscription-ID`, so a receiver can route on headers even when a transform reshapes the body. Custom headers can't override them.
- `POST /v1/consumers/{consumer}/subscriptions/{subscription_id}:sendTestEvent` sends `event_name` (the subscription's event, or any event for a catch-all subscription) to the subscription: one signed delivery to its webhook of the event type's sample or an optional `payload`, rendered through the subscription's transform. It returns the receiver's status, latency and error class. It uses the real delivery path (signing, SSRF protection, timeouts), adds `X-Sparrow-Test: true`, records no delivery and never affects health or auto-disable. Limited to one per webhook every 2 seconds on each server. Portal tokens can call it for their own webhooks.
- A **Test** tab on the webhook page and in the consumer portal sends a test event per subscription, with the sample or a payload you edit, and shows the receiver's status, latency and error class.

### Changed

- The quickstart and the Register webhook page now point to [Testhooks](https://testhooks.vercel.app/), an open-source webhook inspector, for watching test deliveries.

### Fixed

- A pushed event is stored and queued for delivery in one transaction. A crash in between could leave an event stored but never delivered, and a retry with the same idempotency key was then answered as a duplicate. Batch re-push has the same fix.
- Retrying a delivery that is still in flight (pending or retrying) no longer queues a second job and delivers it twice. Retry-all now covers only failed and paused deliveries, and a delivery's reset and its new job commit together.
- A delivery whose subscription was deleted is now failed instead of being sent without its transform, headers or pause. A transient error reading the subscription is retried.
- Waiting for a template execution slot under load no longer counts toward the 5-second render timeout. Valid templates could fail permanently with "possible infinite loop" when many deliveries rendered at once.
- A `Retry-After` header with a huge number of seconds (e.g. `18446744074`) no longer overflows into a sub-second retry; it is capped at 15 minutes like any value over the limit.

### Security

- Webhook destinations are now also refused when they are IPv6 addresses carrying a restricted IPv4 address in IPv4-compatible (`::/96`), IPv4-translated (`::ffff:0:0:0/96`) or Teredo (`2001::/32`) form, and in the reserved ranges `100::/64`, `2001:2::/48`, `2001:db8::/32` and `fec0::/10`. A stock Linux host cannot route these forms to the embedded address, so this is hardening rather than a fix for a reachable bypass.
- The server is built with `google.golang.org/grpc` v1.84.0 (fixes a server panic on requests without an authority or `Host` header, GO-2026-6443). The OTLP gRPC exporters use it.
- The Docker image's base images are pinned by digest, so a rebuilt release tag uses the same bases.

## [v1.0.1] - 2026-10-17

### Added

- The server listens on `PORT` when `SPARROW_HTTP_PORT` is not set.
- `SPARROW_QUEUE_DB_MAX_CONNS` (default `50`) and `SPARROW_DB_MAX_CONNS` (default `25`) size the Postgres connection pools, which were fixed.

## [v1.0.0] - 2026-10-16

Sparrow 1.0 is the first release covered by the
[compatibility policy](https://sarathsp06.github.io/sparrow/reference/compatibility/):
the `/v1` REST API, environment variables, CLI flags, webhook envelope and
signatures, template functions, migrations and Go libraries stay compatible
across 1.x releases. The project also meets the
[OpenSSF Best Practices](https://www.bestpractices.dev/projects/15538) passing
criteria.

### Upgrading

- From v0.14.0 this is an ordinary deploy: replace the binary or image and
  restart. From an older 0.x release, read the
  [upgrading to 1.0 guide](https://sarathsp06.github.io/sparrow/deployment/upgrading-to-1-0/)
  and the Upgrading notes of every release you skip below.
- Pin the image to `ghcr.io/sarathsp06/sparrow:1.0.0` rather than `latest`.
- Saving a subscription or webhook with a `transform_template` that does not
  parse now fails with a 400. Stored templates are not re-checked until you
  next save them.
- New event type names must be lowercase dotted parts (`a-z`, `0-9`, `_`,
  `-`). Existing names keep working.

### Added

- `testTemplate` can render a template against a payload you pass in, not only
  the event type's sample.
- The subscription form shows what each event delivers before you subscribe.
- `sparrow mcp` gains event type tools.
- Docs map common tasks across the UI, CLI and MCP server.
- GitHub private vulnerability reporting, alongside email (see
  [SECURITY.md](https://github.com/sarathsp06/sparrow/blob/main/SECURITY.md)).

### Fixed

- A `transform_template` that does not parse is rejected when saved instead of
  failing every delivery later.

### Security

- CI runs gosec and fails on govulncheck findings; Dependabot keeps Go, npm,
  GitHub Actions and Docker dependencies current.
- The documentation site moved to Astro 7, clearing its critical and high npm
  advisories.
- Release binaries are reproducible: built with `-trimpath` and file times set
  from the commit, so rebuilding a tag gives byte-identical binaries.

## [v0.14.0] - 2026-10-15

The 1.0 readiness release: it settles the API and configuration contract and
fixes several correctness issues in the delivery queue.

### Upgrading

- Read the [upgrading to 1.0 guide](https://sarathsp06.github.io/sparrow/deployment/upgrading-to-1-0/)
  before upgrading; it lists every renamed field and environment variable.
- Draft with AI is now enabled only by `SPARROW_AI_ENABLED=true`. Setting a
  provider key alone no longer turns it on.
- An empty value or `-` for any `SPARROW_*` variable now means "unset", so the
  default applies.
- The server refuses to start in production with the all-zeros encryption key.
  Generate a real keyring before upgrading.
- Go library users: `pkg/crypto`, `pkg/storage` and `pkg/errors` moved to
  `internal/` and are no longer importable; `pkg/template` names lost their
  package-name stutter.

### Added

- `--version` on `sparrow-server`, `sparrow-sinks` and `sparrow-sources`.
- `sparrow-server --migrate` applies migrations and exits, for running them as a
  separate deploy step.
- CLI: `sparrow recipes show` and `validate`; richer `sparrow mcp` tools.
- A compatibility policy defining what 1.0 promises to keep stable.

### Fixed

- Event fan-out is idempotent and atomic: a retried fan-out job no longer
  creates duplicate deliveries.
- Delivery statuses stay accurate when a transient error interrupts an attempt.
- Long deliveries and batch jobs are no longer cut off by the job timeout.
- Delivery attempts and batch jobs are scoped to their consumer in the API.
- Graceful shutdown is bounded, and `/health` (liveness) is separate from
  `/ready` (readiness).
- Templates are limited in the work they can do, not only in output size.

## [v0.13.0] - 2026-10-14

### Added

- `sparrow-sinks` and `sparrow-sources` can be imported as Go libraries.
- One delivery table across the UI, with webhook names and attempt history.
- `SECURITY.md` with the vulnerability reporting contact.

### Fixed

- The server binary in release archives is named `sparrow-server`.
- A top-level `rate_limit_rps` on a webhook is honoured.
- `registerWebhook` returns the stored webhook.
- `max_retries: 0` means exactly one delivery attempt.
- The UI never stores the master key when the token exchange fails, and the API
  console's copy-as-curl escapes single quotes.
- `cmd/migrate` no longer logs `DATABASE_URL`, which contains the password.
- Templates reject `range` over integer literals and deep `range` nesting.

### Changed

- Built with Go 1.27.2. Removed the unused `sparrow_active_webhooks` metric and
  a number of unused internal helpers.

## [v0.12.1] - 2026-10-11

### Added

- `sparrow mcp`: an MCP server so AI agents can inspect webhooks, deliveries and
  recipes, with paging and filters on `list_deliveries`.

## [v0.12.0] - 2026-10-10

Queries no longer scan unbounded history, so large installations stay fast.

### Upgrading

- Webhook deletes are now soft: deleted webhooks keep their delivery history.
- Delivery health and statistics come from running counters kept by the
  health evaluation job instead of queries over all past deliveries.

### Added

- `SPARROW_LOG_LEVEL` (`debug`, `info`, `warn`, `error`).
- `make bench-sql`: a reproducible SQL benchmark on generated data.

## [v0.11.0] - 2026-10-09

### Upgrading

- Delivery and event lists are cursor-paged: pass `next_cursor` back as
  `cursor`. `offset` is no longer accepted and the responses no longer include
  `total_count`.

### Added

- Event filter on delivery lists and inline rename in the UI.

### Fixed

- Rate-limit waits are measured on the database clock, so clock skew between
  instances no longer distorts them.

## [v0.10.3] - 2026-10-09

### Added

- Choose HMAC or HMAC + Ed25519 signing per webhook in the UI, with a link to
  the signature verification guide.

### Fixed

- AI template drafts have their own timeout instead of the 30-second write
  timeout.

## [v0.10.2] - 2026-10-09

### Changed

- AI template drafting uses a three-part prompt with per-destination guidance,
  which produces better drafts.

## [v0.10.1] - 2026-10-09

### Fixed

- AI drafting works with OpenAI-compatible servers that reject `temperature`.

## [v0.10.0] - 2026-10-08

### Changed

- Sparrow's own alert events (`sparrow.*`) belong to the reserved `_sparrow`
  consumer, and list pages filter by consumer.

## [v0.9.0] - 2026-10-07

A performance release: delivery throughput is no longer capped by per-webhook
row contention.

### Added

- Configurable worker pools and queue fetch cooldown (`SPARROW_EVENT_WORKERS`,
  `SPARROW_WEBHOOK_WORKERS`, `SPARROW_QUEUE_FETCH_COOLDOWN`).
- A `curl | sh` install script for the CLI that also updates in place.
- End-to-end benchmark modes in `cmd/benchmark`.

### Changed

- Webhook health is derived by a periodic job instead of on every delivery.

### Fixed

- Periodic cleanup jobs run every interval, not once a day.
- Delivery connections are kept alive across requests.

## [v0.8.2] - 2026-10-06

### Added

- Choose which signals (traces, metrics, logs) are exported over OTLP.

## [v0.8.1] - 2026-10-05

### Changed

- Multi-architecture Docker images cross-compile instead of building under
  emulation, cutting the image build from about 20 minutes. The images
  themselves are unchanged.

## [v0.8.0] - 2026-10-05

### Changed

- The recipe workbench takes the transform template as its input.

## [v0.7.1] - 2026-10-05

### Added

- Infer an event type's schema from events that were already pushed.

## [v0.7.0] - 2026-10-04

### Added

- Failing webhooks are disabled automatically (`SPARROW_AUTO_DISABLE_AFTER`).
- Paused webhooks hold their deliveries instead of dropping them; resuming
  reports how many are waiting.
- Prometheus metrics at `/metrics`.

## [v0.6.2] - 2026-10-02

### Upgrading

- Event type definitions are versioned and can no longer be deleted.
  Incompatible schema changes need an explicit opt-in.
- A template that fails to render now fails the delivery visibly instead of
  falling back silently; fallback is a per-subscription opt-in.

### Added

- AI-drafted transform templates, a full-screen template editor and saved
  template versions.
- Webhooks can require a payload transform.
- Pause and resume a single subscription.
- Export and import event type bundles (API, CLI and UI).
- OTLP export over gRPC via `OTEL_EXPORTER_OTLP_PROTOCOL`.

### Fixed

- Abandoned bulk-action snapshots are purged hourly.

## [v0.6.1] - 2026-09-28

### Upgrading

- Access tokens: the `idempotency_key` field is renamed `external_id`.

## [v0.6.0] - 2026-09-28

### Upgrading

- Consumer portal links are now revocable consumer-scoped access tokens:
  create one with `POST /v1/tokens` (the response includes `portal_path`) and
  revoke it with `DELETE /v1/tokens/{id}`. Portal links issued by earlier
  versions stop working; issue new ones.
- The server never injects `SPARROW_API_KEY` into the UI. Users sign in when
  prompted instead.

### Added

- Signature verification samples in nine languages; access tokens expire by
  default.

## [v0.5.17] - 2026-09-27

### Fixed

- Safer webhook config updates, an explicit TLS verification opt-out, and
  secrets redacted from logged URLs.

## [v0.5.16] - 2026-09-27

### Added

- Per-person access tokens and invite links (API, CLI and UI), including
  one-time links that sign a browser into the UI.
- `pkg/access`, a reusable Go library for tokens and invites.

## [v0.5.15] - 2026-09-26

### Added

- The UI can be hosted separately from the server.

### Fixed

- Cross-origin requests are blocked in production unless
  `CORS_ALLOWED_ORIGINS` allows them.
- `make run` and `make migrate` work without extra setup.

## [v0.5.14] - 2026-09-25

### Added

- Wider editor pages with full-screen editing in the UI.

### Fixed

- `_sparrow` health and delivery-failure events are always pushed.

## [v0.5.13] - 2026-09-23

Documentation site changes only.

## [v0.5.12] - 2026-09-23

### Upgrading

- The SendGrid alert bootstrap environment variables are removed. Register an
  alert webhook under the `_sparrow` consumer instead (see the alerts docs).

## [v0.5.11] - 2026-09-22

### Added

- The `sparrow` CLI ships with the built-in recipes.

### Fixed

- Smaller Docker image.

## [v0.5.10] - 2026-09-21

### Added

- The Recipe Workbench for building and testing recipe templates, including an
  email composer.

### Fixed

- Missing system-event alert subscriptions are recreated at startup.

## [v0.5.9] - 2026-09-18

Documentation changes only.

## [v0.5.8] - 2026-09-18

### Fixed

- Simpler editing of secret headers.

## [v0.5.7] - 2026-09-18

### Upgrading

- The encryption keyring must be configured; the server no longer starts
  without it.

### Added

- Recipes are served by the API, and portal setup supports branding.

### Changed

- Railway and Helm deployment files are removed; use Docker Compose or the
  container image.

## [v0.5.3] – [v0.5.6] - 2026-09-16 to 2026-09-17

Superseded by v0.5.7, which contains the same changes: the consumer portal API
moved behind `/portal/api`, recipes served from the API, and a required
encryption keyring.

## [v0.5.2] - 2026-09-16

Documentation changes only.

## [v0.5.1] - 2026-09-16

### Added

- The consumer portal: consumers manage their own webhooks with scoped,
  expiring access tokens.
- Start registering a webhook from a shipped recipe in the UI.

## [v0.5.0] - 2026-09-16

### Added

- Event retention via `SPARROW_EVENT_RETENTION_DAYS`.

## [v0.4.1] - 2026-09-16

### Added

- An API console drawer in the UI with copy-as-curl.
- The Ed25519 signing public key is shown on the webhook page.
- An optional `consumer` filter on global list endpoints.
- Push Test Event payloads are validated against the schema.

### Fixed

- Every delivery attempt is counted, not only final outcomes.
- The event edit pages no longer crash under the Content Security Policy.

### Removed

- The pre-envelope-encryption decryption fallback. Envelope encryption is the
  only supported secret format.

## [v0.4.0] - 2026-09-15

### Added

- Sparrow emits its own health and failure events, with optional email alerts.

## [v0.3.0] - 2026-09-14

### Upgrading

- "Namespace" is renamed "consumer" throughout the API, CLI and UI.
- `sparrow-sources`: `event_prefix` is renamed `sparrow_event_prefix`, and
  each provider's webhook path is configurable (default
  `/webhooks/stripe` and `/webhooks/github`).

### Added

- `error_category` filter on delivery lists.
- A Twilio recipe and encrypted secret headers.
- The CLI is a separate Go module built with cobra.

## [v0.2.5] - 2026-09-13

### Added

- Satellites: the `sparrow` CLI, destination recipes, `sparrow-sources` and
  `sparrow-sinks` (including an OTLP log sink).

### Fixed

- In-flight retries respect a paused webhook.

## [v0.2.4] - 2026-09-12

### Fixed

- Subscriptions without label filters no longer store `null`.

## [v0.2.3] - 2026-09-12

### Fixed

- Per-webhook `retry_backoff_seconds` is honoured.

## [v0.2.2] - 2026-09-11

### Changed

- The web UI uses a light theme.

## [v0.2.1] - 2026-09-10

### Added

- The Ed25519 public key is returned on webhook registration.

### Fixed

- Delivery statistics for events with no deliveries.

## [v0.2.0] - 2026-08-30

### Upgrading

- The Connect-RPC/gRPC API is replaced by a REST API with an OpenAPI 3.1 spec.
  Clients must move to the REST endpoints under `/v1`.

### Fixed

- Delivery fails closed when a webhook secret cannot be decrypted.
- Internal errors are no longer leaked in API responses.

## [v0.1.14] - 2026-06-23

Internal refactoring only.

## [v0.1.13] - 2026-06-22

### Added

- Dual-signature webhooks with automatically generated signing secrets.

## [v0.1.12] - 2026-06-01

Documentation and test changes only.

## [v0.1.11] - 2026-05-04

Documentation site changes only.

## [v0.1.10] - 2026-05-01

### Added

- Ed25519 asymmetric webhook signing alongside HMAC.

## [v0.1.8] – [v0.1.9] - 2026-04-13 to 2026-04-28

Documentation changes only.

## [v0.1.7] - 2026-04-12

### Added

- Per-webhook rate limiting; HTTP 429 responses are retried.

## [v0.1.5] – [v0.1.6] - 2026-04-10

Documentation and refactoring only.

## [v0.1.4] - 2026-04-10

### Upgrading

- `SPARROW_ENCRYPTION_KEY` is required; the server no longer generates a
  temporary key that was lost on restart.

## [v0.1.3] - 2026-04-09

### Added

- Idempotency keys on pushed events to detect duplicates.

## [v0.1.2] - 2026-04-09

### Added

- An event detail page.

### Fixed

- Retried deliveries get a fresh 24-hour expiry.

## [v0.1.1] - 2026-04-08

### Fixed

- The encryption key is no longer stored in the database.
- Masked secrets are no longer saved back on webhook update.
- Health metrics cover the full 24-hour window.

## [v0.1.0] - 2026-04-07

First release: webhook registration and subscriptions, signed delivery with
retries, payload transforms, soft schema validation, bulk re-push and retry,
and the embedded web UI.

[Unreleased]: https://github.com/sarathsp06/sparrow/compare/v1.0.1...HEAD
[v1.0.1]: https://github.com/sarathsp06/sparrow/compare/v1.0.0...v1.0.1
[v1.0.0]: https://github.com/sarathsp06/sparrow/compare/v0.14.0...v1.0.0
[v0.14.0]: https://github.com/sarathsp06/sparrow/compare/v0.13.0...v0.14.0
[v0.13.0]: https://github.com/sarathsp06/sparrow/compare/v0.12.1...v0.13.0
[v0.12.1]: https://github.com/sarathsp06/sparrow/compare/v0.12.0...v0.12.1
[v0.12.0]: https://github.com/sarathsp06/sparrow/compare/v0.11.0...v0.12.0
[v0.11.0]: https://github.com/sarathsp06/sparrow/compare/v0.10.3...v0.11.0
[v0.10.3]: https://github.com/sarathsp06/sparrow/compare/v0.10.2...v0.10.3
[v0.10.2]: https://github.com/sarathsp06/sparrow/compare/v0.10.1...v0.10.2
[v0.10.1]: https://github.com/sarathsp06/sparrow/compare/v0.10.0...v0.10.1
[v0.10.0]: https://github.com/sarathsp06/sparrow/compare/v0.9.0...v0.10.0
[v0.9.0]: https://github.com/sarathsp06/sparrow/compare/v0.8.2...v0.9.0
[v0.8.2]: https://github.com/sarathsp06/sparrow/compare/v0.8.1...v0.8.2
[v0.8.1]: https://github.com/sarathsp06/sparrow/compare/v0.8.0...v0.8.1
[v0.8.0]: https://github.com/sarathsp06/sparrow/compare/v0.7.1...v0.8.0
[v0.7.1]: https://github.com/sarathsp06/sparrow/compare/v0.7.0...v0.7.1
[v0.7.0]: https://github.com/sarathsp06/sparrow/compare/v0.6.2...v0.7.0
[v0.6.2]: https://github.com/sarathsp06/sparrow/compare/v0.6.1...v0.6.2
[v0.6.1]: https://github.com/sarathsp06/sparrow/compare/v0.6.0...v0.6.1
[v0.6.0]: https://github.com/sarathsp06/sparrow/compare/v0.5.17...v0.6.0
[v0.5.17]: https://github.com/sarathsp06/sparrow/compare/v0.5.16...v0.5.17
[v0.5.16]: https://github.com/sarathsp06/sparrow/compare/v0.5.15...v0.5.16
[v0.5.15]: https://github.com/sarathsp06/sparrow/compare/v0.5.14...v0.5.15
[v0.5.14]: https://github.com/sarathsp06/sparrow/compare/v0.5.13...v0.5.14
[v0.5.13]: https://github.com/sarathsp06/sparrow/compare/v0.5.12...v0.5.13
[v0.5.12]: https://github.com/sarathsp06/sparrow/compare/v0.5.11...v0.5.12
[v0.5.11]: https://github.com/sarathsp06/sparrow/compare/v0.5.10...v0.5.11
[v0.5.10]: https://github.com/sarathsp06/sparrow/compare/v0.5.9...v0.5.10
[v0.5.9]: https://github.com/sarathsp06/sparrow/compare/v0.5.8...v0.5.9
[v0.5.8]: https://github.com/sarathsp06/sparrow/compare/v0.5.7...v0.5.8
[v0.5.7]: https://github.com/sarathsp06/sparrow/compare/v0.5.2...v0.5.7
[v0.5.3]: https://github.com/sarathsp06/sparrow/compare/v0.5.2...v0.5.3
[v0.5.6]: https://github.com/sarathsp06/sparrow/compare/v0.5.2...v0.5.6
[v0.5.2]: https://github.com/sarathsp06/sparrow/compare/v0.5.1...v0.5.2
[v0.5.1]: https://github.com/sarathsp06/sparrow/compare/v0.5.0...v0.5.1
[v0.5.0]: https://github.com/sarathsp06/sparrow/compare/v0.4.1...v0.5.0
[v0.4.1]: https://github.com/sarathsp06/sparrow/compare/v0.4.0...v0.4.1
[v0.4.0]: https://github.com/sarathsp06/sparrow/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/sarathsp06/sparrow/compare/v0.2.5...v0.3.0
[v0.2.5]: https://github.com/sarathsp06/sparrow/compare/v0.2.4...v0.2.5
[v0.2.4]: https://github.com/sarathsp06/sparrow/compare/v0.2.3...v0.2.4
[v0.2.3]: https://github.com/sarathsp06/sparrow/compare/v0.2.2...v0.2.3
[v0.2.2]: https://github.com/sarathsp06/sparrow/compare/v0.2.1...v0.2.2
[v0.2.1]: https://github.com/sarathsp06/sparrow/compare/v0.2.0...v0.2.1
[v0.2.0]: https://github.com/sarathsp06/sparrow/compare/v0.1.14...v0.2.0
[v0.1.14]: https://github.com/sarathsp06/sparrow/compare/v0.1.13...v0.1.14
[v0.1.13]: https://github.com/sarathsp06/sparrow/compare/v0.1.12...v0.1.13
[v0.1.12]: https://github.com/sarathsp06/sparrow/compare/v0.1.11...v0.1.12
[v0.1.11]: https://github.com/sarathsp06/sparrow/compare/v0.1.10...v0.1.11
[v0.1.10]: https://github.com/sarathsp06/sparrow/compare/v0.1.9...v0.1.10
[v0.1.8]: https://github.com/sarathsp06/sparrow/compare/v0.1.7...v0.1.8
[v0.1.9]: https://github.com/sarathsp06/sparrow/compare/v0.1.7...v0.1.9
[v0.1.7]: https://github.com/sarathsp06/sparrow/compare/v0.1.6...v0.1.7
[v0.1.5]: https://github.com/sarathsp06/sparrow/compare/v0.1.4...v0.1.5
[v0.1.6]: https://github.com/sarathsp06/sparrow/compare/v0.1.4...v0.1.6
[v0.1.4]: https://github.com/sarathsp06/sparrow/compare/v0.1.3...v0.1.4
[v0.1.3]: https://github.com/sarathsp06/sparrow/compare/v0.1.2...v0.1.3
[v0.1.2]: https://github.com/sarathsp06/sparrow/compare/v0.1.1...v0.1.2
[v0.1.1]: https://github.com/sarathsp06/sparrow/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/sarathsp06/sparrow/releases/tag/v0.1.0
