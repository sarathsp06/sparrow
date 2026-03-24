# Sparrow -- Configuration Reference

All configuration is done via environment variables. No config files needed.

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | -- | PostgreSQL connection string |
| `SPARROW_SERVE_UI` | No | `false` | Serve the embedded web dashboard on `:8080` |
| `ENVIRONMENT` | No | -- | `development` or `production` (affects logging/OTel) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | -- | OTLP HTTP endpoint for traces, metrics, and logs |
| `CORS_ALLOWED_ORIGINS` | No | -- | Allowed CORS origins for Connect-RPC |
| `PUBLIC_API_URL` | No | `/` | API base URL for the frontend (dev only) |

---

## Deployment Guide

### Docker Compose

The simplest way to run Sparrow. This starts PostgreSQL, runs migrations, and launches the server with the embedded web UI.

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

### Build from Source

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

### Minimal Docker Compose

If you want a minimal compose file for your own setup:

```yaml
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

---

## Observability

Sparrow exports traces, metrics, and logs via OpenTelemetry (OTLP). Set `OTEL_EXPORTER_OTLP_ENDPOINT` to enable.

The included `docker-compose.yml` ships with an OTel Collector sidecar. To use your own collector or observability stack, point the env var to your OTLP endpoint:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
```
