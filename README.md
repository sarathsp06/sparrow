[![CI](https://github.com/sarathsp06/sparrow/actions/workflows/ci.yml/badge.svg)](https://github.com/sarathsp06/sparrow/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/ghcr.io-sarathsp06%2Fsparrow-blue?logo=docker)](https://github.com/sarathsp06/sparrow/pkgs/container/sparrow)
[![Docs](https://img.shields.io/badge/docs-GitHub%20Pages-blue)](https://sarathsp06.github.io/sparrow)

<p align="center">
  <img src="./web/src/lib/assets/favicon.svg" alt="Sparrow" width="120" height="120" />
</p>

# Sparrow

Self-hosted webhook delivery platform with async fan-out, retries, health tracking, and observability. Built for teams that need reliable outbound webhooks without depending on a third-party service.

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/KzfSI3?referralCode=otXr-t&utm_medium=integration&utm_source=template&utm_campaign=generic)

## Features

### Delivery & Reliability
- **Event-driven fan-out** -- push one event, deliver to all matching subscriptions
- **At-least-once delivery** -- configurable retries with exponential backoff
- **Idempotent event ingestion** -- optional idempotency keys to prevent duplicate processing
- **Per-webhook rate limiting** -- leaky bucket algorithm with HTTP 429 Retry-After parsing
- **10-category error classification** -- DNS, TLS, timeout, connection refused, rate limited, and more -- each with retryability flags
- **Bulk operations** -- deterministic snapshot-based batch re-push and retry (up to 10K items)

### Security
- **Dual webhook signing** -- every delivery is signed with both HMAC-SHA256 and Ed25519 ([Standard Webhooks](https://www.standardwebhooks.com/) format)
- **Envelope encryption at rest** -- AES-256-GCM with per-record data encryption keys for webhook secrets and sensitive headers
- **SSRF protection** -- blocks private/loopback/metadata IPs, validates redirect targets
- **Optional API key auth** -- constant-time comparison, HTTP + gRPC support

### Developer Experience
- **Payload transformation** -- Go templates per subscription (50+ functions) to reshape payloads for different consumers
- **Soft schema validation** -- warnings not errors; events are always accepted and stored
- **Dual-protocol API** -- gRPC on `:50051` and Connect-RPC (HTTP/JSON) on `:8080`
- **Web dashboard** -- embedded SvelteKit UI for managing webhooks, events, deliveries, and health
- **Health tracking** -- per-webhook state machine (healthy/degraded/unhealthy) with rolling summaries

### Operations
- **PostgreSQL only** -- no Redis, no message broker, no external dependencies beyond one database
- **OpenTelemetry** -- traces, metrics, and structured logs with job-level trace propagation
- **Helm chart** -- security-hardened Kubernetes deployment (NetworkPolicy, read-only rootfs, non-root, seccomp)
- **One-click deploy** -- Railway, Docker Compose, or any container platform

## Quick Start

Download [`deploy/docker-compose.yml`](deploy/docker-compose.yml) and start it:

```bash
curl -O https://raw.githubusercontent.com/sarathsp06/sparrow/main/deploy/docker-compose.yml
SPARROW_ENCRYPTION_KEY=$(openssl rand -hex 32) docker compose up -d
```

Open **http://localhost:8080** for the web UI.

### Send your first event

```bash
# Register an event type
curl -X POST http://localhost:8080/webhook.EventService/RegisterEvent \
  -H "Content-Type: application/json" \
  -d '{"name": "order.created", "description": "New order", "active": true}'

# Register a webhook (subscription is created automatically)
curl -X POST http://localhost:8080/webhook.WebhookService/RegisterWebhook \
  -H "Content-Type: application/json" \
  -d '{"namespace": "default", "url": "https://httpbin.org/post", "events": ["order.created"], "active": true}'

# Push an event -- Sparrow fans out and delivers
curl -X POST http://localhost:8080/webhook.EventService/PushEvent \
  -H "Content-Type: application/json" \
  -d '{"namespace": "default", "event": "order.created", "payload": {"order_id": "ord_123", "amount": 99.99}}'
```

Check delivery status in the web UI at **Deliveries**, or query the API:

```bash
curl -X POST http://localhost:8080/webhook.DeliveryService/ListDeliveries \
  -H "Content-Type: application/json" \
  -d '{"namespace": "default", "limit": 5}'
```

## Use Cases

- **SaaS webhook notifications** -- notify customer endpoints when resources change
- **Internal event bus** -- fan out domain events to downstream services over HTTP
- **Reliability layer** -- add retries, health tracking, and observability to existing webhook flows
- **Development and testing** -- inspect deliveries, replay failed events, test payload transforms

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

Events are persisted before delivery. The [River](https://riverqueue.com) job queue provides at-least-once delivery with configurable retries (default: 3 attempts, 60s backoff). Failures are classified into 10 categories -- retryable (5xx, timeout, connection refused, network error, rate limited) and non-retryable (4xx, DNS, TLS) -- so you know *why* a delivery failed, not just *that* it failed. Every delivery is dual-signed with HMAC-SHA256 and Ed25519 using the [Standard Webhooks](https://www.standardwebhooks.com/) format. Webhook secrets and sensitive headers are envelope-encrypted at rest using AES-256-GCM with per-record data encryption keys.

See the [architecture reference](docs/src/content/docs/reference/architecture.md) for the full pipeline design, error classification, and health state machine.

## Verifying Webhook Signatures

Every delivery includes three [Standard Webhooks](https://www.standardwebhooks.com/) headers:

| Header | Example |
|--------|---------|
| `webhook-id` | `msg_abc123-def456` |
| `webhook-timestamp` | `1716048000` (Unix seconds) |
| `webhook-signature` | `v1,K7gNU3sdo+OL...` or `v1a,RjB2mN...` |

The signed message is always: `{webhook-id}.{webhook-timestamp}.{raw request body}` -- the exact bytes of the JSON body, not re-serialized.

The `webhook-signature` prefix tells you which algorithm was used:
- `v1,` -- HMAC-SHA256 (symmetric, requires the shared webhook secret)
- `v1a,` -- Ed25519 (asymmetric, requires only the public key from the API)

### Verifying HMAC-SHA256 (`v1,`)

```python
import hmac, hashlib, base64

def verify_hmac(body: bytes, secret: str, headers: dict) -> bool:
    msg_id = headers["webhook-id"]
    timestamp = headers["webhook-timestamp"]
    signature = headers["webhook-signature"]  # "v1,<base64>"

    # Extract the base64 portion after "v1,"
    expected_b64 = signature.removeprefix("v1,")

    # Reconstruct the signed message
    message = f"{msg_id}.{timestamp}.{body.decode()}"

    # Compute HMAC-SHA256
    computed = hmac.new(secret.encode(), message.encode(), hashlib.sha256).digest()
    computed_b64 = base64.b64encode(computed).decode()

    return hmac.compare_digest(computed_b64, expected_b64)
```

```go
func VerifyHMAC(body []byte, secret, msgID, timestamp, signatureHeader string) bool {
    // Extract "v1,<base64>" -> "<base64>"
    b64Sig, _ := strings.CutPrefix(signatureHeader, "v1,")

    message := msgID + "." + timestamp + "." + string(body)
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(message))
    expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

    return hmac.Equal([]byte(expected), []byte(b64Sig))
}
```

### Verifying Ed25519 (`v1a,`)

The public key is returned in the `signing_public_key` field when you register or retrieve a webhook.

```python
import base64
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey

def verify_ed25519(body: bytes, public_key_b64: str, headers: dict) -> bool:
    msg_id = headers["webhook-id"]
    timestamp = headers["webhook-timestamp"]
    signature = headers["webhook-signature"]  # "v1a,<base64>"

    sig_bytes = base64.b64decode(signature.removeprefix("v1a,"))
    pub_key = Ed25519PublicKey.from_public_bytes(base64.b64decode(public_key_b64))

    message = f"{msg_id}.{timestamp}.{body.decode()}".encode()

    try:
        pub_key.verify(sig_bytes, message)
        return True
    except Exception:
        return False
```

```go
func VerifyEd25519(body []byte, publicKeyB64, msgID, timestamp, signatureHeader string) bool {
    b64Sig, _ := strings.CutPrefix(signatureHeader, "v1a,")
    sig, _ := base64.StdEncoding.DecodeString(b64Sig)
    pubKey, _ := base64.StdEncoding.DecodeString(publicKeyB64)

    message := []byte(msgID + "." + timestamp + "." + string(body))
    return ed25519.Verify(ed25519.PublicKey(pubKey), message, sig)
}
```

### Replay Protection

Always validate the `webhook-timestamp` header to prevent replay attacks. Reject deliveries where the timestamp is more than 5 minutes from your server's current time.

## Configuration

All configuration is via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | -- | PostgreSQL connection string |
| `SPARROW_SERVE_UI` | No | `false` | Serve the embedded web dashboard |
| `SPARROW_API_KEY` | No | -- | Require this key in `X-API-Key` header |
| `SPARROW_ENCRYPTION_KEY` | Yes | -- | 64-char hex key for envelope encryption of secrets. Generate with `openssl rand -hex 32` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | -- | OTLP endpoint for traces/metrics/logs |

See the [configuration reference](docs/src/content/docs/getting-started/configuration.md) for the full list.

## Deployment

### Railway

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/KzfSI3?referralCode=otXr-t&utm_medium=integration&utm_source=template&utm_campaign=generic)

Deploy Sparrow + PostgreSQL on Railway with zero infrastructure. See the [Railway deployment guide](https://sarathsp06.github.io/sparrow/deployment/railway/) for setup steps.

### Docker

Pre-built multi-arch images (linux/amd64, linux/arm64) are published on every release:

```bash
docker pull ghcr.io/sarathsp06/sparrow:latest
```

See [Docker Compose deployment guide](https://sarathsp06.github.io/sparrow/deployment/docker-compose/) for details.

### Kubernetes

A Helm chart is included at [`charts/sparrow/`](charts/sparrow/):

```bash
helm install sparrow charts/sparrow/ \
  --set secrets.databaseURL="postgres://user:pass@your-db:5432/sparrow?sslmode=require"
```

See [Kubernetes deployment guide](https://sarathsp06.github.io/sparrow/deployment/kubernetes/) for all chart values and examples.

## Documentation

**[sarathsp06.github.io/sparrow](https://sarathsp06.github.io/sparrow)**

- [Getting Started](https://sarathsp06.github.io/sparrow/getting-started/installation/) -- installation and quickstart
- [API Reference](https://sarathsp06.github.io/sparrow/reference/api/) -- all RPCs and message types
- [Architecture](https://sarathsp06.github.io/sparrow/reference/architecture/) -- system design overview

API reference docs are generated from [`webhook.proto`](webhook.proto) using [proto2astro](https://github.com/sarathsp06/proto2astro).

In-repo docs: [Configuration](docs/src/content/docs/getting-started/configuration.md) | [Architecture](docs/src/content/docs/reference/architecture.md) | [Kubernetes](docs/src/content/docs/deployment/kubernetes.md) | [webhook.proto](webhook.proto)

## Contributing

Contributions are welcome. Please open an issue to discuss larger changes before submitting a PR.

```bash
git clone https://github.com/sarathsp06/sparrow.git
cd sparrow
make build-with-ui   # build server + embedded UI
make test            # run tests
make lint            # run linters
```

See the [architecture reference](docs/src/content/docs/reference/architecture.md) for the package structure and dependency graph.

## License

[MIT](LICENSE)
