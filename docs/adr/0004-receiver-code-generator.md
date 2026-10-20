# 4. Generate receiver projects from a webhook with a separate, deterministic module

- Status: accepted
- Date: 2026-10-20

## Context

Every Sparrow receiver has to do the same few things correctly: verify the
signature on the raw body before parsing it, reject stale timestamps,
deduplicate on `webhook-id`, and decode payloads whose shape depends on the
event type and on the subscription's transform template. Today users copy a
helper from `client/verify/` and write the rest by hand, and the rest is where
mistakes happen: parsing JSON before verifying it, routing on a field a
transform removed, or ignoring a non-JSON transform output.

Sparrow already knows everything needed to write that code for a given
webhook: its signing scheme, its subscriptions, their event types (schema and
sample), their templates and custom headers. The people who need the receiver
are mostly consumers in the portal, then operators, developers with the CLI,
and coding agents through MCP.

## Decision

Add a generator that turns one webhook into a ready-to-run receiver project
(initial targets `go-nethttp`, `python-fastapi`, `node-express`), exposed by
one endpoint, `POST /v1/consumers/{consumer}/webhooks/{id}:generateReceiverCode`,
that the admin UI, the portal, the CLI (`sparrow scaffold`) and the MCP server
all call.

- **Module boundary.** All generation logic lives in a new, separately
  versioned module, `pkg/receivercode`, depending only on the standard
  library, `pkg/template` and `pkg/signature`. It takes plain data (no
  database types, no `context`) and the server reaches it through a single
  call, `receivercode.Generate`, from one handler file. The CLI imports it
  directly for `--offline` generation. It is tagged in lockstep like the
  other `pkg/*` modules (ADR 0002) but documented as **experimental**,
  outside the 1.x promise, until stated otherwise. The endpoint is covered
  by the `/v1` promise; the generated code may improve between releases.
- **Templates, not AI.** The three targets come from deterministic code
  templates (`embed.FS`). The same input gives byte-identical output: stable
  ordering, no timestamps or random values in files. AI may later write
  framework glue for other targets, never verification.
- **Security code is vendored, never generated.** Signature, timestamp and
  raw-body handling come from tested helpers: `pkg/signature` for Go and the
  `client/verify/` samples for Python and TypeScript, copied into
  `pkg/receivercode/helpers/` with a test that keeps them byte-identical.
  Secrets are read from `SPARROW_WEBHOOK_SECRET`; only an Ed25519 public key
  may be embedded. Verification runs on the raw bytes before any parsing,
  whatever the body format.
- **Route on headers.** Deliveries gain `X-Sparrow-Event-Type`,
  `X-Sparrow-Event-Version` and `X-Sparrow-Subscription-ID`. Generated code
  dispatches on the subscription ID, falling back to the event type, never on
  body content, because a transform can drop the event name from the body.
- **Format without a migration.** The output format comes from the
  subscription's custom `Content-Type` header if set, else is detected from
  the sample rendered through the template (`json`, `xml`, `text`).
- **Safe defaults.** Generated receivers check a 5-minute timestamp tolerance
  and deduplicate on `webhook-id` by default, each behind a clearly marked
  constant. Tests are built from per-webhook vectors signed with a throwaway
  secret or keypair. `SPARROW_RECEIVER_TEST_MODE=1` makes a receiver log one
  JSON line per handler call, which the behavioural contract tests read.
- **A test send that stays out of health.** `:sendTestEvent` sends one signed
  delivery of the rendered sample synchronously through the real request
  builder, SSRF guard and timeouts, marked `X-Sparrow-Test: true`. It writes
  no delivery row and never touches health or auto-disable (Design
  Principle 3), and is rate-limited per webhook in process.
- **Testing in two tiers.** Golden snapshots and a Go build of the Go target
  run on every PR; per-language builds, generated tests and a behavioural
  contract against a running receiver run in a separate workflow.

## Alternatives rejected

- **Generate in the browser (WASM).** It would ship the generator to every UI
  load, could not serve the CLI or MCP from the same code, and the portal
  would still need the server for schemas, samples and templates.
- **A separate generator service.** A new deployable for a request that takes
  milliseconds breaks the self-hosted, Postgres-only shape (Design
  Principles 6 and 7) for no gain over a library call.
- **AI-written verification.** Signature checking must be identical and
  tested everywhere. Model output varies between runs and cannot be reviewed
  once per release, so verification stays vendored and AI is limited to glue.

## Consequences

- One more module in `go.work`, `MODULE_TEST_PATHS`, the release script,
  dependabot and the per-module CI steps.
- The vendored helpers are a second copy of `client/verify/`; the identity
  test fails the build when they drift.
- The TypeScript helper is vendored as-is, so the `node-express` target is a
  TypeScript project run by Node's built-in type stripping (Node 22.18 or
  later), with no build step.
- The test-send limiter is per replica: N replicas allow N test sends per
  interval. That bounds abuse well enough for a manual button without a
  shared store.
