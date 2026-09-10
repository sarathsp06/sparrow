[![CI](https://github.com/sarathsp06/sparrow/actions/workflows/ci.yml/badge.svg)](https://github.com/sarathsp06/sparrow/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/ghcr.io-sarathsp06%2Fsparrow-blue?logo=docker)](https://github.com/sarathsp06/sparrow/pkgs/container/sparrow)
[![API Spec](https://img.shields.io/badge/API-OpenAPI%203.1-blue)](api/openapi.yaml)

<p align="center">
  <img src="./web/src/lib/assets/favicon.svg" alt="Sparrow" width="120" height="120" />
</p>

# Sparrow

**Self-hosted webhook delivery platform** with async fan-out, retries, health tracking, and observability. Run it when you need dependable outbound webhooks without routing your payloads through a third-party service.

Everything runs against a single PostgreSQL database. Webhook secrets are encrypted at rest, every delivery is signed, and no data leaves the deployment unless you configure an exporter. That makes it a fit for regulated and air-gapped networks as much as for a small internal service.

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/KzfSI3?referralCode=otXr-t&utm_medium=integration&utm_source=template&utm_campaign=generic)

## Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Use Cases](#use-cases)
- [Architecture](#architecture)
- [Security](#security)
- [Verifying Webhook Signatures](#verifying-webhook-signatures)
- [Configuration](#configuration)
- [Deployment](#deployment)
- [Documentation](#documentation)
- [Contributing](#contributing)

## Features

### Delivery & Reliability
- **Event-driven fan-out:** push one event, deliver it to every matching subscription
- **At-least-once delivery:** configurable retries with exponential backoff
- **Idempotent ingestion:** optional idempotency keys stop duplicate processing
- **Per-webhook rate limiting:** leaky bucket, with HTTP 429 `Retry-After` parsing
- **Error classification:** DNS, TLS, timeout, connection refused, rate limited, and more, each flagged retryable or not
- **Bulk operations:** deterministic snapshot-based batch re-push and retry, up to 10K items

### Security
- **Webhook signing:** HMAC-SHA256 on every delivery, plus Ed25519 for webhooks set to `signature_type: ed25519`, both in [Standard Webhooks](https://www.standardwebhooks.com/) format, with verifiers for Go, Python, and TypeScript
- **Envelope encryption at rest:** AES-256-GCM with per-record data encryption keys for secrets and sensitive headers
- **SSRF protection:** private, loopback, and metadata IPs are blocked by default, and redirect targets are re-validated
- **Optional API key auth:** constant-time comparison on the `X-API-Key` header
- **No surprise egress:** no telemetry or callbacks leave the deployment unless you turn them on

### Developer Experience
- **Payload transformation:** per-subscription Go templates with built-in helpers to reshape payloads for each consumer
- **Soft schema validation:** warnings, not errors; events are always accepted and stored
- **REST/OpenAPI API:** versioned REST on `:8080`, interactive docs (Scalar) at `/docs`, spec at `/openapi.yaml`
- **Web dashboard:** embedded SvelteKit UI for webhooks, events, deliveries, and health
- **Health tracking:** per-webhook state machine (healthy/degraded/unhealthy) with rolling summaries

### Operations
- **PostgreSQL only:** no Redis, no message broker, nothing to run beyond one database
- **OpenTelemetry:** traces, metrics, and structured logs with job-level trace propagation
- **Hardened Helm chart:** NetworkPolicy, read-only rootfs, non-root, seccomp
- **One-click deploy:** Railway, Docker Compose, or any container platform

## Quick Start

Download [`deploy/docker-compose.yml`](deploy/docker-compose.yml) and start it:

```bash
curl -O https://raw.githubusercontent.com/sarathsp06/sparrow/main/deploy/docker-compose.yml
SPARROW_ENCRYPTION_KEY=$(openssl rand -hex 32) docker compose up -d
```

Open **http://localhost:8080** for the web UI.

> [!IMPORTANT]
> `SPARROW_ENCRYPTION_KEY` is the master key for everything encrypted at rest. Generate it once with `openssl rand -hex 32`, keep it in your secret manager, and back it up. Lose it and encrypted webhook secrets and headers are gone; leak it and you must rotate the key and re-encrypt.

### Send your first event

```bash
# Register an event type
curl -X POST http://localhost:8080/v1/event-types \
  -H "Content-Type: application/json" \
  -d '{"name": "order.created", "description": "New order", "active": true}'

# Register a webhook (subscription is created automatically)
curl -X POST http://localhost:8080/v1/namespaces/default/webhooks \
  -H "Content-Type: application/json" \
  -d '{"url": "https://httpbin.org/post", "events": ["order.created"], "active": true}'

# Push an event; Sparrow fans out and delivers
curl -X POST "http://localhost:8080/v1/namespaces/default/events?event=order.created" \
  -H "Content-Type: application/json" \
  -d '{"payload": {"order_id": "ord_123", "amount": 99.99}}'
```

Check delivery status in the web UI under **Deliveries**, or query the API:

```bash
curl "http://localhost:8080/v1/namespaces/default/deliveries?limit=5"
```

## Use Cases

- **SaaS webhook notifications:** tell customer endpoints when a resource changes
- **Internal event bus:** fan domain events out to downstream services over HTTP
- **Reliability layer:** add retries, health tracking, and observability to an existing webhook flow
- **Development and testing:** inspect deliveries, replay failed events, test payload transforms

## Architecture

```
PushEvent API
  -> persist event in PostgreSQL
  -> enqueue fan-out job (River)
     -> match subscriptions, apply transforms, create deliveries
     -> enqueue delivery jobs
        -> HTTP POST with HMAC signature
        -> retry on failure (server errors, timeouts, network errors)
        -> track health per webhook
```

Events are persisted before delivery. The [River](https://riverqueue.com) job queue handles at-least-once delivery with configurable retries (default: 3 attempts, 60s backoff). Failures are sorted into retryable categories (5xx, timeout, connection refused, network error, rate limited) and non-retryable ones (4xx, DNS, TLS), so you learn *why* a delivery failed, not just *that* it did. Every delivery is signed with HMAC-SHA256; webhooks set to `signature_type: ed25519` also get an Ed25519 signature, both in [Standard Webhooks](https://www.standardwebhooks.com/) format. Webhook secrets and sensitive headers are envelope-encrypted at rest with AES-256-GCM and per-record data encryption keys.

See [`okf/architecture/overview.md`](okf/architecture/overview.md) for the full pipeline design, error classification, and health state machine.

## Security

Sparrow assumes it runs inside a network you control. The controls below cover data at rest and in transit, outbound network access, and the container runtime.

### Data protection
- **Encryption at rest:** webhook secrets and sensitive headers are sealed with AES-256-GCM envelope encryption. Each record gets its own data encryption key, wrapped by the master key (`SPARROW_ENCRYPTION_KEY`). See [`okf/concepts/envelope-encryption.md`](okf/concepts/envelope-encryption.md).
- **Signed deliveries:** every outbound request carries HMAC-SHA256 (and optionally Ed25519) signatures in Standard Webhooks format, so receivers can authenticate the payload and reject replays.
- **Encryption in transit:** use `sslmode=require` (or stricter) in `DATABASE_URL` and terminate TLS at your ingress. Sparrow never requires plaintext transport.

### Network hardening
- **SSRF protection:** outbound webhook URLs that resolve to private, loopback, link-local, or cloud-metadata addresses are rejected by default, and redirect targets are re-checked. Set `SPARROW_ALLOW_PRIVATE_NETWORKS=true` only on trusted internal networks.
- **NetworkPolicy:** the Helm chart restricts ingress to Sparrow and isolates PostgreSQL so only Sparrow pods reach it.
- **CORS allowlist:** browser origins are denied unless you list them in `CORS_ALLOWED_ORIGINS`.

### Runtime hardening (Helm)
| Control | Setting |
|---------|---------|
| Run as non-root | `runAsNonRoot: true`, `runAsUser: 65532` |
| Immutable filesystem | `readOnlyRootFilesystem: true` |
| Drop all Linux capabilities | `capabilities.drop: [ALL]` |
| No privilege escalation | `allowPrivilegeEscalation: false` |
| Seccomp | `seccompProfile: RuntimeDefault` |
| Service account token | `automountServiceAccountToken: false` |

### Access control
- **Optional API key auth:** set `SPARROW_API_KEY` to require the `X-API-Key` header on every request. Comparison is constant-time. It is a shared secret, not a full identity system, so pair it with network controls.
- **Small dependency surface:** one PostgreSQL database, no broker or cache. Less to audit, less to attack.

> [!NOTE]
> Sparrow emits no telemetry by default. Observability data is exported only when you set `OTEL_EXPORTER_OTLP_ENDPOINT`, so it runs in air-gapped and egress-restricted environments.

> [!WARNING]
> With `SPARROW_API_KEY` unset, the API and dashboard are open to anyone who can reach the port. On shared or internet-facing deployments, set an API key **and** restrict network access.

## Verifying Webhook Signatures

Every delivery includes three [Standard Webhooks](https://www.standardwebhooks.com/) headers:

| Header | Example |
|--------|---------|
| `webhook-id` | `msg_abc123-def456` |
| `webhook-timestamp` | `1716048000` (Unix seconds) |
| `webhook-signature` | `v1,K7gNU3sdo+OL...` or `v1,K7gN... v1a,RjB2mN...` |

The signed message is always `{webhook-id}.{webhook-timestamp}.{raw request body}`, using the exact body bytes as received, not re-serialized JSON.

`webhook-signature` holds one or more space-delimited signatures:

- `v1,` is HMAC-SHA256, keyed with the webhook secret. It is present on every delivery (a secret is auto-generated at registration if you don't supply one). Secrets use the Standard Webhooks format `whsec_<base64>`; decode the base64 part to get the raw HMAC key.
- `v1a,` is Ed25519, present when the webhook's `signature_type` is `ed25519`. Verify it with the hex-encoded public key from the webhook's `signing_public_key` field. No shared secret needed.

Reject any delivery whose `webhook-timestamp` is more than 5 minutes off your clock (replay protection). The helpers below handle the details: secret decoding, multi-signature parsing, constant-time comparison, and the timestamp window.

| Language | Helper |
|----------|--------|
| Go | `go get github.com/sarathsp06/sparrow`, then import [`pkg/signature`](pkg/signature/signature.go) |
| Python | copy [`client/verify/python/sparrow_verify.py`](client/verify/python/sparrow_verify.py) (HMAC is stdlib-only; Ed25519 needs `cryptography`) |
| TypeScript / JavaScript | copy [`client/verify/js/sparrow-verify.ts`](client/verify/js/sparrow-verify.ts) (Node >= 16 or Bun, `node:crypto` only) |

### Go

```go
import "github.com/sarathsp06/sparrow/pkg/signature"

func handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	if err := signature.VerifyHMAC(body, r.Header, secret); err != nil {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}
	// or, with only the webhook's signing_public_key (hex):
	// err := signature.VerifyEd25519(body, r.Header, publicKeyHex)
}
```

### Python

```python
from sparrow_verify import verify_hmac, verify_ed25519, SignatureVerificationError

try:
    verify_hmac(raw_body, request.headers, secret)  # v1, shared secret
    # verify_ed25519(raw_body, request.headers, public_key_hex)  # v1a, public key only
except SignatureVerificationError:
    return Response(status=401)
```

### TypeScript / JavaScript

```ts
import { verifyHmac, verifyEd25519, SignatureVerificationError } from "./sparrow-verify";

try {
  verifyHmac(rawBody, req.headers, secret);  // v1, shared secret
  // verifyEd25519(rawBody, req.headers, publicKeyHex);  // v1a, public key only
} catch (err) {
  res.status(401).end();
}
```

Rolling your own? The rules that matter: sign-check the raw body bytes; base64-decode the part of the secret after `whsec_` for the HMAC key; hex-decode `signing_public_key` for Ed25519; split `webhook-signature` on spaces and check every entry with a matching prefix; compare digests in constant time; enforce the timestamp window.

## Configuration

All configuration is through environment variables.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string. Use `sslmode=require` (or stricter) in production |
| `SPARROW_ENCRYPTION_KEY` | Yes | — | 64-char hex (32-byte) master key for envelope encryption. Generate with `openssl rand -hex 32` |
| `SPARROW_API_KEY` | No | — | Require this key in the `X-API-Key` header (constant-time comparison) |
| `SPARROW_SERVE_UI` | No | `false` | Serve the embedded web dashboard |
| `SPARROW_ALLOW_PRIVATE_NETWORKS` | No | `false` | Allow private/loopback IPs as webhook targets (disables the SSRF guard) |
| `CORS_ALLOWED_ORIGINS` | No | — | Comma-separated allowlist of browser origins |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | — | OTLP endpoint for traces/metrics/logs (nothing is exported when unset) |

See [`okf/config/env-vars.md`](okf/config/env-vars.md) for the full list.

## Deployment

### Railway

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/KzfSI3?referralCode=otXr-t&utm_medium=integration&utm_source=template&utm_campaign=generic)

Deploy Sparrow and PostgreSQL on Railway with no infrastructure to manage: click the button, set `SPARROW_ENCRYPTION_KEY` (generate with `openssl rand -hex 32`), and deploy.

### Docker

Pre-built multi-arch images (linux/amd64, linux/arm64) are published on every release:

```bash
docker pull ghcr.io/sarathsp06/sparrow:latest
```

See [`deploy/docker-compose.yml`](deploy/docker-compose.yml) for the full example (Postgres plus Sparrow, health-checked startup).

### Kubernetes

The Helm chart at [`charts/sparrow/`](charts/sparrow/) ships hardened defaults: non-root, read-only root filesystem, dropped capabilities, seccomp, NetworkPolicy isolation, and no auto-mounted service-account token (details under [Security](#runtime-hardening-helm)).

```bash
helm install sparrow charts/sparrow/ \
  --set secrets.databaseURL="postgres://user:pass@your-db:5432/sparrow?sslmode=require" \
  --set secrets.existingSecret="sparrow-secrets"   # pull encryption key + API key from your secret manager
```

See [`charts/sparrow/values.yaml`](charts/sparrow/values.yaml) for every value and default, and [`okf/devops/helm-chart.md`](okf/devops/helm-chart.md) for the hardening reference.

## Documentation

The running server serves its own API reference: interactive docs at `/docs` (powered by
[Scalar](https://github.com/scalar/scalar)) and the OpenAPI document at `/openapi.yaml` and
`/openapi.json`. The contract is generated from Go via [Huma](https://github.com/danielgtaylor/huma).

The [`okf/`](okf/) knowledge bundle covers the architecture, concepts, data model, and operational details in depth.

## Contributing

Contributions are welcome. Open an issue to discuss larger changes before sending a PR.

```bash
git clone https://github.com/sarathsp06/sparrow.git
cd sparrow
make build-with-ui   # build server + embedded UI
make test            # run tests
make lint            # run linters
```

### Release Process

Releases are automated by CI (GoReleaser runs in GitHub Actions).

```bash
# 1) create a semantic version tag
git tag vX.Y.Z

# 2) push main + tags
git push origin main --tags
```

Notes:
- Use Conventional Commit prefixes (`feat:`, `fix:`, `docs:`, etc.) for clean autogenerated release notes.
- If a tag already exists remotely, create the next version tag and push that one instead.

See [`okf/architecture/overview.md`](okf/architecture/overview.md) for the package structure and dependency graph, and [CONTRIBUTING.md](CONTRIBUTING.md) for the full contributor workflow.

## License

[MIT](LICENSE)
