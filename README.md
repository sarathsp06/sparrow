
# sparrow

Event-driven webhook delivery system with reliable processing, retry logic, and comprehensive management capabilities.

## Features

### Core Webhook Operations
- **Register/Unregister** webhooks for namespace/event pairs
- **Push events** to trigger webhooks automatically
- **List webhooks** with filtering by namespace and event
- **Track delivery status** and retry attempts

### Advanced Management
- **Pause/Resume webhooks** - Temporarily disable deliveries
- **Resend failed deliveries** - Retry with force option
- **Delivery history** - Paginated webhook delivery tracking
- **Namespace isolation** - Secure multi-tenant operations

### Architecture & APIs
- **Clean service layer** - Protocol-agnostic business logic
- **gRPC and Connect-RPC** (HTTP/JSON) APIs
- **OpenTelemetry** metrics and distributed tracing
- **Durable job queue** (River) with PostgreSQL backend
- **Comprehensive error handling** and validation

## Quick Start

```bash
# Full system with Docker Compose
make grpc-up                    # Start all services (Postgres, River, server)
make grpc-test                  # Run example client tests
make grpc-logs                  # View server logs
make grpc-down                  # Stop all services

# Check job queue status
make grpc-jobs                  # View River job queue
make grpc-db-shell             # Connect to PostgreSQL
```

## Development Workflow

```bash
# Local development
make docker-dev                 # Start only Postgres
make migrate                    # Run database migrations
go run main.go                  # Run server locally

# API testing
go run examples/grpc_client.go  # Test gRPC API
go run examples/connect_client.go  # Test Connect-RPC API

# Code generation and testing
make proto                      # Regenerate gRPC/Connect code
make test                       # Run tests
make build                      # Build binaries
```

## API Endpoints

### gRPC (Port 50051)
- `RegisterWebhook` - Register webhook for events
- `UnregisterWebhook` - Remove webhook registration  
- `PushEvent` - Trigger event processing
- `ListWebhooks` - List registered webhooks
- `GetWebhookStatus` - Check delivery status

### Connect-RPC HTTP/JSON (Port 8080)
- Same methods as gRPC but over HTTP with JSON payloads
- RESTful-style endpoints with Connect protocol

### Event Management Endpoints
Both gRPC and Connect-RPC support:
- `RegisterEvent` - Register new event types
- `ListEvents` - List all registered events
- `UpdateEvent` - Update event information
- `DeleteEvent` - Remove event registrations

### Testing & Examples
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
- **Service Layer**: Protocol-agnostic business logic in `/internal/services/`
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