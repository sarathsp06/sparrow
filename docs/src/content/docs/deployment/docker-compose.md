---
title: Docker Compose Deployment
description: Deploy Sparrow with Docker Compose
---

The simplest way to run Sparrow. Docker Compose starts PostgreSQL, runs database migrations, and launches the server with the embedded web UI.

## Quick Start

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

## Minimal Docker Compose

If you want a minimal compose file for your own setup:

```yaml title="docker-compose.yml"
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: sparrow
      POSTGRES_USER: sparrow
      POSTGRES_PASSWORD: sparrow
    ports: ["5432:5432"]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U sparrow"]
      interval: 5s
      timeout: 5s
      retries: 5

  sparrow:
    image: sparrow:latest
    environment:
      DATABASE_URL: postgres://sparrow:sparrow@postgres:5432/sparrow?sslmode=disable
      SPARROW_SERVE_UI: "true"
    ports: ["8080:8080", "50051:50051"]
    depends_on:
      postgres: { condition: service_healthy }
```

## Full Docker Compose

The repository includes a comprehensive `docker-compose.yml` with:

- **PostgreSQL** -- Database with health checks
- **Migrate** -- Runs migrations automatically on startup (one-shot container)
- **Sparrow** -- Server with embedded UI
- **OTel Collector** -- OpenTelemetry sidecar for traces, metrics, and logs

### Startup Order

```
postgres (healthy) -> migrate (exits) -> sparrow (starts)
```

The migrate container runs once to apply all database migrations, then exits. Sparrow only starts after migrations complete successfully.

## Build from Source

If you prefer building from source instead of Docker:

```bash
# Build with embedded UI
make build-with-ui

# Start infrastructure (or provide your own Postgres)
export DATABASE_URL=postgres://user:pass@localhost:5432/sparrow?sslmode=disable

# Run migrations
./build/server-* migrate  # or: make migrate

# Start the server
SPARROW_SERVE_UI=true ./build/server-*
```

## Observability

Sparrow exports traces, metrics, and logs via OpenTelemetry (OTLP). Set `OTEL_EXPORTER_OTLP_ENDPOINT` to enable:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
```

The included `docker-compose.yml` ships with an OTel Collector sidecar. To use your own collector, point the environment variable to your OTLP endpoint.

## Dockerfile

The Sparrow Dockerfile uses a 3-stage build:

1. **`frontend`** (`node:22-alpine`) -- Builds the SvelteKit UI with static adapter
2. **`builder`** (`golang:1.25-alpine`) -- Compiles Go binaries with embedded UI
3. **Final** (`distroless/static-debian12:nonroot`) -- Minimal runtime image

The final image exposes ports `50051` (gRPC) and `8080` (HTTP/UI).
