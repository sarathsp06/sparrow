# HTTPQueue

Event-driven webhook delivery system with reliable processing and retry logic.

## Overview

HTTPQueue allows you to:

- Register webhooks for namespace/event combinations
- Push events that automatically trigger all matching webhooks  
- Track delivery status with built-in retries
- Monitor performance with OpenTelemetry metrics and tracing

Built with Go, gRPC, PostgreSQL, and River Queue for durability and performance.

## Quick Start

### Using Docker (Recommended)

1. Start the system:
   ```bash
   make grpc-up
   ```

2. Test with example client:
   ```bash
   go run examples/grpc_client.go
   ```

3. View logs:
   ```bash
   make grpc-logs
   ```

### Local Development

1. Start PostgreSQL:
   ```bash
   make docker-dev
   ```

2. Run migrations:
   ```bash
   make migrate-up
   ```

3. Start server:
   ```bash
   make run
   ```

## Basic Usage

### Register a webhook:
```go
client.RegisterWebhook(ctx, &pb.RegisterWebhookRequest{
    Namespace: "user",
    Event:     "signup", 
    Url:       "https://api.example.com/webhooks/user-signup",
    Headers:   map[string]string{"Authorization": "Bearer token"},
    Timeout:   30,
})
```

### Push an event:
```go
client.PushEvent(ctx, &pb.PushEventRequest{
    Namespace:  "user",
    Event:      "signup",
    Payload:    `{"user_id": "12345", "email": "user@example.com"}`,
    TtlSeconds: 3600,
})
```

### Check delivery status:
```go
client.GetWebhookStatus(ctx, &pb.GetWebhookStatusRequest{
    Identifier: &pb.GetWebhookStatusRequest_WebhookId{
        WebhookId: "webhook-id",
    },
})
```

## Configuration

Set environment variables:

- `DATABASE_URL` - PostgreSQL connection (default: localhost)
- `GRPC_PORT` - gRPC server port (default: 50051)

### OpenTelemetry Configuration

HTTPQueue includes built-in OpenTelemetry support for distributed tracing and metrics:

- `OTEL_EXPORTER_OTLP_ENDPOINT` - OTLP endpoint (default: http://localhost:4318)
- `ENVIRONMENT` - Deployment environment (default: development)  
- `OTEL_TRACE_SAMPLE_RATE` - Trace sampling rate 0.0-1.0 (default: 1.0)

## Observability

Start the full observability stack:

```bash
make obs-up    # Starts Jaeger, Prometheus, Grafana, OTEL Collector
```

Access the UIs:
- **Grafana**: http://localhost:3000 (admin/admin) - Dashboards and metrics
- **Jaeger**: http://localhost:16686 - Distributed tracing
- **Prometheus**: http://localhost:9090 - Raw metrics

### Quick Start with Observability

```bash
# Start observability stack
make obs-up

# Start HTTPQueue with tracing
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 make run

# Run example client to generate traces
go run examples/grpc_client.go
```

## Development Commands

```bash
make help           # Show all available commands
make build          # Build the server
make test           # Run tests
make proto          # Generate gRPC code with buf
make proto-lint     # Lint protobuf files
make proto-format   # Format protobuf files
make migrate-create # Create new migration
```

## Architecture

- **gRPC API**: Webhook registration and event publishing
- **River Queue**: Reliable job processing with PostgreSQL
- **Event Workers**: Process events and create delivery jobs
- **Webhook Workers**: HTTP delivery with retry logic
- **PostgreSQL**: Stores webhooks, events, and delivery tracking