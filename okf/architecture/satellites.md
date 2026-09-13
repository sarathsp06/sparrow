---
type: Concept
title: Satellites (Ecosystem Ring)
description: Companion binaries that orbit the Sparrow core over its public REST API and Standard Webhooks signature — never touching internal/
tags: [satellites, ecosystem, architecture]
timestamp: 2026-09-13T00:00:00Z
---

# Satellites

Satellites are companion tools that **orbit the core**. They speak only
Sparrow's public contract — the REST API (`POST /v1/namespaces/{ns}/events`,
webhook/subscription CRUD) and the Standard Webhooks signature — and are built
entirely outside `internal/`.

## Boundary rule

Code under `satellites/` MUST NOT import `internal/`. The only shared code it
may depend on is `pkg/` — currently `pkg/signature` (verify/sign deliveries)
and `pkg/template` (render transforms). This keeps the ecosystem decoupled from
core internals; a satellite is just another API client + webhook receiver.

## The ring

| Satellite | Direction | Role |
|---|---|---|
| `satellites/sparrow` | tooling | The `sparrow` CLI: `init`, `push`, `listen`, `tail`, `use`, `template test`. A pure REST client + a local receiver for `listen`. |
| `satellites/sparrow-sources` | world → Sparrow | Cron emitter + Stripe/GitHub webhook normalizers + Kafka consumers that republish external triggers as typed Sparrow events. |
| `satellites/sparrow-sinks` | Sparrow → world | Signed-delivery receiver forwarding out of HTTP land: SMTP email, S3/MinIO archive, OTLP log export. Stateless; leans on core retries. |
| `satellites/recipes` | config | Apply-time YAML transforms (Slack, Discord, ntfy, PagerDuty, ClickHouse) rendered via `pkg/template` — no service runs, the transform happens inside core delivery. |

## Why this shape

Every sink is a verified HTTP receiver in, provider call out, non-2xx on
failure — so Sparrow's retry/backoff and delivery ledger apply to all of them
for free. Sources sit in front of the core and push through the same event API.
The value is that the broker (Postgres + River + delivery ledger) stays the one
durable, observable path; satellites are stateless edges.

## Citations

- `satellites/` — CLI, sources, sinks, recipes
- `docs/src/content/docs/satellites/` — user-facing satellite docs
- `pkg/template`, `pkg/signature` — the only core code satellites import
