# HTTPQueue - AI Development Guide

## Architecture Overview

HTTPQueue is an event-driven webhook delivery system with these key components:

- **gRPC API Server** (`internal/grpc/webhook_server.go`): Handles webhook registration and event publishing
- **River Queue Workers** (`internal/workers/`): Processes events and delivers webhooks asynchronously  
- **PostgreSQL**: Stores webhook registrations, events, and delivery tracking
- **Event Flow**: Register webhooks → Push events → Workers process & deliver → Track status

## Critical Development Patterns

### River Queue Job System
Jobs are defined in `internal/jobs/` and processed by workers in `internal/workers/`:

```go
// Event processing triggers webhook deliveries
type EventArgs struct {
    EventID    string
    Namespace  string  
    Event      string
    Payload    string
    TTLSeconds int64
    // ...
}

// Webhook delivery with retry logic
type WebhookArgs struct {
    DeliveryID string
    WebhookID  string
    URL        string
    Headers    map[string]string
    Payload    string
    ExpiresAt  time.Time
    // ...
}
```

Workers implement `river.Worker[JobArgs]` interface. The `EventProcessingWorker` queries registered webhooks and creates `WebhookArgs` jobs for delivery.

### Database Schema Convention
Three core tables follow a clear pattern:
- `webhook_registrations`: Store webhook configs with multiple events per webhook
- `event_records`: Track pushed events with TTL/expiry
- `webhook_deliveries`: Track delivery attempts with status and retry logic

### Project Structure Rules
- `internal/`: Business logic organized by domain (webhooks, workers, jobs, queue)
- `proto/`: gRPC definitions with generated `.pb.go` files
- `cmd/`: Application entry points (main server, migration utility)
- `examples/`: Working client code demonstrating API usage

## Essential Workflows

### Development Setup
```bash
# Quick start with Docker
make grpc-up                    # Start full system
make grpc-test                  # Run example client
make grpc-logs                  # View logs
make grpc-down                  # Stop system

# Local development  
make docker-dev                 # Start just PostgreSQL
go run main.go                  # Run server locally
go run examples/grpc_client.go  # Test client
```

### Database Management
```bash
# Connect to DB in development
make grpc-db-shell

# View recent jobs
make grpc-jobs

# Database lives in docker-compose.grpc.yml with River extensions
```

### Protobuf Development
```bash
make proto  # Regenerate gRPC code from proto/webhook.proto
```

## Key Implementation Details

### Queue Configuration
The queue manager (`internal/queue/manager.go`) sets up:
- **events queue**: Processes incoming events (5 workers)
- **webhooks queue**: Delivers HTTP requests (8 workers)  
- **default queue**: General processing (10 workers)

### Error Handling Pattern
Workers return errors for retries, update delivery status in database:
```go
// In WebhookWorker - non-2xx responses trigger retries
if resp.StatusCode >= 200 && resp.StatusCode < 300 {
    webhookRepo.UpdateDeliveryStatus(ctx, args.DeliveryID, webhooks.StatusSuccess, ...)
    return nil  // Success
}
return fmt.Errorf("webhook delivery failed: %s", errorMessage)  // Retry
```

### Event-Driven Flow
1. `RegisterWebhook` stores webhook config for namespace/event pairs
2. `PushEvent` creates `EventArgs` job in "events" queue
3. `EventProcessingWorker` finds matching webhooks, creates `WebhookArgs` jobs
4. `WebhookWorker` delivers HTTP requests with status tracking

### Configuration Sources
- Environment: `DATABASE_URL`, `GRPC_PORT` 
- Defaults in `main.go`: PostgreSQL localhost, gRPC port 50051
- Docker configs: `docker-compose.grpc.yml` (full system), `docker-compose.dev.yml` (DB only)

## Testing & Debugging

- Test via `examples/grpc_client.go` - complete workflow demonstration
- Monitor jobs via River UI (http://localhost:8080 when using Docker)
- Database inspection via `make grpc-db-shell` or pgAdmin (localhost:8081)
- All workers use structured logging with request IDs for tracing

When adding features, follow the established patterns: define job types in `internal/jobs/`, implement workers in `internal/workers/`, extend gRPC service in `internal/grpc/webhook_server.go`.