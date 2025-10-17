# HTTP Queue - Event-Driven Webhook System

A robust, event-driven webhook delivery system built with Go, River Queue, PostgreSQL, and gRPC. The system allows you to register webhooks for specific namespace/event combinations, push events that trigger registered webhooks, and track delivery status with comprehensive retry logic.

## Features

- **🎯 Event-driven Architecture**: Register webhooks for namespace/event pairs, then push events to trigger deliveries
- **🔄 Reliable Delivery**: Built on River Queue with PostgreSQL for durability and retry logic
- **📊 Delivery Tracking**: Complete visibility into webhook delivery status, attempts, and failures
- **⚙️ Configurable**: Flexible timeout, retry, and TTL settings per webhook
- **🌐 gRPC API**: Modern, efficient API for webhook management and event pushing
- **🐳 Docker Ready**: Complete Docker Compose setup for easy deployment

## Architecture

The system follows an event-driven pattern:

1. **Register** webhooks for specific namespace/event combinations
2. **Push** events to trigger all registered webhooks for that namespace/event
3. **Track** delivery status and retry failed deliveries automatically

### Core Components

- **gRPC Server**: Handles webhook registration, event pushing, and status queries
- **Event Processing Worker**: Processes events and creates webhook delivery jobs
- **Webhook Worker**: Handles HTTP delivery with status tracking and retries
- **PostgreSQL Database**: Stores webhook registrations, events, and delivery records
- **River Queue**: Manages job processing with reliability and retry logic

## Quick Start

### Prerequisites

- Go 1.24+
- PostgreSQL 13+
- Docker & Docker Compose (optional)

### Using Docker Compose

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd httpqueue
   ```

2. Start the services:
   ```bash
   docker-compose up -d
   ```

3. Run the example client:
   ```bash
   go run examples/grpc_client.go
   ```

### Manual Setup

1. Start PostgreSQL and create a database:
   ```bash
   createdb riverqueue
   ```

2. Run database migrations:
   ```bash
   go run migrations/setup.go
   ```

3. Start the server:
   ```bash
   go run main.go
   ```

4. Test with the example client:
   ```bash
   go run examples/grpc_client.go
   ```

## API Reference

### gRPC Service Methods

#### RegisterWebhook
Register a webhook URL for specific namespace/event combinations.

```protobuf
rpc RegisterWebhook(RegisterWebhookRequest) returns (RegisterWebhookResponse);
```

**Example:**
```go
client.RegisterWebhook(ctx, &pb.RegisterWebhookRequest{
    Namespace: "user",
    Event:     "signup",
    Url:       "https://api.example.com/webhooks/user-signup",
    Method:    "POST",
    Headers: map[string]string{
        "Authorization": "Bearer secret-token",
        "X-Event-Type":  "user-signup",
    },
    Timeout:     30,
    Active:      true,
    Description: "User signup notifications",
})
```

#### PushEvent
Push an event that triggers all registered webhooks for the namespace/event.

```protobuf
rpc PushEvent(PushEventRequest) returns (PushEventResponse);
```

**Example:**
```go
client.PushEvent(ctx, &pb.PushEventRequest{
    Namespace:  "user",
    Event:      "signup",
    Payload:    `{"user_id": "12345", "email": "user@example.com"}`,
    TtlSeconds: 3600, // 1 hour
    Metadata: map[string]string{
        "source": "api",
        "region": "us-east-1",
    },
})
```

#### GetWebhookStatus
Check the delivery status of webhooks.

```protobuf
rpc GetWebhookStatus(GetWebhookStatusRequest) returns (GetWebhookStatusResponse);
```

**Example:**
```go
// Get status by webhook ID
client.GetWebhookStatus(ctx, &pb.GetWebhookStatusRequest{
    Identifier: &pb.GetWebhookStatusRequest_WebhookId{
        WebhookId: "webhook-id-here",
    },
})

// Get status by event ID  
client.GetWebhookStatus(ctx, &pb.GetWebhookStatusRequest{
    Identifier: &pb.GetWebhookStatusRequest_EventId{
        EventId: "event-id-here",
    },
})
```

#### ListWebhooks
List all registered webhooks for a namespace.

```protobuf
rpc ListWebhooks(ListWebhooksRequest) returns (ListWebhooksResponse);
```

#### UnregisterWebhook
Remove a webhook registration.

```protobuf
rpc UnregisterWebhook(UnregisterWebhookRequest) returns (UnregisterWebhookResponse);
```

## Database Schema

The system uses several PostgreSQL tables:

### webhook_registrations
Stores webhook registration details:
- `id`: Unique webhook identifier
- `namespace`: Event namespace (e.g., "user", "order")
- `event`: Event type (e.g., "signup", "created")
- `url`: Webhook endpoint URL
- `method`: HTTP method (POST, PUT, etc.)
- `headers`: Custom HTTP headers (JSONB)
- `timeout`: Request timeout in seconds
- `active`: Whether the webhook is active
- `description`: Human-readable description

### event_records
Stores event history:
- `id`: Unique event identifier
- `namespace`: Event namespace
- `event`: Event type
- `payload`: Event data (JSON)
- `ttl_seconds`: Time-to-live for webhook retries
- `metadata`: Additional event metadata (JSONB)

### webhook_deliveries
Tracks webhook delivery attempts:
- `id`: Unique delivery identifier
- `webhook_id`: Associated webhook
- `event_id`: Associated event
- `status`: Delivery status (pending, success, failed, etc.)
- `attempt_count`: Number of delivery attempts
- `max_attempts`: Maximum retry attempts
- `response_code`: HTTP response code
- `response_body`: HTTP response body
- `error_message`: Error details if failed

## Configuration

### Environment Variables

- `DATABASE_URL`: PostgreSQL connection string (default: `postgres://localhost/riverqueue?sslmode=disable`)
- `GRPC_PORT`: gRPC server port (default: `50051`)

### Webhook Configuration

Each webhook can be configured with:

- **Timeout**: Request timeout (1-300 seconds)
- **Headers**: Custom HTTP headers
- **Method**: HTTP method (GET, POST, PUT, PATCH, DELETE)
- **Active Status**: Enable/disable webhook
- **Description**: Human-readable description

### Event Configuration

Events support:

- **TTL**: How long to retry failed webhooks (seconds)
- **Metadata**: Additional context data
- **Payload**: JSON event data

## Examples

### Complete Workflow Example

```go
// 1. Register webhook for user signups
registerResp, err := client.RegisterWebhook(ctx, &pb.RegisterWebhookRequest{
    Namespace: "user",
    Event:     "signup",
    Url:       "https://api.example.com/hooks/user-signup",
    Method:    "POST",
    Headers: map[string]string{
        "Authorization": "Bearer your-token",
    },
    Timeout: 30,
    Active:  true,
})

// 2. Push a user signup event
eventResp, err := client.PushEvent(ctx, &pb.PushEventRequest{
    Namespace:  "user",
    Event:      "signup",
    Payload:    `{"user_id": "12345", "email": "new-user@example.com"}`,
    TtlSeconds: 3600,
})

// 3. Check delivery status
statusResp, err := client.GetWebhookStatus(ctx, &pb.GetWebhookStatusRequest{
    Identifier: &pb.GetWebhookStatusRequest_WebhookId{
        WebhookId: registerResp.WebhookId,
    },
})
```

### Multiple Webhooks for Same Event

You can register multiple webhooks for the same namespace/event:

```go
// Primary webhook
client.RegisterWebhook(ctx, &pb.RegisterWebhookRequest{
    Namespace: "order",
    Event:     "created",
    Url:       "https://api.primary.com/orders",
    // ...
})

// Analytics webhook  
client.RegisterWebhook(ctx, &pb.RegisterWebhookRequest{
    Namespace: "order", 
    Event:     "created",
    Url:       "https://analytics.company.com/track",
    // ...
})

// One event triggers both webhooks
client.PushEvent(ctx, &pb.PushEventRequest{
    Namespace: "order",
    Event:     "created", 
    Payload:   `{"order_id": "12345", "total": 99.99}`,
})
```

## Development

### Building

```bash
# Build main server
go build -o httpqueue .

# Build example client
go build -o client examples/grpc_client.go

# Build with Docker
docker build -t httpqueue .
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/webhooks/
```

### Generating Protobuf

```bash
# Generate Go code from proto files
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/webhook.proto
```

## Deployment

### Docker Compose

The provided `docker-compose.yml` includes:

- **httpqueue**: Main application server
- **postgres**: PostgreSQL database with River extensions
- **adminer**: Database admin interface (optional)

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f httpqueue

# Stop services
docker-compose down
```

### Production Considerations

1. **Database**: Use managed PostgreSQL with connection pooling
2. **Monitoring**: Add metrics collection (Prometheus/Grafana)
3. **Logging**: Configure structured logging with appropriate levels
4. **Security**: Use TLS for gRPC, secure webhook endpoints
5. **Scaling**: Run multiple instances behind a load balancer
6. **Backup**: Regular database backups including job state

## Monitoring & Observability

### Key Metrics to Monitor

- **Webhook Registration Rate**: New webhooks registered per minute
- **Event Processing Rate**: Events processed per second
- **Delivery Success Rate**: Percentage of successful webhook deliveries
- **Average Delivery Time**: Time from event to successful delivery
- **Failed Delivery Count**: Number of failed deliveries requiring retry
- **Queue Depth**: Number of pending jobs in each queue

### Health Checks

The system provides several health check endpoints:

- **Database Connectivity**: Verify PostgreSQL connection
- **Queue Processing**: Confirm River workers are processing jobs
- **gRPC Service**: Verify API responsiveness

## Troubleshooting

### Common Issues

1. **Connection Refused**: Check PostgreSQL is running and accessible
2. **Jobs Not Processing**: Verify River workers are started
3. **Webhook Timeouts**: Check network connectivity and increase timeout
4. **High Memory Usage**: Review job queue sizes and processing rates

### Debug Commands

```bash
# Check database connectivity
psql $DATABASE_URL -c "SELECT 1"

# View recent jobs
# (Use database client to query river_job table)

# Check webhook delivery logs
docker-compose logs httpqueue | grep "webhook-worker"
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details.