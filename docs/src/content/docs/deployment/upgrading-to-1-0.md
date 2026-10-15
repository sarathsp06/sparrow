---
title: Upgrading to 1.0
description: Move a 0.x Sparrow deployment to 1.0 -- migrations on large instances, rolling upgrades, the Postgres connection budget, and the encryption keyring.
---

Sparrow 1.0 is the first release covered by the [compatibility policy](/sparrow/reference/compatibility/). The upgrade from 0.x is an ordinary deploy for most installations: the server applies its own migrations on startup. This page covers the cases where that isn't enough.

## Before you start

- **Back up the database.** Migrations are forward-only: the way back from a failed upgrade is a restore, not a down migration.
- **Pin the image.** Use `ghcr.io/sarathsp06/sparrow:1.0.0` (no leading `v`), not `latest`. `latest` only moves on stable releases, never on a `-rc.N`, but a pinned tag is still the only way to know what is running. `sparrow-server --version` prints the version of a binary.
- **Read the release notes** of every minor version you are skipping.

## Run migrations before the rollout on large instances

Every server applies pending migrations at startup, before it serves traffic, and Postgres advisory locks make concurrent starts safe: one replica migrates, the others wait. That is fine while migrations take seconds.

On a large instance (millions of deliveries), a migration that rewrites or indexes a big table can take minutes. Meanwhile every new pod is blocked on the lock and not ready, and a liveness probe with a short timeout can kill the one doing the work, which starts the migration over. Run them once, before the rollout, instead:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: sparrow-migrate-1-0-0
  annotations:
    # Helm: run before the release's Deployment is updated
    helm.sh/hook: pre-upgrade,pre-install
    helm.sh/hook-delete-policy: before-hook-creation
    # Argo CD: run in the PreSync phase
    argocd.argoproj.io/hook: PreSync
spec:
  backoffLimit: 0          # a failed migration needs a person, not a retry loop
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: migrate
          image: ghcr.io/sarathsp06/sparrow:1.0.0   # the version you are deploying
          args: ["--migrate"]                     # apply migrations and exit
          envFrom:
            - secretRef:
                name: sparrow                      # same config as the Deployment
```

`--migrate` loads the same configuration as the server, applies the River and application migrations, and exits. Once the Job has succeeded, the Deployment's pods find nothing to do and become ready at once. On a plain Docker host, run the image with `--migrate` before replacing the container.

## Rolling upgrades and replicas

Replicas share everything through Postgres: the River queue, the delivery rows, the health evaluator. Several replicas are safe, and old and new versions can run side by side during a rollout.

- **Within 1.x**, a rolling update is supported: each migration applies on top of the previous release's schema while replicas of that release are still running.
- **From 0.x to 1.0**, and whenever you skip several minor versions, run the migrate Job first and keep the rollout short. The 0.x releases didn't promise that old code keeps working against a newer schema.
- **Rollback** means restoring the backup. Don't run down migrations against production: they exist for development and can't bring back data a migration dropped or reshaped.
- **Shutdown.** On `SIGTERM` the server stops taking requests and gives running deliveries until `SPARROW_SHUTDOWN_TIMEOUT` (default `10s`) runs out, so keep `terminationGracePeriodSeconds` above that (Kubernetes' default of 30 is fine). A delivery cut off mid-attempt is picked up again by the queue and retried, so a receiver can see it twice: deduplicate on the `webhook-id` header, which stays the same across attempts.

## Postgres connection budget

Each replica opens up to **75 connections**: 50 for the River queue (`pgxpool`, 10 kept open even when idle) and 25 for application queries (`sqlx`). These sizes are fixed in 1.0.

Postgres ships with `max_connections = 100`, and reserves a few of those for superusers. So:

| Replicas | Peak connections | Fits `max_connections = 100`? |
|----------|-----------------|-------------------------------|
| 1 | 75 | Yes, with little room for a migrate Job, `psql` or a backup tool |
| 2 | 150 | No |
| 1, during a rolling update with `maxSurge: 1` | 150 | No |

Plan for `75 × (replicas + maxSurge)`, plus the migrate Job and whatever else connects (backups, monitoring, `psql`). When Postgres runs out, new connections fail with `too many connections` and the replica that can't connect fails its readiness probe. Either raise `max_connections` (managed services often tie it to instance size), or roll with `maxSurge: 0` and `maxUnavailable: 1` so the total never exceeds the steady-state count.

## Encryption keyring: never remove an old key

`SPARROW_ENCRYPTION_KEYS` can hold several keys, and `SPARROW_ENCRYPTION_PRIMARY_KEY_ID` picks the one that encrypts new data. Every webhook secret and secret header stays encrypted under the key that was primary when it was written. Nothing re-encrypts existing rows yet.

So **don't remove a key from the ring**, even after switching the primary. Any webhook secret or secret header still encrypted under it becomes unreadable, and deliveries to that webhook fail. Rotating is safe:

1. Add the new key: `SPARROW_ENCRYPTION_KEYS="old=…,new=…"`.
2. Make it primary: `SPARROW_ENCRYPTION_PRIMARY_KEY_ID=new`.
3. Keep `old` in the ring.

Retiring a key will become safe once Sparrow ships a tool that rewraps existing data under the primary key; its release notes will say so. Until then, treat every key that has ever been primary as permanent, and back up the whole keyring: lose it and the encrypted secrets are gone for good.

## The all-zeros encryption key is refused in production

The development defaults (`make run`, the local Compose file) use an all-zeros
key, which is public. With `SPARROW_ENVIRONMENT=production`, 1.0 refuses to start if
any key in `SPARROW_ENCRYPTION_KEYS` is all zeros; elsewhere it logs a warning.
If production ever ran on it, generate a real key (`openssl rand -hex 32`) and
make it primary before turning production mode on.

## `/health` is liveness only

`/health` no longer pings the database: it answers `200` while the process is
up, so a database outage makes pods unready instead of restarting them.
`/ready` checks the database and answers `503` from the moment `SIGTERM`
arrives. If you used `/health` to check the database (a load balancer, an
uptime monitor), point it at `/ready`.

## Template limits

Besides the 1 MB output and 5 second caps, a render now stops after 1,000,000
loop iterations and template calls, or 16 MB of strings and lists built by
helpers. `range` over an integer held in a variable is refused (a literal
like `{{ range 5 }}` already was), and `printf` widths above 1000 are rejected. A template that hits one of these fails its
deliveries with `template_error`; check yours with
`POST /v1/subscriptions:testTemplate` before upgrading. See
[Payload transformation](/sparrow/guides/payload-transformation/#good-to-know).

## Smaller behaviour changes

- A webhook's `request_timeout_seconds` (up to 300) now applies in full.
  Before, a hidden 30 second client timeout cut longer requests short.
- A delivery whose webhook or event lookup fails transiently stays `retrying`
  instead of showing `failed` and later flipping to `success`.
- `SPARROW_UI_INJECT_KEY` is no longer supported. If it is still set, the
  server logs a warning at startup and ignores it; the UI asks for a
  credential on the first `401` instead (master key, access token or invite).

## Renamed environment variables

Every setting is now under the `SPARROW_` prefix, apart from the standard
`DATABASE_URL` and `OTEL_*`:

| 0.x | 1.0 |
|-----|-----|
| `ENVIRONMENT` | `SPARROW_ENVIRONMENT` |
| `CORS_ALLOWED_ORIGINS` | `SPARROW_CORS_ALLOWED_ORIGINS` |

The old names are no longer read; startup logs a warning when one is set
without its new name. `SPARROW_ENVIRONMENT=production` now matches in any case
(`Production` counts), and a value like `prod` logs a warning instead of
silently running as development.

## API changes

- **Arrays are never `null`.** Responses send `[]` for an empty list, and the
  spec types every array as a plain array. A request that sends `null` for an
  optional array is treated as if it left the field out.
- **Timestamps are typed** `format: date-time` in the spec (the values were
  already RFC3339). Generated clients now get date-time types instead of
  strings; regenerate yours.
- **`:retryBatch` takes `retry_id`**, the field name the deliveries list
  returns, instead of `repush_id`. Event re-push (`events:rePush`) still
  takes `repush_id`.
- **No empty-string enum values.** `signature_type`, `on_transform_error`,
  `template_missing_key` and `template_source` on webhook registration, and
  the `health` filters, no longer accept `""`; leave the field out for the
  default.

## AI drafting needs `SPARROW_AI_ENABLED=true`

Before 1.0, setting `SPARROW_AI_API_KEY` (or `SPARROW_AI_BASE_URL` for an
OpenAI-compatible server) was enough to turn on **Draft with AI**. In 1.0,
`SPARROW_AI_ENABLED` is the switch and defaults to off. If you use drafting,
add:

```bash
SPARROW_AI_ENABLED=true
```

Without it the editor offers **Copy prompt** only, and startup logs a warning
that a key is set but drafting is off. With it, a missing provider setting no
longer stops the server: the draft request answers `503` naming the setting.
See [Turning drafting on and off](/sparrow/getting-started/configuration/#turning-drafting-on-and-off).

## Empty and `-` values mean unset

A variable set to an empty value or a lone `-` now counts as unset, so its
default applies. Before, an empty typed variable (`SPARROW_AI_TIMEOUT=`) stopped
startup, and `SPARROW_AI_API_KEY=-` counted as a key.

## Event type bundles

A bundle exported from a 0.x server and imported into 1.0 shows a "version differs" warning: the stamp records the exact server version that exported it. The import still works. Export again from 1.0 to clear it.

## Go library changes

If you import Sparrow's Go modules, 1.0 renames a few identifiers so they can stay put for the rest of the major version:

| Module | 0.x | 1.0 |
|--------|-----|-----|
| `pkg/template` | `TemplateEngine`, `NewTemplateEngine()` | `Engine`, `New()` |
| `pkg/template` | `GetTemplateFunctions()`, `TemplateFunc` | `Functions()`, `Func` |
| `pkg/template` | `GetFunctionMap()` | `FuncMap()` |
| `pkg/template` | `WebhookTemplateContext`, `NewWebhookTemplateContext()` | `WebhookContext`, `NewWebhookContext()` |
| `pkg/template` | `MaxTemplateOutputBytes`, `TemplateExecutionTimeout` | `MaxOutputBytes`, `ExecutionTimeout` |
| `pkg/access/pgstore` | `Schema`, `SchemaParts` (variables) | `Schema()`, `SchemaParts()` (functions) |

`pkg/template` also gains `MaxSteps`, `MaxWorkBytes` and `ErrBusy`, returned by `ExecuteWith` when no execution slot frees up within `ExecutionTimeout`.

`pkg/crypto`, `pkg/storage` and `pkg/errors` moved to `internal/`. They were never importable from outside the repository's own module in practice (the root module can't be fetched as a dependency), so nothing outside Sparrow should notice.
