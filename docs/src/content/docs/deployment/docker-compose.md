---
title: Docker Compose Deployment
description: Deploy Sparrow with Docker Compose
---

The simplest way to run Sparrow. No need to clone the repo -- download the compose file and start it.

## Quick Start

```bash
curl -O https://raw.githubusercontent.com/sarathsp06/sparrow/main/deploy/docker-compose.yml
echo "SPARROW_ENCRYPTION_KEYS=main=$(openssl rand -hex 32)" > .env
echo "SPARROW_ENCRYPTION_PRIMARY_KEY_ID=main" >> .env
docker compose up -d
```

On Windows PowerShell:

```powershell
"SPARROW_ENCRYPTION_KEYS=main=$(-join (1..32 | % { '{0:x2}' -f (Get-Random -Max 256) }))" | Out-File -Encoding ascii .env
"SPARROW_ENCRYPTION_PRIMARY_KEY_ID=main" | Out-File -Encoding ascii -Append .env
docker compose up -d
```

> [!IMPORTANT]
> `SPARROW_ENCRYPTION_KEYS` is never stored by Sparrow -- it only decrypts what
> it already encrypted. Writing it to `.env` (not passing it inline to a
> single command) is required: an inline `KEY=$(openssl rand -hex 32) docker
> compose up -d` generates a *new* random key on every invocation and
> permanently orphans anything encrypted under the previous one. Back up
> `.env` (or move the key into a secrets manager) before you rely on this
> instance.

The server is available at:
- **Web UI:** http://localhost:8080
- **REST API:** http://localhost:8080/v1

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
