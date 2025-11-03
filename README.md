# sparrow

Minimal event-driven webhook delivery system.

## Usage & Makefile Commands

| Command           | Description                                      |
|-------------------|--------------------------------------------------|
| make docker-purge | Remove all Docker containers and volumes         |
| make docker-dev   | Start dev setup (Postgres, River) except app     |
| make migrate      | Run DB migrations (assumes docker-dev running)   |
| make run-connect  | Run gRPC and Connect servers                     |
| make example      | Run example gRPC client                          |
| make run-web      | Run the web UI locally                           |

### Typical Workflow

1. `make docker-dev`      | Start database and queue in Docker
2. `make migrate`         | Run migrations on the database
3. `make run-connect`     | Start gRPC and Connect servers
4. `make run-web`         | Launch the web UI for local development
5. `make example`         | Run example client to test API
6. `make docker-purge`    | Clean up all Docker resources

### Other Commands

| make build   | Build the gRPC server binary
| make test    | Run Go tests
| make proto   | Regenerate gRPC/Connect code from proto
| make clean   | Remove built binaries

Refer to the Makefile for more details and options.

```bash
# Test webhook operations
go run examples/grpc_client.go
go run examples/connect_client.go

# Test event management
./examples/test_event_management.sh
go run examples/event_management_client.go
```

See `proto/webhook.proto` for complete API definitions.

## Core Management Methods

### Webhook Operations

```go
// Get webhooks by namespace
GetRegisteredWebhooks(namespace, webhook_id?, active_only?)

// List webhooks for specific events
ListRegisteredWebhooksByEvent(namespace, event, active_only?)

// Pause webhook deliveries temporarily
PauseWebhook(webhook_id, namespace, reason?)

// Resume paused webhook deliveries  
ResumeWebhook(webhook_id, namespace)
```

### Delivery Management
```go
// Get delivery status
GetWebhookDeliveryStatus(delivery_id, namespace)

// Resend failed delivery
ResendWebhook(delivery_id, namespace, force_resend?)

// Get delivery history with pagination
GetWebhookDeliveryHistory(webhook_id, namespace, limit?, offset?)
```

### Event Type Management
```go
// Register a new event type (namespace-independent)
RegisterEvent(name, description, schema?, metadata?, active?)

// List all registered event types
ListEvents(active_only?)

// Update event type information
UpdateEvent(name, description?, schema?, metadata?, active?)

// Delete an event type registration
DeleteEvent(name)
```

#### Event Registration Features
- **Schema Validation** - JSON Schema definitions for event payloads
- **Metadata Support** - Custom key-value pairs for categorization
- **Active/Inactive States** - Enable/disable event types
- **Namespace Independent** - Global event registry across all namespaces
- **Unique Names** - Prevent duplicate event type registrations

### Webhook Health Management

```go
// Get health metrics for a specific webhook
GetWebhookHealth(webhook_id, namespace)

// List webhooks filtered by health status
ListWebhooksByHealth(health_status) // HEALTHY, DEGRADED, UNHEALTHY, UNKNOWN

// Get overall health summary across all namespaces
GetHealthSummary()
```

#### Health Monitoring Features

- **Automatic Health Tracking** - Real-time health status updates based on delivery metrics
- **Health States** - Healthy (>90% success), Degraded (80-90%), Unhealthy (<80% or 5+ consecutive failures), Unknown (no deliveries)
- **Comprehensive Metrics** - Success rate, response times, failure counts, last success/failure timestamps
- **Database Triggers** - Automatic health status updates when delivery metrics change
- **Health-based Filtering** - Query webhooks by health status for monitoring and alerting

#### Health Calculation Rules

- **Healthy**: Success rate ≥ 90% with at least 5 deliveries
- **Degraded**: Success rate 80-89% with at least 10 deliveries
- **Unhealthy**: 5+ consecutive failures or success rate < 80%
- **Unknown**: No delivery attempts yet

### Health Testing

```bash
# Test webhook health functionality
./examples/test_webhook_health.sh
go run examples/webhook_health_client.go
```

## Configuration

### Environment Variables
- `DATABASE_URL` - PostgreSQL connection string (required)
- `GRPC_PORT` - gRPC server port (default: 50051)
- `HTTP_PORT` - Connect-RPC HTTP server port (default: 8080)
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OpenTelemetry collector endpoint

### Example Configuration
```bash
DATABASE_URL="postgres://user:pass@localhost:5432/sparrow?sslmode=disable"
GRPC_PORT=50051
HTTP_PORT=8080
OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"
```

## Observability & Monitoring

### OpenTelemetry Integration
```bash
# Start observability stack
make obs-up                     # OTEL Collector + backends
```

### Monitoring Capabilities
- **Distributed tracing** - Request flows across services
- **Metrics collection** - Webhook registrations, deliveries, failures
- **Structured logging** - JSON logs with correlation IDs
- **Job queue visibility** - River queue status and metrics

### Available Dashboards
- River job queue metrics via `make grpc-jobs`  
- Database connections via `make grpc-db-shell`
- Application logs via `make grpc-logs`

## Architecture

### Clean Service Layer
- **Service Layer**: Protocol-agnostic business logic in `/internal/webhooks/`
- **Transport Layer**: gRPC and Connect-RPC adapters in `/internal/grpc/` and `/internal/connect/`
- **Repository Layer**: Database access abstraction in `/internal/webhooks/`
- **Queue Layer**: River job processing in `/internal/queue/` and `/internal/workers/`

### Key Benefits
- **Namespace isolation** - Multi-tenant security
- **Protocol flexibility** - Same logic for gRPC and HTTP APIs
- **Comprehensive testing** - Service layer is easily testable
- **Observability first** - Built-in tracing and metrics

---

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Run `make test` and `make proto` if needed
5. Submit a pull request

## License

MIT License - see LICENSE file for details.