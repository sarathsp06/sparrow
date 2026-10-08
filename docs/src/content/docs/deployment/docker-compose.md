---
title: Docker Compose (Local)
description: Try Sparrow locally with Docker Compose -- no clone, no secrets, no .env.
---

The fastest way to try Sparrow on your own machine. No clone, no secrets, no `.env` needed.

> [!NOTE]
> This Compose file is for **local evaluation**, not production. The API is
> open, the encryption key is a well-known all-zeros value, and the port is
> bound to `127.0.0.1` only. For a real deployment, see
> [Production Deployment](/sparrow/deployment/production/).

## Quick Start

```bash
curl -O https://raw.githubusercontent.com/sarathsp06/sparrow/main/deploy/docker-compose.yml
docker compose up -d
```

- **Web UI:** http://localhost:8080
- **REST API:** http://localhost:8080/v1
- **API docs:** http://localhost:8080/docs

SSRF protection is relaxed so webhooks can target services on your machine. Use `http://host.docker.internal:<port>` to reach the host from inside Docker (works on macOS, Windows, and Linux with the Compose file's `extra_hosts` mapping).

### Trying authentication locally

Uncomment `SPARROW_API_KEY` in the Compose file and restart:

```bash
docker compose up -d
```

The UI will show a **Sign in to Sparrow** prompt. Paste the key from the Compose file (`local-test-key`) to sign in. API calls now need `-H "X-API-Key: local-test-key"`. You can also try [access tokens and invites](/sparrow/deployment/access/).

To stop:

```bash
docker compose down        # stop containers
docker compose down -v     # stop and delete data
```

## Docker Image

Pre-built multi-arch images (linux/amd64, linux/arm64) are published to GitHub Container Registry on every release:

Latest Docker release: [`ghcr.io/sarathsp06/sparrow:latest`](https://github.com/sarathsp06/sparrow/pkgs/container/sparrow?tag=latest).

```bash
docker pull ghcr.io/sarathsp06/sparrow:latest
```

You can also pin to a specific version (image tags have no leading `v`; see the [releases](https://github.com/sarathsp06/sparrow/releases) for the latest):

```bash
docker pull ghcr.io/sarathsp06/sparrow:0.9.0
```

## Development (Build from Source)

The repo contains a `docker-compose.dev.yml` that builds from source. This is useful for development:

```bash
git clone https://github.com/sarathsp06/sparrow.git
cd sparrow
docker compose -f docker-compose.dev.yml up -d
```

Or build without Docker:

```bash
make build-with-ui
export DATABASE_URL=postgres://sparrow:sparrow@localhost:5432/sparrow?sslmode=disable
export SPARROW_ENCRYPTION_KEYS="main=$(openssl rand -hex 32)"
export SPARROW_ENCRYPTION_PRIMARY_KEY_ID=main
SPARROW_SERVE_UI=true ./build/server-*
```

The server runs migrations on startup, so there is no separate migrate step. Without a `.env`, `make run` and `make migrate` default to the `make dev-db` database and a dev-only all-zeros keyring, so the exports above are only needed when running the binary directly.

## Observability

Sparrow exports traces, metrics, and logs via OpenTelemetry (OTLP). Set `OTEL_EXPORTER_OTLP_ENDPOINT` to point to your collector (set `OTEL_EXPORTER_OTLP_PROTOCOL=grpc` and port `4317` for OTLP/gRPC; see [Configuration](/sparrow/getting-started/configuration/#observability)):

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://your-otel-collector:4318
```

For a traces-only backend such as Jaeger or Tempo, add `SPARROW_OTLP_SIGNALS=traces` so Sparrow does not try to upload metrics and logs it cannot accept.

Without a collector, Prometheus can scrape the same metrics from `http://sparrow:8080/metrics` (on by default; `SPARROW_METRICS_ENABLED=false` turns it off). See [Prometheus](/sparrow/getting-started/configuration/#prometheus).
