# Sparrow Technical Documentation

This document provides comprehensive technical details about Sparrow's architecture, implementation, and internal systems.

## 🏗️ Technical Stack Overview

### Backend Architecture

**Go 1.24+**
- High-performance, concurrent webhook processing with goroutines
- Context-based request handling for graceful cancellation
- Memory-efficient JSON processing and HTTP client pooling
- Comprehensive error handling and circuit breaker patterns

**PostgreSQL Database**
- ACID compliance for reliable webhook and event storage
- JSON/JSONB support for flexible event payloads and metadata
- Advanced indexing strategies for high-performance queries
- Database triggers for automatic health status calculations
- Partitioning support for high-volume delivery logs

**River Job Queue**
- PostgreSQL-based job queue for consistency with main database
- ACID job processing with guaranteed delivery semantics
- Built-in retry logic with exponential backoff algorithms
- Worker pool management with dynamic scaling capabilities
- Job prioritization and scheduling for time-sensitive deliveries

**Protocol Buffers + gRPC**
- Type-safe API definitions with backward/forward compatibility
- High-performance binary protocol for internal services
- Auto-generated client libraries for multiple languages
- Streaming support for real-time webhook status updates
- Built-in authentication and authorization mechanisms

**Connect-RPC**
- HTTP/JSON compatibility layer over gRPC services
- Standard REST-like endpoints for easy integration
- Compatible with existing HTTP tooling and proxies
- Browser and curl-friendly for development and debugging
- OpenAPI documentation generation

### Frontend Architecture

**SvelteKit Framework**
- Reactive component system with minimal runtime overhead
- Server-side rendering for improved initial page load
- Type-safe integration with auto-generated TypeScript bindings
- Hot module replacement for rapid development cycles
- Progressive enhancement for accessibility

**TypeScript Integration**
- Auto-generated API types from Protocol Buffer definitions
- Compile-time type checking for reduced runtime errors
- Enhanced developer experience with IntelliSense
- Strict type safety across the entire frontend codebase

**Vite Build System**
- Lightning-fast development server with instant HMR
- Optimized production builds with tree-shaking
- Plugin ecosystem for extended functionality
- Modern JavaScript/TypeScript support out of the box

### Infrastructure Components

**Docker Compose Development Environment**
- Consistent development setup across team members
- PostgreSQL with required extensions (pgcrypto, uuid-ossp)
- OpenTelemetry collector for observability testing
- Network isolation and service discovery
- Volume persistence for database state

**Database Migration System**
- Version-controlled schema changes with migration files
- Forward and backward migration support for rollbacks
- Safe deployment practices with migration validation
- Automated migration execution in CI/CD pipelines
- Database seed data for development environments

**Health Check System**
- Kubernetes-compatible liveness and readiness probes
- Database connectivity and transaction validation
- Queue system health monitoring and worker status
- Graceful degradation handling for partial outages
- Custom health check endpoints for load balancers

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

### Layered Architecture Implementation

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Applications                       │
├─────────────────────────┬───────────────────────────────────────┤
│       gRPC Clients      │         HTTP/JSON Clients            │
│    (High Performance)   │        (Wide Compatibility)          │
├─────────────────────────┼───────────────────────────────────────┤
│      gRPC Server        │        Connect-RPC Server            │
│   (Binary Protocol)     │       (HTTP/JSON Protocol)           │
├─────────────────────────┴───────────────────────────────────────┤
│                     Service Layer (Protocol Agnostic)          │
│              Business Logic, Validation, Authorization         │
├─────────────────────────────────────────────────────────────────┤
│        Repository Layer (Database Abstraction)                 │
│           SQL Query Optimization, Transaction Management       │
├─────────────────────────────────────────────────────────────────┤
│     PostgreSQL Database + River Job Queue                      │
│        ACID Transactions, Indexing, Triggers, Partitioning     │
├─────────────────────────────────────────────────────────────────┤
│              Background Workers (Webhook Delivery)             │
│        Concurrent Processing, Retry Logic, Health Monitoring   │
└─────────────────────────────────────────────────────────────────┘
```

### Detailed Component Architecture

**1. API Layer (`/internal/grpc/`, `/internal/connect/`)**

- **Protocol Adapters**: Convert between gRPC protobuf messages and HTTP JSON
- **Input Validation**: Request payload validation using protocol-specific validators
- **Authentication Handlers**: API key validation and rate limiting per client
- **Error Translation**: Convert internal errors to appropriate protocol responses
- **Middleware Pipeline**: Logging, metrics collection, and request tracing
- **Content Negotiation**: Support for multiple content types and encodings

**2. Service Layer (`/internal/webhooks/`)**

- **Webhook Service**: Core business logic for webhook lifecycle management
- **Event Service**: Event processing, routing, and delivery orchestration
- **Health Service**: Endpoint health calculation and status management
- **Validation Engine**: Input sanitization and business rule enforcement
- **Authorization Logic**: Namespace-based access control and permission checking
- **Integration Interfaces**: Clean abstractions for external service dependencies

**3. Repository Layer (`/internal/webhooks/store/`)**

- **Database Interfaces**: Abstract database operations with clean contracts
- **Query Optimization**: Efficient SQL queries with proper indexing strategies
- **Transaction Management**: ACID compliance for complex multi-table operations
- **Model Mapping**: Translation between database rows and domain objects
- **Connection Pooling**: Optimized database connection management
- **Migration Support**: Schema versioning and backward compatibility

**4. Queue System (`/internal/webhooks/queue/`, `/internal/webhooks/workers/`)**

- **Job Scheduling**: Event-triggered webhook delivery job creation
- **Worker Pool Management**: Configurable concurrent webhook delivery workers
- **Retry Logic**: Exponential backoff with jitter for failed deliveries
- **Dead Letter Handling**: Failed job preservation for manual intervention
- **Priority Queuing**: Support for urgent vs. standard delivery priorities
- **Metrics Collection**: Queue depth, processing times, and success rates

**5. Background Processing Architecture**

- **Event-Driven Delivery**: Webhook jobs triggered by event creation
- **Health Monitoring**: Automatic endpoint health updates based on delivery results
- **Performance Analytics**: Real-time calculation of delivery metrics
- **Circuit Breaker**: Automatic webhook disabling for consistently failing endpoints
- **Batch Processing**: Efficient bulk operations for maintenance tasks
- **Resource Management**: Memory and CPU usage optimization

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

## 🔧 Comprehensive Configuration Management

### Complete Environment Variables

**Database Configuration**
```bash
# Primary database connection
DATABASE_URL="postgres://user:pass@localhost:5432/sparrow?sslmode=disable"

# Connection pool settings
DB_MAX_OPEN_CONNS=25              # Maximum open connections
DB_MAX_IDLE_CONNS=5               # Maximum idle connections
DB_CONN_MAX_LIFETIME=1h           # Connection lifetime
DB_CONN_MAX_IDLE_TIME=15m         # Idle connection timeout

# Migration settings
MIGRATION_TABLE="schema_migrations" # Migration tracking table
MIGRATION_AUTO_RUN=true           # Auto-run migrations on startup
```

**Server Configuration**
```bash
# HTTP Server (Connect-RPC)
HTTP_PORT=8080                    # HTTP server port
HTTP_HOST=0.0.0.0                # HTTP bind address
HTTP_READ_TIMEOUT=30s             # Request read timeout
HTTP_WRITE_TIMEOUT=30s            # Response write timeout
HTTP_IDLE_TIMEOUT=120s            # Keep-alive timeout
HTTP_MAX_HEADER_SIZE=1MB          # Maximum header size

# gRPC Server
GRPC_PORT=50051                   # gRPC server port
GRPC_HOST=0.0.0.0                # gRPC bind address
GRPC_MAX_RECV_MSG_SIZE=4MB        # Maximum receive message size
GRPC_MAX_SEND_MSG_SIZE=4MB        # Maximum send message size
GRPC_KEEPALIVE_TIME=2h            # Client keepalive time
GRPC_KEEPALIVE_TIMEOUT=20s        # Client keepalive timeout

# Graceful shutdown
GRACEFUL_SHUTDOWN_TIMEOUT=30s     # Shutdown timeout
SHUTDOWN_DELAY=5s                 # Delay before shutdown starts
```

**Queue and Worker Configuration**
```bash
# River Job Queue Settings
RIVER_WORKERS=10                  # Number of worker goroutines
RIVER_MAX_ATTEMPTS=6              # Maximum retry attempts
RIVER_RETRY_POLICY=exponential    # Retry policy (exponential, linear, fixed)
RIVER_FETCH_COOLDOWN=100ms        # Worker fetch cooldown
RIVER_FETCH_POLL_INTERVAL=1s      # Poll interval for new jobs
RIVER_RESCUE_STUCK_JOBS_AFTER=1h  # Rescue jobs stuck for this duration

# Webhook Delivery Settings
WEBHOOK_TIMEOUT=30s               # Default webhook timeout
WEBHOOK_MAX_RETRIES=5             # Default maximum retries
WEBHOOK_RETRY_BACKOFF=60s         # Default retry backoff
WEBHOOK_CONCURRENT_DELIVERIES=100 # Maximum concurrent deliveries

# Queue Health
QUEUE_HEALTH_CHECK_INTERVAL=30s   # Queue health check frequency
QUEUE_MAX_DEPTH_WARNING=10000     # Warning threshold for queue depth
QUEUE_MAX_DEPTH_CRITICAL=50000    # Critical threshold for queue depth
```

**Observability Configuration**
```bash
# OpenTelemetry
OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"
OTEL_EXPORTER_OTLP_HEADERS="api-key=your-key"  # Optional headers
OTEL_SERVICE_NAME="sparrow"                     # Service name in traces
OTEL_SERVICE_VERSION="1.0.0"                   # Service version
OTEL_RESOURCE_ATTRIBUTES="environment=production,datacenter=us-west"

# Logging
LOG_LEVEL=info                    # Log level (debug, info, warn, error)
LOG_FORMAT=json                   # Log format (json, text)
LOG_OUTPUT=stdout                 # Log output (stdout, stderr, file)
LOG_FILE_PATH="/var/log/sparrow.log"  # Log file path (if LOG_OUTPUT=file)
LOG_MAX_SIZE=100MB                # Maximum log file size
LOG_MAX_BACKUPS=3                 # Number of backup log files
LOG_MAX_AGE=7d                    # Maximum age of log files

# Metrics
METRICS_ENABLED=true              # Enable metrics collection
METRICS_PORT=9090                 # Prometheus metrics port
METRICS_PATH="/metrics"           # Metrics endpoint path
METRICS_NAMESPACE="sparrow"       # Metrics namespace

# Tracing
TRACING_ENABLED=true              # Enable distributed tracing
TRACING_SAMPLE_RATE=0.1           # Trace sampling rate (0.0 to 1.0)
TRACING_JAEGER_ENDPOINT="http://localhost:14268/api/traces"
```

**Security Configuration**
```bash
# Rate Limiting
RATE_LIMIT_ENABLED=true           # Enable rate limiting
RATE_LIMIT_REQUESTS=1000          # Requests per window
RATE_LIMIT_WINDOW=1m              # Rate limit window
RATE_LIMIT_BURST=100              # Burst allowance
RATE_LIMIT_REDIS_URL="redis://localhost:6379"  # Redis for distributed rate limiting

# Authentication
API_KEY_HEADER="X-API-Key"         # API key header name
API_KEY_REQUIRED=false            # Require API key for all requests
JWT_SECRET="your-jwt-secret"      # JWT signing secret
JWT_EXPIRY=24h                    # JWT token expiry

# TLS Configuration
TLS_ENABLED=false                 # Enable TLS for HTTP server
TLS_CERT_FILE="/path/to/cert.pem" # TLS certificate file
TLS_KEY_FILE="/path/to/key.pem"   # TLS private key file
TLS_MIN_VERSION="1.2"             # Minimum TLS version

# CORS
CORS_ENABLED=true                 # Enable CORS
CORS_ALLOWED_ORIGINS="*"          # Allowed origins (comma-separated)
CORS_ALLOWED_METHODS="GET,POST,PUT,DELETE,OPTIONS"  # Allowed methods
CORS_ALLOWED_HEADERS="*"          # Allowed headers
CORS_MAX_AGE=3600                 # Preflight cache duration
```

**Application Limits**
```bash
# Payload Limits
MAX_PAYLOAD_SIZE=1MB              # Maximum webhook payload size
MAX_HEADER_SIZE=32KB              # Maximum header size
MAX_URL_LENGTH=2048               # Maximum webhook URL length
MAX_CUSTOM_HEADERS=50             # Maximum custom headers per webhook

# Resource Limits
MAX_CONCURRENT_WEBHOOKS=1000      # Maximum concurrent webhook deliveries
MAX_EVENTS_PER_SECOND=10000       # Maximum events processed per second
MAX_MEMORY_USAGE=2GB              # Maximum memory usage
MAX_CPU_USAGE=80                  # Maximum CPU usage percentage

# Retention Settings
DELIVERY_LOG_RETENTION=30d        # Delivery log retention period
EVENT_RETENTION=90d               # Event data retention period
HEALTH_METRICS_RETENTION=365d     # Health metrics retention period
AUDIT_LOG_RETENTION=1y            # Audit log retention period
```

### Configuration Validation

Sparrow performs comprehensive configuration validation at startup:

- **Type Checking**: Numeric values, durations, and boolean flags
- **Range Validation**: Ports, percentages, and size limits
- **Dependency Checks**: Database connectivity and external service availability
- **Security Validation**: TLS certificate validity and encryption settings
- **Performance Warnings**: Potentially problematic configuration combinations

### Configuration Hot-Reloading

Supported hot-reload configurations:
- Log level changes via SIGHUP signal
- Rate limiting adjustments via admin API
- Webhook timeout modifications
- Worker pool size adjustments
- Health check threshold updates

### Environment-Specific Configurations

**Development Environment**
```bash
# Relaxed settings for local development
LOG_LEVEL=debug
RIVER_WORKERS=2
RIVER_FETCH_COOLDOWN=10ms
WEBHOOK_TIMEOUT=5s
RATE_LIMIT_ENABLED=false
```

**Production Environment**
```bash
# Optimized settings for production
LOG_LEVEL=info
RIVER_WORKERS=20
RIVER_FETCH_COOLDOWN=100ms
WEBHOOK_TIMEOUT=30s
RATE_LIMIT_ENABLED=true
TLS_ENABLED=true
METRICS_ENABLED=true
```

**High-Scale Environment**
```bash
# Settings for high-volume deployments
RIVER_WORKERS=50
DB_MAX_OPEN_CONNS=100
WEBHOOK_CONCURRENT_DELIVERIES=500
MAX_EVENTS_PER_SECOND=50000
TRACING_SAMPLE_RATE=0.01
```

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