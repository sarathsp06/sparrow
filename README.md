# HTTPQueue

Event-driven webhook delivery system with reliable processing and retry logic.

## Overview

HTTPQueue allows you to:

- Register webhooks for namespace/event combinations
- Push events that automatically trigger all matching webhooks  
- Track delivery status with built-in retries
- Monitor performance with OpenTelemetry metrics and tracing

Built with Go, gRPC/Connect-RPC, PostgreSQL, and River Queue for durability and performance.

## API Options

HTTPQueue provides **two API interfaces**:

1. **gRPC API** (port 50051): High-performance binary protocol for backend services
2. **Connect-RPC HTTP/JSON API** (port 8080): Web-friendly HTTP API compatible with browsers and curl

Both APIs provide identical functionality and can be used simultaneously.

## Quick Start

### Using Docker (Recommended)

1. Start the system:

   ```bash
   make grpc-up
   ```

2. Test gRPC API:

   ```bash
   go run examples/grpc_client.go
   ```

3. Test HTTP/JSON API:

   ```bash
   ./examples/test_connect_api.sh
   ```

4. View logs:

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

### gRPC API Examples

Register a webhook:

```go
client.RegisterWebhook(ctx, &pb.RegisterWebhookRequest{
    Namespace: "user",
    Event:     "signup", 
    Url:       "https://api.example.com/webhooks/user-signup",
    Headers:   map[string]string{"Authorization": "Bearer token"},
    Timeout:   30,
})
```

Push an event:

```go
client.PushEvent(ctx, &pb.PushEventRequest{
    Namespace:  "user",
    Event:      "signup",
    Payload:    `{"user_id": "12345", "email": "user@example.com"}`,
    TtlSeconds: 3600,
})
```

Check delivery status:

```go
client.GetWebhookStatus(ctx, &pb.GetWebhookStatusRequest{
    Identifier: &pb.GetWebhookStatusRequest_WebhookId{
        WebhookId: "webhook-id",
    },
})
```

### HTTP/JSON API Examples

Register a webhook:

```bash
curl -X POST http://localhost:8080/webhook.WebhookService/RegisterWebhook \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "user",
    "events": ["signup"],
    "url": "https://api.example.com/webhooks/user-signup",
    "headers": {"Authorization": "Bearer token"},
    "timeout": 30
  }'
```

Push an event:

```bash
curl -X POST http://localhost:8080/webhook.WebhookService/PushEvent \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "user",
    "event": "signup",
    "payload": "{\"user_id\": \"12345\", \"email\": \"user@example.com\"}",
    "ttlSeconds": 3600
  }'
```

Check webhook status:

```bash
curl -X POST http://localhost:8080/webhook.WebhookService/GetWebhookStatus \
  -H "Content-Type: application/json" \
  -d '{"webhookId": "your-webhook-id"}'
```

Health check:

```bash
curl http://localhost:8080/health
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