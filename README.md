
<div align="center">
	<img src="./web/src/lib/assets/favicon.svg" alt="Sparrow" width="240" height="240" />
	<h1 style="font-family:monospace;font-weight:900;color:#222;">sparrow</h1>
	<p style="font-size:1.2em;color:#555;">Modern event-driven webhook delivery system with full observability</p>
</div>

---

## ✨ What is Sparrow?

Sparrow is a high-performance, production-ready webhook delivery system that acts as a reliable intermediary between your application and external webhook consumers. It solves the common challenges of webhook delivery at scale: reliability, observability, retry logic, and delivery tracking.

### Core Problem It Solves

When applications need to notify external systems about events (user signups, order completions, payment processing), direct HTTP calls are fragile. Network failures, timeouts, and downstream service issues can cause lost notifications. Sparrow provides:

- **Guaranteed Delivery**: Persistent job queue ensures webhooks are delivered even if initial attempts fail
- **Intelligent Retries**: Exponential backoff with configurable limits prevents overwhelming failing endpoints
- **Complete Audit Trail**: Every webhook attempt is logged with full request/response data for debugging
- **Health Monitoring**: Automatic endpoint health tracking helps identify problematic integrations
- **Multi-tenancy**: Namespace isolation allows serving multiple applications or customers

### Technical Architecture

**Backend Stack:**

- **Go 1.24+** - High-performance, concurrent webhook processing
- **PostgreSQL** - Persistent storage for webhooks, events, and delivery tracking
- **River Job Queue** - PostgreSQL-based job queue for reliable background processing
- **gRPC + Connect-RPC** - Dual protocol support (binary gRPC + HTTP/JSON)
- **Protocol Buffers** - Type-safe API definitions with auto-generated clients
- **OpenTelemetry** - Distributed tracing and metrics collection

**Frontend Stack:**

- **SvelteKit** - Modern reactive web UI for webhook management
- **TypeScript** - Type-safe frontend with auto-generated API bindings
- **Vite** - Fast development build system

**Infrastructure:**

- **Docker Compose** - Local development environment
- **Database Migrations** - Version-controlled schema evolution
- **Health Checks** - Kubernetes-ready health endpoints
- **Observability** - Structured logging with correlation IDs

### Key Technical Features

- **Dual API Protocols**: Same business logic exposed via gRPC (high performance) and Connect-RPC (HTTP/JSON compatibility)
- **Clean Architecture**: Service layer abstraction allows easy testing and protocol flexibility
- **Request Body Storage**: Complete webhook payload preservation for audit and replay capabilities
- **Optimized Queue Payloads**: Minimal data in job queue (80-95% size reduction) with database normalization
- **Event Type Registry**: Global schema validation and event management across namespaces
- **Health State Machine**: Automatic endpoint health calculation based on delivery success rates
- **Zero-downtime Deployments**: Database migrations support backward compatibility

Whether you're building microservices, integrating third-party APIs, or need reliable event-driven communication, Sparrow provides enterprise-grade webhook infrastructure out of the box.

## 🚀 Features

- **Webhook Management**: Register, update, pause/resume webhook endpoints
- **Event Processing**: Trigger events with automatic delivery to subscribed webhooks
- **Reliable Delivery**: Intelligent retry logic with exponential backoff
- **Health Monitoring**: Real-time endpoint health tracking and alerting
- **Audit Trail**: Complete delivery history with request/response logging
- **Multi-tenant**: Namespace isolation for serving multiple applications
- **Dual APIs**: Both gRPC (performance) and HTTP/JSON (compatibility) protocols
- **Web UI**: Interactive dashboard for webhook management and monitoring

## 🏗️ Architecture

Sparrow uses a clean, layered architecture:

- **API Layer**: gRPC and Connect-RPC servers with protocol adapters
- **Service Layer**: Protocol-agnostic business logic for webhook operations
- **Repository Layer**: Database abstraction with optimized queries
- **Queue System**: River job queue for reliable background processing
- **Storage**: PostgreSQL with comprehensive indexing and triggers

**Tech Stack:** Go, PostgreSQL, River Queue, SvelteKit, TypeScript, OpenTelemetry

📋 **For detailed technical documentation, see [TECHNICAL.md](TECHNICAL.md)**

## 🚀 Technical Features

## 🚀 Quick Start

### Development Setup

```bash
# Start infrastructure
make docker-dev     # PostgreSQL + River queue + OpenTelemetry
make migrate        # Run database migrations

# Start services
make run           # gRPC + Connect-RPC servers
make run-web       # SvelteKit web UI (localhost:5173)

# Test the system
make example       # Run example client
```

### Available Commands

| Command | Description |
|---------|-------------|
| `make docker-dev` | Start development environment with Docker Compose |
| `make migrate` | Run database migrations |
| `make run` | Start gRPC and Connect-RPC servers |
| `make run-web` | Start web development server |
| `make example` | Run gRPC client example |
| `make docker-purge` | Clean up Docker resources |
| `make build` | Build server binary |
| `make test` | Run tests |
| `make generate` | Generate protobuf code |
| `make lint` | Run linter |
| `make fmt` | Format code |
| `make clean` | Clean build artifacts |

## 🔗 API Examples

### Register a Webhook

```bash
# Using Connect-RPC (HTTP/JSON)
curl -X POST http://localhost:8080/webhook/webhooks \
  -H "Content-Type: application/json" \
  -d '{
    "webhook": {
      "namespace": "my-app",
      "url": "https://api.example.com/webhook",
      "events": ["user.created", "user.updated"]
    }
  }'
```

### Trigger an Event

```bash
# Push an event to trigger webhook deliveries
curl -X POST http://localhost:8080/webhook/events \
  -H "Content-Type: application/json" \
  -d '{
    "event": {
      "namespace": "my-app",
      "name": "user.created",
      "payload": {
        "user_id": "12345",
        "email": "user@example.com",
        "created_at": "2024-01-15T10:30:00Z"
      }
    }
  }'
```

### Check Webhook Health

```bash
# Get webhook health status
curl http://localhost:8080/webhook/health/{webhook_id}?namespace=my-app
```

## 📊 Monitoring

- **Web Dashboard**: `http://localhost:5173` - Interactive webhook management
- **River Queue**: `http://localhost:8082/jobs` - Job queue monitoring
- **Health Endpoints**: `/health` and `/ready` - Kubernetes-compatible probes
- **OpenTelemetry**: Distributed tracing and metrics collection

## 🛠️ Configuration

Key environment variables:

```bash
DATABASE_URL="postgres://user:pass@localhost:5432/sparrow"
HTTP_PORT=8080
GRPC_PORT=50051
OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"
```

## 📚 Documentation

- **[TECHNICAL.md](TECHNICAL.md)** - Comprehensive technical documentation
- **[API Examples](examples/)** - Ready-to-run client examples
- **[Proto Definitions](proto/)** - gRPC service definitions

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make changes and add tests
4. Run tests: `make test && make lint`
5. Submit a pull request

## 📄 License

MIT License - see LICENSE file for details.
