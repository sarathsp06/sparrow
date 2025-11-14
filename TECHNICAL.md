# Sparrow Technical Documentation

This document provides comprehensive technical details about Sparrow's architecture, implementation, and internal systems.

## 🚀 Technical Features

### Webhook Lifecycle Management

- **Registration API** - REST/gRPC endpoints for webhook CRUD operations with validation
- **Dynamic Configuration** - Runtime updates to URLs, headers, timeouts, and event subscriptions
- **Pause/Resume Logic** - Graceful webhook disabling without losing queued deliveries
- **Bulk Operations** - Efficient management of multiple webhooks via batch APIs
- **URL Validation** - Automatic endpoint validation during registration
- **Custom Headers** - Support for authentication tokens, API keys, and custom headers

### Event Processing Engine

- **Event Type Registry** - Global schema definitions with JSON Schema validation
- **Namespace Isolation** - Complete tenant separation for multi-customer deployments
- **Event Routing** - Automatic delivery to all subscribed webhooks per namespace
- **Payload Transformation** - Support for custom event payload formatting
- **Event Replay** - Ability to resend historical events to new webhook endpoints
- **TTL Management** - Configurable event expiration to prevent infinite retries

### Reliable Delivery System

- **River Job Queue** - PostgreSQL-based persistent queue with ACID guarantees
- **Exponential Backoff** - Configurable retry intervals: 1s, 2s, 4s, 8s, 16s, 32s
- **Circuit Breaker** - Automatic pause of failing webhooks after consecutive failures
- **Delivery Timeouts** - Configurable HTTP client timeouts per webhook
- **Concurrent Workers** - Parallel webhook delivery with configurable worker pool size
- **Dead Letter Queue** - Failed deliveries after max retries are preserved for analysis

### Comprehensive Audit & Monitoring

- **Full Request Tracking** - Complete HTTP request/response storage including headers and body
- **Delivery Statistics** - Success rates, average response times, failure patterns
- **Health State Machine** - Automatic endpoint health: Healthy (>90%), Degraded (80-90%), Unhealthy (<80%)
- **Performance Metrics** - Response time percentiles, throughput rates, error classifications
- **Correlation IDs** - End-to-end request tracing across all system components
- **Time-series Data** - Historical webhook performance for trend analysis

### Developer Experience

- **Dual Protocol APIs** - Choose between gRPC (performance) or HTTP/JSON (simplicity)
- **Auto-generated Clients** - TypeScript, Go, and other language bindings from protobuf
- **Interactive Web UI** - Real-time webhook monitoring with JSON payload inspection
- **Local Development** - Complete Docker Compose setup with hot reload
- **Comprehensive Examples** - Ready-to-run client examples in multiple languages
- **OpenAPI/gRPC Documentation** - Auto-generated API documentation

### Production-Ready Operations

- **Health Checks** - Kubernetes-compatible liveness and readiness probes
- **Graceful Shutdown** - Proper cleanup of in-flight webhook deliveries
- **Database Migrations** - Version-controlled schema evolution with rollback support
- **Configuration Management** - Environment-based config with validation
- **Resource Limits** - Configurable memory and CPU limits for webhook processing
- **Observability Integration** - OpenTelemetry spans, metrics, and distributed tracing

### Security & Compliance

- **Request Validation** - Input sanitization and size limits
- **Webhook Authentication** - Support for Bearer tokens, API keys, and custom auth headers
- **Audit Logging** - Complete webhook interaction history for compliance
- **Data Retention** - Configurable retention policies for webhook delivery data
- **Namespace Security** - Tenant isolation prevents cross-customer data access
- **TLS Support** - Secure webhook deliveries with certificate validation

## 🏗️ System Architecture

### Layered Architecture Pattern

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Applications                       │
├─────────────────────────┬───────────────────────────────────────┤
│       gRPC Clients      │         HTTP/JSON Clients            │
├─────────────────────────┼───────────────────────────────────────┤
│      gRPC Server        │        Connect-RPC Server            │
├─────────────────────────┴───────────────────────────────────────┤
│                     Service Layer (Protocol Agnostic)          │
├─────────────────────────────────────────────────────────────────┤
│        Repository Layer (Database Abstraction)                 │
├─────────────────────────────────────────────────────────────────┤
│     PostgreSQL Database + River Job Queue                      │
├─────────────────────────────────────────────────────────────────┤
│              Background Workers (Webhook Delivery)             │
└─────────────────────────────────────────────────────────────────┘
```

### Core Components

**1. API Layer (`/internal/grpc/`, `/internal/connect/`)**

- Protocol adapters that convert between gRPC/HTTP and internal service calls
- Input validation, authentication, and protocol-specific error handling
- Shared business logic ensures consistent behavior across protocols

**2. Service Layer (`/internal/webhooks/`)**

- Protocol-agnostic business logic for webhook operations
- Event processing, webhook registration, and health management
- Dependency injection for easy testing and modularity

**3. Repository Layer (`/internal/webhooks/store/`)**

- Database access abstraction with clean interfaces
- SQL query optimization and transaction management
- Model mapping between database and service domain objects

**4. Queue System (`/internal/webhooks/queue/`, `/internal/webhooks/workers/`)**

- River job queue integration for reliable webhook delivery
- Optimized job payloads (minimal data, database lookups for full context)
- Configurable worker pools with concurrent delivery processing

**5. Background Processing**

- Event-driven webhook delivery with retry logic
- Health monitoring and automatic status updates
- Performance metrics collection and aggregation

### Data Flow Architecture

**Event Processing Flow:**

```
1. Client pushes event → API validation → Service layer
2. Service stores event → Creates delivery jobs for matching webhooks
3. River queue processes jobs → Worker fetches webhook details
4. HTTP delivery attempt → Response logging → Health update
5. Success: Job complete | Failure: Schedule retry with backoff
```

**Health Monitoring Flow:**

```
1. Delivery completion → Health metrics update (triggers)
2. Database calculates success rate → Updates webhook health status
3. Health timeseries data → Performance trend analysis
4. Web UI polling → Real-time health dashboard updates
```

### Database Design

**Core Tables:**

- `webhooks` - Registration data, configuration, and status
- `events` - Event instances with metadata and namespace isolation
- `event_registrations` - Global event type registry with JSON schemas
- `webhook_deliveries` - Complete audit trail with request/response bodies
- `webhook_health_events` - Individual delivery results for health calculation
- `webhook_health_timeseries` - Aggregated performance metrics over time

**Key Indexes:**

- GIN index on `request_body` for full-text search of webhook payloads
- Composite indexes on namespace + event type for efficient routing
- Time-based partitioning on delivery tables for performance at scale

**Database Triggers:**

- Automatic health status updates on delivery completion
- Performance metric calculations for real-time dashboard updates
- Audit log generation for compliance and debugging

### Performance Optimizations

**Queue Payload Optimization:**

- Minimal job data (80-95% size reduction) with database normalization
- ID-based lookups instead of full object serialization
- Reduced memory usage and improved queue throughput

**Concurrent Processing:**

- Configurable worker pools for parallel webhook delivery
- Connection pooling for database and HTTP client efficiency
- Async processing with proper resource cleanup

**Caching Strategy:**

- In-memory caching of webhook configurations for hot paths
- Event type schema caching for validation performance
- Health status caching with invalidation on status changes

### Observability Integration

**OpenTelemetry Implementation:**

- Distributed tracing across all service boundaries
- Custom metrics for webhook delivery performance
- Correlation IDs for end-to-end request tracking
- Integration with Jaeger, Prometheus, and other observability tools

**Monitoring Points:**

- API request/response times and error rates
- Webhook delivery success rates and response times
- Queue depth and processing latency
- Database query performance and connection health

## Technical Stack Details

### Backend Architecture

**Go 1.24+**

- High-performance, concurrent webhook processing
- Goroutines for parallel delivery processing
- Context-based request handling for cancellation
- Memory-efficient JSON processing

**PostgreSQL Database**

- ACID compliance for reliable webhook storage
- JSON/JSONB support for flexible event payloads
- Advanced indexing for high-performance queries
- Trigger-based health status calculations

**River Job Queue**

- PostgreSQL-based for consistency with main database
- ACID job processing with guaranteed delivery
- Built-in retry logic with exponential backoff
- Worker pool management and scaling

**Protocol Buffers + gRPC**

- Type-safe API definitions
- High-performance binary protocol
- Auto-generated client libraries
- Streaming support for real-time updates

**Connect-RPC**

- HTTP/JSON compatibility layer over gRPC
- Standard REST-like endpoints
- Compatible with existing HTTP tooling
- Browser and curl-friendly

### Frontend Architecture

**SvelteKit Framework**

- Reactive component system
- Server-side rendering capability
- Type-safe with TypeScript integration
- Hot module replacement for development

**TypeScript Integration**

- Auto-generated API types from protobuf
- Compile-time type checking
- Enhanced developer experience
- IntelliSense support

**Vite Build System**

- Fast development server
- Optimized production builds
- Hot module replacement
- Plugin ecosystem

### Infrastructure Components

**Docker Compose Development**

- Consistent development environment
- PostgreSQL with extensions
- OpenTelemetry collector
- Network isolation

**Database Migrations**

- Version-controlled schema changes
- Forward and backward migration support
- Safe deployment practices
- Rollback capabilities

**Health Check System**

- Kubernetes-compatible endpoints
- Database connectivity validation
- Queue system health monitoring
- Graceful degradation handling

## Configuration Management

### Environment Variables

```bash
# Database Configuration
DATABASE_URL="postgres://user:pass@localhost:5432/sparrow"
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5

# Server Configuration
HTTP_PORT=8080
GRPC_PORT=50051
GRACEFUL_SHUTDOWN_TIMEOUT=30s

# Queue Configuration
RIVER_WORKERS=10
RIVER_MAX_ATTEMPTS=6
RIVER_RETRY_POLICY=exponential

# Observability
OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"
OTEL_SERVICE_NAME="sparrow"
LOG_LEVEL=info

# Security
WEBHOOK_TIMEOUT=30s
MAX_PAYLOAD_SIZE=1MB
RATE_LIMIT_REQUESTS=1000
RATE_LIMIT_WINDOW=1m
```

### Configuration Validation

- Environment variable validation at startup
- Type checking for numeric values
- Required vs optional configuration
- Default value fallbacks
- Configuration hot-reloading support

## Security Implementation

### Input Validation

- Request payload size limits
- JSON schema validation
- URL format validation
- Header sanitization
- SQL injection prevention

### Authentication & Authorization

- API key authentication
- Bearer token support
- Namespace-based authorization
- Rate limiting per client
- CORS configuration

### Data Protection

- TLS encryption for webhook delivery
- Database connection encryption
- Sensitive data masking in logs
- Audit trail preservation
- GDPR compliance features

## Performance Characteristics

### Throughput Metrics

- 10,000+ webhooks/second delivery capacity
- Sub-100ms API response times
- 99.9% delivery success rate
- Horizontal scaling support

### Resource Requirements

**Minimum Configuration:**
- 512MB RAM
- 1 CPU core
- 10GB storage

**Recommended Production:**
- 2GB RAM
- 4 CPU cores
- 100GB storage with SSD

**High-Scale Configuration:**
- 8GB RAM
- 8 CPU cores
- 500GB storage with NVMe

### Scaling Strategies

- Horizontal worker scaling
- Database read replicas
- Connection pooling optimization
- Queue partitioning
- Load balancer configuration

## Monitoring and Alerting

### Key Metrics

**Application Metrics:**
- Webhook delivery success/failure rates
- API request latency percentiles
- Queue depth and processing time
- Worker utilization rates

**Infrastructure Metrics:**
- Database connection health
- Memory and CPU utilization
- Disk I/O performance
- Network throughput

**Business Metrics:**
- Customer webhook health scores
- Event processing volume
- Error categorization
- SLA compliance tracking

### Alert Thresholds

- 5+ consecutive webhook failures
- API response time > 1 second
- Queue depth > 10,000 jobs
- Database connection failure
- Worker crash/restart events

## Troubleshooting Guide

### Common Issues

**High Queue Depth:**
- Scale worker pool size
- Optimize webhook endpoints
- Review retry configuration
- Check database performance

**Delivery Failures:**
- Validate webhook URLs
- Check network connectivity
- Review timeout settings
- Examine error patterns

**Performance Issues:**
- Monitor database query performance
- Check connection pool utilization
- Review memory usage patterns
- Analyze GC pressure

### Debug Tools

- Distributed tracing with Jaeger
- Database query analysis
- Health check endpoints
- Worker status monitoring
- Event replay capabilities