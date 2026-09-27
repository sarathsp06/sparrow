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

You can also pin to a specific version:

```bash
docker pull ghcr.io/sarathsp06/sparrow:0.5.6
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
export DATABASE_URL=postgres://user:pass@localhost:5432/sparrow?sslmode=disable
make migrate
SPARROW_SERVE_UI=true ./build/server-*
```

## Observability

Sparrow exports traces, metrics, and logs via OpenTelemetry (OTLP). Set `OTEL_EXPORTER_OTLP_ENDPOINT` to point to your collector:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://your-otel-collector:4318
```
