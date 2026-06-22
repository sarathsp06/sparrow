---
okf_version: "0.1"
---

# Sparrow

A self-hosted webhook delivery platform with event-driven architecture. Accepts webhook registrations, event definitions, and subscriptions (with Go-template payload transformation), then fans out events into a River job queue for async HTTP delivery with retries, health tracking, and error classification.

# Knowledge

* [Architecture Overview](/architecture/overview.md) — system architecture and design
* [Data Flow](/architecture/data-flow.md) — event push through delivery pipeline
* [Packages](/packages/index.md) — Go package reference
* [Services](/services/index.md) — gRPC/Connect-RPC service surface
* [Concepts](/concepts/index.md) — domain concepts
* [Database Schema](/database/schema.md) — tables, relationships, migrations
* [Configuration](/config/env-vars.md) — environment variables and tuning
* [Frontend](/frontend/index.md) — SvelteKit UI
* [DevOps](/devops/index.md) — Docker, Helm, CI/CD
* [References](/references/index.md) — proto definition, external dependencies
