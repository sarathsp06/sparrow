# Satellite ideas: landscape research and candidates

Date: 2026-10-07. Status: research note, nothing here is committed to.

Sparrow ships four satellites today, all built on the public REST API and the
Standard Webhooks signature, none importing `internal/`:

| Satellite | Direction | What exists |
|---|---|---|
| `sparrow` CLI | both | init, push, events (list/versions/export/import), webhooks, tail, stats, listen, use, tokens, invite(s), recipes, template test, functions |
| `recipes` | out | Slack, Discord, ntfy, PagerDuty, ClickHouse, Twilio, SendGrid, CloudEvents |
| `sparrow-sources` | in | cron emitter; Stripe and GitHub signature verifiers |
| `sparrow-sinks` | out | SMTP email, S3/MinIO archive, OTLP logs |

This note surveys what comparable products ship around their cores (section 1),
then lists satellite candidates ranked by fit and effort (sections 2 to 4).
Section 5 is the citation appendix from the research.

## 1. What the landscape ships as "ecosystem"

Products surveyed: Svix, Hookdeck (and its open-source Outpost), Convoy, Hook0,
Webhook Relay, HostedHooks, Standard Webhooks, plus adjacent platforms
(EventBridge, Knative, Argo Events, Dapr, Redpanda Connect, Inngest,
Trigger.dev, Novu, Sequin, Debezium, n8n/Zapier/Make/Pipedream/Windmill) and
dev tooling (ngrok, Stripe CLI, Svix Play, RequestBin, Speakeasy/Fern).

Recurring component types, with who ships them:

| Component type | Shipped by | Sparrow today |
|---|---|---|
| Message-bus destinations (Kafka, SQS, Pub/Sub, RabbitMQ, NATS, Service Bus, Kinesis, EventBridge) | Outpost (11 types), Svix Bridge, Dapr, Redpanda Connect | none |
| Message-bus sources (queue to webhook) | Convoy ingest, Svix Bridge, Kafka Connect HTTP sink | none |
| Postgres CDC as event source | Sequin, Debezium Server, Supabase DB webhooks, Hasura event triggers | none |
| Provider signature registry (30 to 70 providers) | Hookdeck, ngrok Traffic Policy, verihook | Stripe, GitHub |
| "Catch hook" unique-URL ingestion | n8n, Zapier, Make, Pipedream, Windmill, Hookdeck | none |
| MCP server for agents | Hookdeck CLI (11 tools, `--allow-write`), Svix, RequestBin, PactFlow | none |
| Declarative config from code / GitOps | Svix AutoConfig GitHub Action, Terraform providers (Svix, Hookdeck) | event-type export/import only |
| Helm chart / k8s manifests | Convoy | docker-compose only |
| Grafana dashboards, alert rules | nobody | none (but `/metrics` exists) |
| Fixture triggering from the CLI | Stripe CLI `trigger` (100+ events) | none |
| Hosted capture with a REST API for CI assertions | Svix Play, RequestBin, Hookdeck Console | `sparrow listen` (local only) |
| Archive and replay a time window | EventBridge, Pipedream, Hookdeck | S3 sink writes, nothing reads it back |
| Typed event SDKs from event-type schemas | Svix, Speakeasy (OpenAPI `webhooks` object) | OpenAPI client only |
| Inbox/outbox consumer libraries | Switchboard (Laravel), OutboxNet (.NET), Sequin | verification examples in 8 languages |
| AI-agent "webhook skills" for coding assistants | Hookdeck Webhook Skills (Stripe/Shopify/GitHub x Next/Express/FastAPI) | none |
| OpenAPI import of event schemas | Convoy `convoy openapi` | none |

Not gaps, despite one report claiming so: Sparrow already has an embeddable
consumer portal, server-side transforms, operational (system) events, and
per-webhook rate limiting.

## 2. Extend the existing satellites (lowest effort, highest leverage)

### 2.1 Sources

- **Standard Webhooks verifier as a generic provider.** One config block
  (`standard_webhooks: {path, secret, sparrow_event_prefix}`) verifies with
  `pkg/signature.VerifyHMAC` and covers every Svix-signed SaaS at once: Clerk,
  Resend, WorkOS, Lemon Squeezy, OpenAI, Anthropic, Supabase, Vanta, Kong. It
  also enables Sparrow-to-Sparrow federation (2.4). Cheapest possible win.
- **More named providers,** in order of appearance across competitor lists:
  Shopify (HMAC-SHA256 base64), Slack (versioned HMAC + timestamp), Linear,
  Paddle, Twilio (HMAC-SHA1 over URL + params), SendGrid events (ECDSA),
  Discord (Ed25519, `pkg/signature.VerifyEd25519` already exists), GitLab
  (shared token), Mailgun, Postmark, PagerDuty. Each is a path, a secret, a
  verify function, and an event-name mapping: one `sources/<provider>`
  package with a `Config` and a `Handler` built on `sources.Handle`, like
  `sources/stripe`.
- **Catch hook: unverified unique-URL ingestion.** `catch: [{path: /in/abc,
  event: form.submitted, secret: optional-bearer}]`. Turns any "POST a JSON
  blob here" integration (Zapier, Make, n8n, IoT, internal scripts) into a
  typed Sparrow event. Event name from config, a header, or a path segment.
- **Postgres LISTEN/NOTIFY source.** `pg_notify: {dsn, channels: [...]}`:
  each NOTIFY payload becomes an event. Zero extensions, fits the
  Postgres-only ethos, and is the minimum viable CDC story (see 3.2 for the
  full one). Pair it with a shipped SQL snippet that installs a row trigger.
- **Message-bus sources.** Kafka, NATS JetStream, SQS, Pub/Sub, Redis Streams
  consumers that push each message as an event (Svix Bridge, Convoy ingest).
  Heavier dependencies; see 3.1 for whether this belongs in `sparrow-sources`
  or its own binary.

### 2.2 Sinks

- **Message-bus sinks** (same list as above). Outpost ships 11 destination
  types and this is the single most common ecosystem component. Each sink is
  `deliverFunc` plus a config section, exactly the existing `sinkDef` pattern.
- **Postgres table sink ("webhook to database").** Insert the envelope into a
  configured table (JSONB payload, event id as primary key for dedup). Covers
  the Sequin/Supabase "land events in my DB" use case with no Kafka.
- **GCS and Azure Blob** next to S3, since Svix Ingest lists all three.
- **Filesystem / stdout sink** for air-gapped debugging and tests.
- **OTLP metrics sink**: count deliveries by event name as an OTel counter,
  complementing the existing OTLP logs sink.

### 2.3 Recipes

Grouped by how often they appear across Svix Connectors, Novu, Hookdeck
integrations and Webhook Relay alerting:

- Chat: Microsoft Teams, Google Chat, Telegram, Mattermost, Zulip.
- Incidents: Opsgenie, Grafana OnCall, incident.io, Better Stack, Datadog
  Events API, Grafana annotations.
- Work tracking: Jira issue create, Linear issue create, GitHub issue or
  repository_dispatch (which turns a Sparrow event into a GitHub Actions run).
- Email/SMS/push: Resend, Postmark, Mailgun, Amazon SES, Pushover, Gotify,
  OneSignal.
- Data: BigQuery streaming insert, PostgREST/Supabase insert, Elasticsearch
  index, Loki push, Splunk HEC, Axiom.
- Workflow engines: Inngest `send event`, Trigger.dev, Temporal signal via
  HTTP, Argo Events webhook source, n8n/Zapier/Make catch hooks, Windmill.
- CRM: HubSpot timeline event, Salesforce platform event.
- Sparrow: a `sparrow` recipe that re-pushes to another Sparrow instance
  (federation, 2.4).

Also worth adding a `recipes test` CI target that renders every recipe
against a sample envelope in `satellites/recipes/testdata`, since the
`json`-quoting rule is the thing contributors most often break.

### 2.4 CLI

- **`sparrow trigger <event-type>`**: generate a fixture payload from the
  event type's JSON Schema (or a stored example) and push it. The Stripe CLI
  pattern; makes `listen` plus `trigger` a complete local loop.
- **`sparrow apply -f sparrow.yaml`**: declarative event types, webhooks,
  subscriptions and recipe applications, idempotently reconciled. This is
  Svix AutoConfig and 80% of a Terraform provider at 10% of the cost, and it
  wraps naturally in a GitHub Action (4.3).
- **`sparrow replay --from --to --event`**: re-push a window from the S3
  archive the sink already writes. Turns the sink into EventBridge-style
  archive and replay; the server's repush-jobs API does the rest.
- **`sparrow events import --openapi spec.yaml`**: create event types from an
  OpenAPI 3.1 `webhooks` object (Convoy's `convoy openapi`). The reverse,
  AsyncAPI export of event types, gives consumers a contract to test against.
- **`sparrow listen --inspect`**: keep captured deliveries in memory, serve a
  tiny local page and a JSON endpoint so CI can assert on what was delivered
  (the Svix Play API idea, without hosting anything).
- **`sparrow mcp`**: see 3.3; the Hookdeck precedent is to ship the MCP server
  inside the CLI binary rather than as a separate install.
- **`sparrow doctor`**: check server reachability, API key, private-network
  guard, clock skew for signatures, and recipe param placeholders.

## 3. New satellites

### 3.1 `sparrow-bus`: message-bus bridge, both directions

Kafka, NATS JetStream, RabbitMQ, SQS, Google Pub/Sub, Redis Streams as both
source (consume, push event) and sink (verified delivery, publish). This is
the most-shipped ecosystem component across every product surveyed (Outpost,
Svix Bridge, Convoy, Dapr, Redpanda Connect). Reasons for a separate binary
rather than folding into sources/sinks: client libraries are heavy (the
Kafka and cloud SDKs dwarf the rest of the satellite), and a team using Kafka
rarely needs SMTP. (Library users are already isolated: each source and sink
is its own package, so importing `sources/stripe` never compiles a Kafka
client; the separate binary only keeps the shipped `sparrow-sources` and
`sparrow-sinks` lean.) Keep the per-package shape (`Config` + `Handler`, or
`New` + `Run` for consumers) so recipes, docs and tests read the same.

Effort: medium. Core changes: none.

### 3.2 `sparrow-cdc`: Postgres change capture as events

Logical replication (`pgoutput` or `wal2json`) from any Postgres into typed
Sparrow events: `orders.inserted`, `orders.updated` with old/new rows. Sequin
proved the market and the shape (single container, SQL filters, exactly-once
via idempotency keys, which Sparrow already has on ingest). This is the one
idea that is strictly on-brand: Postgres is Sparrow's only dependency and
most self-hosting teams already keep the source of truth there. Tiered
delivery: ship the LISTEN/NOTIFY source (2.1) first, then the replication
slot consumer.

Effort: high (slot management, backfill, schema changes). Core changes: none.

### 3.3 `sparrow mcp`: MCP server for coding agents and operators

Read tools: list events, deliveries, attempts, webhook health, stats,
template functions, recipes; test a template; explain a delivery failure
category. Write tools behind `--allow-write`: push event, retry delivery,
pause/resume, apply recipe. Hookdeck, Svix and RequestBin all shipped this
in the last year, and the "fix my failing webhook" loop is exactly what an
agent is good at. Builds on the existing CLI client code, so it is a cobra
subcommand, not a new module.

Effort: low to medium. Core changes: none.

### 3.4 `sparrow-inbox`: durable consumer-side receiver

The mirror image of sinks: a small library plus reference service that a
receiver runs. It verifies the Standard Webhooks signature, writes the
envelope into a Postgres `webhook_inbox` table keyed by event id (dedup),
acknowledges 200 immediately, and lets the app consume from the table in its
own transaction. Covers the inbox/outbox pattern that Switchboard and
OutboxNet sell to Laravel and .NET teams, in Go first, then as framework
middleware packages (chi, Express, FastAPI, Rails) generated from the same
contract. Pairs with 4.2 typed payloads.

Effort: medium. Core changes: none.

### 3.5 `sparrow-capture`: a self-hosted webhook.site

A throwaway inbound URL per bin, a list/inspect page, replay-to-localhost,
and a JSON API for CI assertions. Svix Play, RequestBin and Hookdeck Console
all exist because people do not want to ngrok production. Self-hosted first
users are exactly the people who cannot use those. Overlaps with
`sparrow listen --inspect` (2.4); do the CLI flag first and promote to a
service only if teams want shared bins.

Effort: medium. Core changes: none.

### 3.6 Sparrow-to-Sparrow federation

Not a new binary: the Standard Webhooks source (2.1) plus a `sparrow` recipe
(2.3). A spoke Sparrow in each VPN segment forwards selected events to a hub,
signatures verified on both hops, with the hub's delivery ledger as the
audit trail. Worth a docs page and a docker-compose example because it is a
story none of the competitors tell for self-hosted deployments.

## 4. Operations, codegen and agent-facing packaging

### 4.1 Helm chart and Grafana pack

Convoy ships Helm; nobody ships dashboards. A `deploy/helm/sparrow` chart
(server, migrate job, optional Postgres) plus `deploy/grafana/` with a
dashboard JSON built on the existing `/metrics` names (delivery success
rate, latency, queue backlog, webhooks by health) and a Prometheus alert
rules file. The dashboard is cheap and differentiating.

### 4.2 Typed event payloads from event-type schemas

`sparrow events codegen --lang ts|go|py` emits payload types from the stored
JSON Schemas, and the OpenAPI export gains a `webhooks` object listing every
event type so Speakeasy or Fern can generate typed receiver handlers for
free. This closes the "SDK breadth" gap listed in `why-sparrow` without
hand-maintaining SDKs.

### 4.3 GitHub Action and Terraform provider

`sparrow-action`: runs `sparrow apply` (2.4) or `sparrow events import` from
a repo on push, mirroring Svix AutoConfig. A `terraform-provider-sparrow`
is the natural follow-up once `apply` has stabilised the resource model; it
is a separate module under `satellites/` like the CLI.

### 4.4 Webhook skills for coding agents

A `satellites/skills/` directory of SKILL.md files: "receive a Sparrow
webhook in Next.js / FastAPI / Go / Rails with verification and idempotency",
"apply a recipe", "debug a failed delivery". Hookdeck shipped this in
February 2026 and it is nearly free to produce from existing docs.

## 5. Ranked recommendation

Ordered by expected value against effort, assuming the boundary rule holds
(public API only, no `internal/` imports):

1. Standard Webhooks generic source plus `sparrow` recipe (federation). Days.
2. `sparrow mcp` inside the CLI. About a week.
3. `sparrow apply -f` declarative config, then the GitHub Action. One to two weeks.
4. Message-bus sinks and sources (`sparrow-bus`), Kafka and NATS first. Two to three weeks.
5. Postgres table sink and LISTEN/NOTIFY source (the cheap CDC story). Days.
6. Recipes batch: Teams, Telegram, Opsgenie, Resend, Jira, Inngest, repository_dispatch. Days each.
7. `sparrow trigger` from schemas and `listen --inspect`. One week.
8. Helm chart plus Grafana dashboard and alert rules. One week.
9. `sparrow-inbox` consumer library. Two weeks.
10. `sparrow-cdc` with logical replication. A month plus.

Explicitly not recommended as satellites: a JavaScript transform runtime
(contradicts the Go-template decision), a hosted multi-tenant SaaS layer, a
Novu-style preference-driven notification router (recipes plus
subscriptions already cover channel fan-out), or anything that needs Redis.

## 6. Research appendix (sources)

Webhook platforms:
- Svix: open-source server, Ingest (S3/GCS/Azure endpoints), Play, Stream (Nov 2025), Bridge (queues to webhooks via Omniqueue, with reverse mode), Connectors (Aug 2026, Slack/Discord/HubSpot/Teams/email/OTel), Terraform provider (Apr 2025), Webhooks AutoConfig GitHub Action (Jul 2026), Rust CLI v2 and v2 SDKs in 8 languages (Aug 2026), MCP server. https://www.svix.com/blog/changelog-08-2026/ , https://www.svix.com/blog/changelog-07-2026/ , https://www.svix.com/blog/changelog-04-2025/ , https://svix.com/blog/send-webhooks-from-your-message-queue-svix-bridge , https://www.svix.com/resources/tutorials/webhook-mcp-server/
- Hookdeck: Event Gateway with 60+ source verifiers, Outpost GA Apr 2026 with 11 destination types (Webhook, Hookdeck, SQS, S3, Kinesis, EventBridge, Service Bus, Pub/Sub, RabbitMQ, Kafka, Cloudflare Queues), CLI v3 (Sep 2026) with built-in MCP server (11 tools, `--allow-write`), Terraform provider, Webhook Skills for coding agents (Feb 2026). https://hookdeck.com/docs/outpost , https://hookdeck.com/docs/mcp , https://hookdeck.com/changelog/cli-v3.md , https://hookdeck.com/changelog/webhook-skills , https://hookdeck.com/supported-providers
- Convoy: ingest from HTTP, Kafka, SQS, Pub/Sub, RabbitMQ with JS transforms; Portal Links; `convoy openapi` schema extraction; 6 SDKs; Helm chart v1.0. https://www.getconvoy.io/docs/product-manual/sources , https://www.getconvoy.io/docs/cli , https://www.getconvoy.io/blog/convoy-helm-chart-v1.0
- Hook0 (Rust, subscriber portal, EU SaaS): https://hook0.com . Webhook Relay (alerting to Slack/Discord/Teams/Telegram/Pushover/SMTP, cron webhooks, CloudEvents validator): https://webhookrelay.com . HostedHooks: https://elements.heroku.com/addons/hosted-hooks
- Standard Webhooks spec and libraries (Python, JS/TS, Java/Kotlin, Rust, Go, Ruby, PHP, C#, Elixir, D); adopters include OpenAI, Anthropic, Supabase, Kong, PagerDuty, Twilio, ngrok. https://github.com/standard-webhooks/standard-webhooks

Adjacent platforms:
- EventBridge (Pipes sources, API Destinations, Schema Registry, Archive and Replay): https://docs.aws.eu/eventbridge/latest/userguide/eb-pipes-event-source.html
- Knative Eventing sources and sinks: https://docs.redhat.com/en/documentation/red_hat_openshift_serverless/1.29/html/eventing/event-discovery
- Argo Events (20+ sources, 10+ triggers): https://codefresh.io/learn/argo-events
- Dapr bindings (50 components): https://docs.dapr.io/reference/components-reference/supported-bindings
- Redpanda Connect (300+ connectors, `http_server` input, `http_client` output): https://docs.redpanda.com/connect/components/about/
- Novu channels (17 email, 16 SMS, push, chat): https://novu.co/integrations/channels
- Sequin (Postgres CDC to webhooks/queues, exactly-once): https://github.com/sequinstream/sequin ; Supabase database webhooks (pg_net): https://supabase.com/docs/guides/database/webhooks ; Debezium Server HTTP sink: https://dev.to/sequin/standing-up-debezium-server-for-postgres-cdc-b75
- Kafka Connect HTTP sink: https://docs.confluent.io/kafka-connectors/http/current/overview.html , https://aiven.io/developer/kafka-webhooks-http-sink
- Trigger.dev vs Inngest: https://automationatlas.io/guides/trigger-dev-vs-inngest-2026-comparison/

Developer tooling:
- ngrok Traffic Policy verify-webhook (70+ providers): https://ngrok.com/docs/traffic-policy/actions/verify-webhook.md
- Stripe CLI listen/trigger: https://stripe.com/docs/stripe-cli/overview
- Svix Play (REST API for captured requests): https://svix.com/play ; RequestBin (replay, forwarding, MCP): https://requestbin.net ; Hookdeck Console: https://hookdeck.com/webhooks/platforms/requestbin-alternatives
- Speakeasy webhook support in generated SDKs: https://speakeasy.com/post/release-webhooks-support ; Fern: https://buildwithfern.com
- PactFlow AsyncAPI contract testing: https://support.smartbear.com/swagger/contract-testing/docs/en/on-premises-admin-guide/on-premises-release-notes/2-6-0.html
- Provider signature comparison: https://dev.to/eventdock/webhook-signature-verification-compared-stripe-paddle-hubspot-slack-twilio-4jfk ; verihook: https://github.com/creatorpiyush/verihook
- Convoy transactional outbox: https://www.getconvoy.io/blog/webhooks-with-transactional-outbox
