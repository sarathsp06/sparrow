---
title: Docker Compose Deployment
description: Deploy Sparrow with Docker Compose
---

The simplest way to run Sparrow. No need to clone the repo -- download the compose file and start it.

## Quick Start

```bash
curl -O https://raw.githubusercontent.com/sarathsp06/sparrow/main/deploy/docker-compose.yml
SPARROW_ENCRYPTION_KEY=$(openssl rand -hex 32) docker compose up -d
```

The server is available at:
- **Web UI:** http://localhost:8080
- **HTTP API (Connect-RPC):** http://localhost:8080
- **gRPC API:** localhost:50051

To stop:

```bash
docker compose down        # stop containers
docker compose down -v     # stop and delete data
```

## Docker Image

Pre-built multi-arch images (linux/amd64, linux/arm64) are published to GitHub Container Registry on every release:

```bash
docker pull ghcr.io/sarathsp06/sparrow:latest
```

You can also pin to a specific version:

```bash
docker pull ghcr.io/sarathsp06/sparrow:0.2.0
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
