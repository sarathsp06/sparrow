![Work in Progress](https://img.shields.io/badge/Status-Work%20in%20Progress-orange?style=for-the-badge)
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

Sparrow uses a clean, layered architecture with dual API protocols:

- **API Layer**: Both gRPC (high performance) and HTTP/JSON (compatibility) endpoints
- **Service Layer**: Protocol-agnostic business logic for webhook operations
- **Queue System**: Reliable background processing with intelligent retry logic
- **Storage**: PostgreSQL with comprehensive auditing and health tracking

**Tech Stack:** Go, PostgreSQL, River Queue, SvelteKit, TypeScript, OpenTelemetry

📋 **For detailed technical documentation, see [TECHNICAL.md](TECHNICAL.md)**


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

```bash
DATABASE_URL="postgres://user:pass@localhost:5432/sparrow"
HTTP_PORT=8080
GRPC_PORT=50051
```

See [TECHNICAL.md](TECHNICAL.md) for complete configuration options.

## 📚 Documentation

- **[TECHNICAL.md](TECHNICAL.md)** - Comprehensive technical documentation
- **[BENCHMARKING.md](docs/BENCHMARKING.md)** - Performance testing and capacity planning
- **[API Examples](examples/)** - Ready-to-run client examples
- **[Proto Definitions](proto/)** - gRPC service definitions

## 🎯 Performance & Benchmarking

Sparrow includes comprehensive benchmarking tools for capacity planning:

```bash
# Run performance benchmarks
go test -bench=. -benchmem ./internal/benchmarks/

# Run load test with custom parameters
go build -o bin/benchmark ./cmd/benchmark/
./bin/benchmark -duration=2m -rps=1000 -payload=10
```

**Resource Estimates** (with 50% safety margin):
- **100 RPS**: 512 MB RAM, 1 CPU core, 10 Mbps bandwidth
- **1,000 RPS**: 2 GB RAM, 2 CPU cores, 50 Mbps bandwidth
- **10,000 RPS**: 16 GB RAM, 8 CPU cores, 500 Mbps bandwidth

See [BENCHMARKING.md](docs/BENCHMARKING.md) for detailed profiling and optimization guides.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make changes and add tests
4. Run tests: `make test && make lint`
5. Submit a pull request

## 📄 License

MIT License - see LICENSE file for details.
