
# sparrow

Event-driven webhook delivery system with reliable processing, retry logic, and observability.

## Features

- Register webhooks for namespace/event pairs
- Push events to trigger webhooks automatically
- Track delivery status and retries
- gRPC and Connect-RPC (HTTP/JSON) APIs
- OpenTelemetry metrics and tracing
- Durable job queue (River) with PostgreSQL

## Quick Start

```bash
make grpc-up         # Start all services (Postgres, River, server)
go run examples/grpc_client.go   # Test gRPC API
make grpc-logs       # View logs
make grpc-down       # Stop services
```

## Development

```bash
make docker-dev      # Start only Postgres
make migrate-up      # Run DB migrations
make run             # Run server locally
make test            # Run tests
make proto           # Regenerate gRPC/Connect code
```

## API

- gRPC: port 50051
- HTTP/JSON (Connect): port 8080

See `examples/grpc_client.go` and `proto/webhook.proto` for usage.

## Configuration

- `DATABASE_URL` (Postgres connection)
- `GRPC_PORT` (default: 50051)
- `OTEL_EXPORTER_OTLP_ENDPOINT` (for tracing)

## Observability

- `make obs-up` to start Jaeger, Prometheus, Grafana, OTEL Collector

---