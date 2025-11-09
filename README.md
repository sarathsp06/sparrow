
<div align="center">
	<img src="./web/src/lib/assets/favicon.svg" alt="Sparrow" width="240" height="240" />
	<h1 style="font-family:monospace;font-weight:900;color:#222;">sparrow</h1>
	<p style="font-size:1.2em;color:#555;">Modern event-driven webhook delivery system with full observability</p>
</div>

---

## ✨ Introduction

Sparrow is a production-ready webhook delivery system built for developers who value reliability, observability, and developer experience. With a clean service architecture, modern web UI, and comprehensive monitoring, Sparrow makes webhook management effortless—whether you're building for scale or prototyping.

- **Modern stack:** Go backend with gRPC/Connect-RPC APIs, SvelteKit web UI, PostgreSQL, River queue
- **Built for reliability:** Automatic health tracking, intelligent retries, delivery metrics, and event reporting
- **Full observability:** OpenTelemetry tracing, health monitoring, delivery statistics, and real-time dashboards
- **Developer experience first:** Clean APIs, comprehensive web UI, simple setup, and easy local development
- **Production ready:** Namespace isolation, event type management, pagination, and robust error handling

Whether you're integrating webhooks, building event-driven systems, or need a reliable message queue with monitoring, Sparrow provides everything you need out of the box.

## 🚀 Features

### Core Webhook Management
- **Webhook Registration & Management** - Register, update, pause/resume webhooks
- **Event Type Registry** - Global event schema management with validation
- **Namespace Isolation** - Multi-tenant webhook management
- **Automatic Retries** - Intelligent retry logic with exponential backoff
- **Health Monitoring** - Real-time webhook health tracking and alerts

### Advanced Event Reporting
- **Event Reports** - Comprehensive event instance tracking with delivery statistics
- **Delivery Analytics** - Success/failure rates, response times, and delivery status
- **Event Filtering** - Search and filter events by type, namespace, and status
- **Pagination** - Efficient browsing of large event datasets
- **Export Capabilities** - JSON export of event data and delivery logs

### Modern Web Interface
- **Interactive Dashboard** - Real-time webhook and event management
- **Event Instance Browser** - Paginated event reports with detailed delivery stats
- **JSON Payload Viewer** - Expandable, formatted payload inspection
- **Health Status Indicators** - Visual webhook health monitoring
- **Responsive Design** - Mobile-friendly interface

### Observability & Monitoring
- **OpenTelemetry Integration** - Full distributed tracing support
- **Health Metrics** - Automatic webhook health calculation
- **Delivery Statistics** - Success rates, failure counts, response times
- **Time-series Data** - Historical performance tracking
- **Real-time Updates** - Live dashboard updates and notifications



## Usage & Makefile Commands

| Command           | Description                                      |
|-------------------|--------------------------------------------------|
| make docker-purge | Remove all Docker containers and volumes         |
| make docker-dev   | Start dev setup (Postgres, River) except app     |
| make migrate      | Run DB migrations (assumes docker-dev running)   |
| make run          | Run gRPC and Connect servers                     |
| make example      | Run example gRPC client                          |
| make run-web      | Run the web UI locally                           |

### Typical Workflow

1. `make docker-dev`      - Start database and queue in Docker
2. `make migrate`         - Run migrations on the database
3. `make run`             - Start gRPC and Connect servers
4. `make run-web`         - Launch the web UI for local development
5. `make example`         - Run example client to test API
6. `make docker-purge`     Clean up all Docker resources

### Other Commands

| Command           | Description                                      |
|-------------------|--------------------------------------------------|
| make build   | Build the gRPC server binary                          |
| make test    | Run Go tests                                          |
| make proto   | Regenerate gRPC/Connect code from proto
| make clean   | Remove built binaries and artifacts

Refer to the Makefile for more details and options.


## 🔧 Core API Methods

### Webhook Operations

```go
// Register a new webhook for specific events
RegisterWebhook(namespace, events[], url, headers?, timeout?, description?)

// Get webhooks by namespace
GetRegisteredWebhooks(namespace, webhook_id?, active_only?)

// List webhooks for specific events
ListRegisteredWebhooksByEvent(namespace, event, active_only?)

// Update existing webhook configuration
UpdateWebhook(namespace, webhook_id, events?, url?, headers?, timeout?, description?, active?)

// Pause webhook deliveries temporarily
PauseWebhook(webhook_id, namespace, reason?)

// Resume paused webhook deliveries  
ResumeWebhook(webhook_id, namespace)

// Unregister webhook completely
UnregisterWebhook(webhook_id, namespace)
```

### Event & Delivery Management

```go
// Push events to registered webhooks
PushEvent(namespace, event, payload, ttl?, metadata?)

// Get delivery status and details
GetWebhookDeliveryStatus(delivery_id, namespace)

// Resend failed or expired deliveries
ResendWebhook(delivery_id, namespace, force_resend?)

// Get delivery history with pagination
GetWebhookDeliveryHistory(webhook_id, namespace, limit?, offset?)

// List event instances with delivery statistics
ListEventReports(namespace, event_name?, limit?, offset?)

// Get detailed delivery statistics for specific event
GetEventDeliveryStats(event_id) → (webhook_count, successful, failed, pending)
```

### Event Type Management

```go
// Register a new event type (global registry)
RegisterEvent(name, description, schema?, metadata?, active?)

// List all registered event types
ListEvents(active_only?)

// Update event type information
UpdateEvent(name, description?, schema?, metadata?, active?)

// Delete an event type registration
DeleteEvent(name)

// Get specific event type details
GetEventByName(name)
```

#### Event Registration Features

- **Schema Validation** - JSON Schema definitions for event payloads
- **Metadata Support** - Custom key-value pairs for categorization and tagging
- **Active/Inactive States** - Enable/disable event types without deletion
- **Namespace Independent** - Global event registry shared across all namespaces
- **Unique Names** - Prevent duplicate event type registrations
- **Versioning Support** - Track event schema changes over time

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

## 📊 Monitoring & Dashboards

### Available Dashboards

- River job queue metrics via `http://0.0.0.0:8082/jobs`
- Web UI dashboard at `http://localhost:5173` (after running `make run-web`)


## Architecture

### Clean Service Layer

- **Service Layer**: Protocol-agnostic business logic in `/internal/webhooks/`
- **Transport Layer**: gRPC and Connect-RPC adapters in `/internal/grpc/` and `/internal/connect/`
- **Repository Layer**: Database access abstraction in `/internal/webhooks/`
- **Queue Layer**: River job processing in `/internal/webhook/queue/` and `/internal/webhook/workers/`

### Key Benefits

- **Namespace isolation** - Multi-tenant security
- **Protocol flexibility** - Same logic for gRPC and HTTP APIs
- **Comprehensive testing** - Service layer is easily testable

### Database Schema

- **Webhooks** - Registration data with headers, timeouts, and active status
- **Events** - Event instances with payloads and metadata
- **Event Registrations** - Global event type registry with schema validation
- **Webhook Health Events** - Delivery tracking with timestamps and response details
- **Webhook Health Timeseries** - Aggregated health metrics for performance analysis

## 🚀 Development Workflow

### Quick Start

```bash
# Start backend services
make docker-compose-up
make migrate-up
make run-grpc

# Start web interface (separate terminal)
make run-web
```

### Available Commands

- **`make run-grpc`** - Start gRPC server on port 50051
- **`make run-web`** - Start SvelteKit dev server on port 5173
- **`make migrate-up`** - Apply database migrations
- **`make migrate-down`** - Rollback database migrations
- **`make docker-compose-up`** - Start PostgreSQL and OpenTelemetry
- **`make buf-generate`** - Regenerate protobuf and gRPC code

### Testing

The system includes comprehensive test examples:

- **`examples/full_api_test.go`** - Complete API workflow test
- **`examples/test_event_management.sh`** - Event type management via Connect API
- **`examples/test_webhook_health.sh`** - Webhook health monitoring tests

## 📦 Production Deployment

### Docker Build

```bash
docker build -t httpqueue .
```

### Environment Configuration

Required environment variables for production:

- `DATABASE_URL` - PostgreSQL connection string
- `HTTP_PORT` - Web server port (default: 8080)
- `GRPC_PORT` - gRPC server port (default: 50051)
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OpenTelemetry collector endpoint

---

**httpqueue** provides a complete webhook delivery platform with modern web UI, comprehensive monitoring, and production-ready reliability for managing event-driven communication at scale.

---

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Run `make test` and `make proto` if needed
5. Submit a pull request

## License

MIT License - see LICENSE file for details.
