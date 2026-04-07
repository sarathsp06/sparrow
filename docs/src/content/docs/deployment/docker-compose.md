---
title: Docker Compose Deployment
description: Deploy Sparrow with Docker Compose
---

The simplest way to run Sparrow. No need to clone the repo -- save the compose file below, run `docker compose up -d`, and you're done.

## Quick Start

Create a `docker-compose.yml`:

```yaml title="docker-compose.yml"
services:
  postgres:
    image: postgres:15-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: sparrow
      POSTGRES_USER: sparrow
      POSTGRES_PASSWORD: sparrow
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U sparrow"]
      interval: 5s
      timeout: 3s
      retries: 10

  sparrow:
    image: ghcr.io/sarathsp06/sparrow:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "50051:50051"
    environment:
      DATABASE_URL: postgres://sparrow:sparrow@postgres:5432/sparrow?sslmode=disable
      SPARROW_SERVE_UI: "true"
      # Optional: require an API key for all requests
      # SPARROW_API_KEY: "change-me-to-a-secure-random-string"
      # Optional: explicit encryption key (auto-generated if omitted)
      # SPARROW_ENCRYPTION_KEY: "your-64-char-hex-key-from-openssl-rand-hex-32"
    depends_on:
      postgres: { condition: service_healthy }

volumes:
  postgres_data:
```

```bash
docker compose up -d
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

The repo root contains a `docker-compose.yml` that builds from source. This is useful for development:

```bash
git clone https://github.com/sarathsp06/sparrow.git
cd sparrow
docker compose up -d
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
