---
title: Compatibility Policy
description: What Sparrow 1.0 promises to keep stable, what it doesn't, and how breaking changes are handled.
---

Sparrow follows [semantic versioning](https://semver.org/) from 1.0.0. Within a major version, a minor or patch release never breaks the surfaces listed below: upgrade `1.x` to a later `1.y` without changing your integrations, configuration or receivers. Anything that would break them waits for 2.0 and is announced in the release notes of a 1.x release first.

## What 1.0 freezes

| Surface | The promise |
|---------|-------------|
| **REST API under `/v1`** | Additive only. New endpoints, new optional request fields, new response fields and new enum values can appear in a minor release. Nothing is removed or renamed, no field changes type or meaning, and no optional field becomes required. Write clients that ignore fields they don't know. The spec is [`api/openapi.yaml`](https://github.com/sarathsp06/sparrow/blob/main/api/openapi.yaml). |
| **Portal gateway (`/portal/api/*`) and `POST /invite/redeem`** | Same rules as `/v1`: they re-dispatch to it. |
| **Environment variables** | Every documented `SPARROW_*` variable, `DATABASE_URL` and the `OTEL_*` variables keep their name, format and default. New variables can be added; a default changes only if the old one was unsafe, and the release notes say so. See [Configuration](/sparrow/getting-started/configuration/). |
| **CLI flags and commands** | The `sparrow` CLI's commands, flags and exit codes, the `--version` and `--migrate` flags of `sparrow-server`, and the `--config` and `--version` flags of `sparrow-sinks` and `sparrow-sources`. Structured output (`--output json` or `yaml`) is additive like the REST API. Human-readable output can change. |
| **Webhook envelope** | The JSON body a receiver gets without a transform: `version`, `event_id`, `event_name`, `timestamp`, `attempt`, `payload`. Fields can be added, never removed or retyped. |
| **Webhook signatures** | The `webhook-id`, `webhook-timestamp` and `webhook-signature` headers, the signed message `{id}.{timestamp}.{body}`, and the `v1` (HMAC-SHA256) and `v1a` (Ed25519) schemes. See [Verify signatures](/sparrow/guides/verify-signatures/). A new scheme can be added alongside; existing ones stay. |
| **Transform templates** | Every [template function](/sparrow/reference/template-functions/) and the template context fields. A template that renders on 1.0 renders the same on any 1.x. |
| **Database migrations** | Forward-only from `000042`. Every 1.x migration applies on top of the previous release's schema while older replicas are still running (see [Upgrading to 1.0](/sparrow/deployment/upgrading-to-1-0/#rolling-upgrades-and-replicas)). Down migrations exist for development, not for production rollback: to go back, restore a backup. |
| **Encryption keyring** | Data encrypted under a key stays readable as long as that key is in `SPARROW_ENCRYPTION_KEYS`. |
| **Go libraries** | The separately versioned modules: `pkg/signature`, `pkg/template`, `pkg/access` (and its `memstore`, `pgstore`, `storetest`, `httpauth` packages), and the library packages of `satellites/sparrow-sinks` and `satellites/sparrow-sources`. They follow Go's module compatibility rules: no exported identifier is removed or changes signature in a `v1` tag. They are tagged in lockstep with the server (`pkg/signature/v1.2.3` ships with `v1.2.3`). |

## What isn't covered

- **Anything under `internal/`** and the root module `github.com/sarathsp06/sparrow` as a Go dependency. It isn't `go install`-able (its `go.mod` has `replace` directives for the workspace) and its packages can change in any release.
- **The database schema as an API.** Tables and columns are Sparrow's own. Read them for debugging, but don't build on them: a migration can reshape them in any minor release.
- **The embedded UI**: its routes, layout and the requests it makes.
- **Log lines.** Messages and fields of the JSON logs can change. Alert on metrics, not on log text.
- **Metric names and labels** are kept stable where possible, and any rename is called out in the release notes, but they aren't frozen in 1.0.
- **Performance characteristics** such as throughput, queue cooldowns and pool sizes, and the defaults of tuning variables that exist to be tuned.
- **Behaviour that contradicts the docs.** A bug fix that brings Sparrow back in line with its documentation isn't a breaking change, even if something relied on the bug.
- **Pre-releases.** `vX.Y.Z-rc.N` tags make no promise beyond their own testing window.

## Deprecations

A surface that will go away in the next major version is marked deprecated in a minor release: in the OpenAPI spec (`deprecated: true`), in the docs, and in the release notes. It keeps working for the rest of the major version.

## Security fixes

A security fix may tighten behaviour in a minor or patch release when keeping the old behaviour would leave users exposed (for example, rejecting a request that was accepted unsafely). The release notes say what changed and why. See [SECURITY.md](https://github.com/sarathsp06/sparrow/blob/main/SECURITY.md) to report a vulnerability.

## Supported versions

Fixes land on the latest minor release. Upgrade within the major version to get them: that is what the promises above are for.
