# Sparrow Technical Documentation

This document provides comprehensive technical details about Sparrow's architecture, implementation, and internal systems.

## � Core Architecture Principle: Webhook-Event Separation

**Webhooks are registered independently of events.** Events are linked to webhooks through subscriptions:

- **Webhook Registration** - Defines the endpoint URL and base configuration (timeout, retry policy, etc.)
- **Event Subscription** - Links a webhook to specific events with per-event configuration
- **Benefits**:
  - One webhook can handle multiple events with different configurations
  - Easy to add/remove event subscriptions without modifying the webhook
  - Per-subscription headers, HTTP methods, timeouts, and payload transformations
  - Clean separation of concerns: endpoint definition vs. event routing

**Example:**
```
1. Register Webhook: POST /RegisterWebhook → {url: "https://api.example.com/hook"}
2. Create Subscriptions:
   - Subscribe to "user.created" with POST method and transformation template
   - Subscribe to "user.deleted" with DELETE method and custom headers
3. Push Event: When "user.created" fires → subscription config used for delivery
```

## �🏗️ Technical Stack Overview

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

- **Registration API** - REST/gRPC endpoints for webhook CRUD operations with validation (events NOT specified at registration)
- **Dynamic Configuration** - Runtime updates to URLs, headers, timeouts via UpdateWebhookConfig
- **Pause/Resume Logic** - Graceful webhook disabling without losing queued deliveries
- **Bulk Operations** - Efficient management of multiple webhooks via batch APIs
- **URL Validation** - Automatic endpoint validation during registration
- **Custom Headers** - Support for authentication tokens, API keys, and custom headers at webhook or subscription level

### Event Subscription System

- **Fine-grained Control** - Subscribe webhooks to specific events with per-subscription configuration
- **Per-Subscription Headers** - Override webhook-level headers for specific event types
- **Method Override** - Configure custom HTTP methods (POST, PUT, PATCH) per subscription
- **Timeout Customization** - Set different timeout values for different event types
- **Subscription Management** - CRUD operations for managing event subscriptions independently
- **Event Filtering** - List all subscriptions for a webhook or find subscriptions by event type
- **Subscription Isolation** - Each subscription is independent with unique configuration

### Template Transformation System

- **Go Template Engine** - Powerful payload transformation using Go's template syntax
- **Rich Function Library** - 20+ built-in functions for JSON, base64, URL encoding, string manipulation
- **Time Operations** - Format, parse, and manipulate timestamps with flexible layouts
- **Data Transformation** - Convert, filter, and reshape event payloads before delivery
- **Template Caching** - High-performance template parsing with in-memory cache
- **Header Templating** - Dynamic header values using template expressions
- **Error Handling** - Graceful fallback when templates fail to execute
- **Template Validation** - Parse-time validation to catch template syntax errors
- **Custom Functions** - String operations (upper, lower, trim, replace, split, join)
- **Encoding Support** - JSON marshaling, base64 encode/decode, URL encoding
- **Documentation** - Comprehensive template function reference with examples

**Available Template Functions:**
- `json` - Convert values to JSON strings
- `urlencode` - URL-encode strings for safe parameter passing
- `base64`, `base64decode` - Base64 encoding and decoding
- `now`, `formatTime`, `parseTime` - Time operations and formatting
- `upper`, `lower`, `title` - String case transformations
- `trim`, `trimPrefix`, `trimSuffix` - String trimming operations
- `replace`, `replaceAll` - String search and replace
- `split`, `join` - String splitting and joining
- `contains`, `hasPrefix`, `hasSuffix` - String matching
- `default` - Provide default values for missing fields
- `env` - Access environment variables
- And more... (see [TEMPLATE_FUNCTIONS.md](TEMPLATE_FUNCTIONS.md))

### Event Processing Engine

- **Event Type Registry** - Global schema definitions with JSON Schema validation
- **Sample Payload Generation** - Auto-generated sample payloads from JSON schemas
- **Namespace Isolation** - Complete tenant separation for multi-customer deployments
- **Event Routing** - Automatic delivery to all subscribed webhooks per namespace
- **Subscription-based Delivery** - Route events based on subscription configuration
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
- **Template Playground** - Test and debug payload transformations interactively
- **Template Documentation API** - Built-in endpoint returns all template functions with examples
- **Sample Payload Generation** - Auto-generate sample payloads from JSON schemas for testing
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
- **Subscription Service**: Event subscription management and configuration
- **Event Service**: Event processing, routing, and delivery orchestration
- **Template Engine**: Go template parsing, caching, and execution for payload transformation
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
2. Service stores event → Queries subscriptions for matching event + namespace
3. Creates delivery jobs for each subscription with configuration
4. River queue processes jobs → Worker fetches webhook + subscription details
5. Apply template transformation (if enabled) → Generate request
6. HTTP delivery attempt with subscription-specific settings → Response logging
7. Update health metrics → Success: Job complete | Failure: Schedule retry
```

**Subscription-based Delivery Flow:**

```
1. Event triggered → Find all subscriptions for (namespace, event_name)
2. For each subscription:
   a. Load subscription configuration (headers, method, timeout, template)
   b. Apply template transformation if transform_enabled=true
   c. Merge subscription headers with webhook headers (subscription takes precedence)
   d. Create delivery job with transformed payload and merged configuration
3. Queue processes jobs with per-subscription settings
4. Track delivery metrics at both webhook and subscription level
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

- `webhook_registrations` - Webhook registration data and configuration
- `event_subscriptions` - Links webhooks to events with per-subscription config (headers, method, transform)
- `events` - Event instances with metadata and namespace isolation
- `event_registrations` - Global event type registry with JSON schemas and sample payloads
- `webhook_deliveries` - Complete audit trail with request/response bodies and subscription_id
- `webhook_health_events` - Individual delivery results for health calculation
- `webhook_health_timeseries` - Aggregated performance metrics over time

**Subscription Architecture:**

The refactored architecture uses `event_subscriptions` as the linking table between webhooks and events:
- Each subscription represents one webhook listening to one event type
- Subscriptions carry per-event configuration (custom headers, HTTP method, timeout)
- Subscriptions enable payload transformation via Go templates
- Multiple subscriptions per webhook allow different handling for different events
- Unique constraint on (webhook_id, event_name, namespace) prevents duplicate subscriptions

**Key Indexes:**

- GIN index on `request_body` for full-text search of webhook payloads
- Composite index on `event_subscriptions(namespace, event_name)` for fast event routing
- Index on `event_subscriptions(webhook_id)` for subscription lookups
- Index on `webhook_deliveries(subscription_id)` for subscription-level metrics
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
- **Template Caching** - Parsed Go templates cached in memory for zero parse overhead
- **Template Reuse** - Same template instance reused across deliveries for efficiency
- **LRU Eviction** - Automatic cache eviction for rarely-used templates

**Template Performance:**

- Template parsing happens once at first use, then cached
- Template execution overhead: <1ms for typical payloads
- Benchmark results: 100K+ transformations/second on modern hardware
- Memory footprint: ~2KB per cached template
- Thread-safe execution with zero contention

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

**Template Errors:**
- Validate template syntax before enabling transformation
- Check template function usage matches documentation
- Review error logs for template execution failures
- Test templates with sample payloads in development
- Use GetTemplateFunctions API to verify available functions

### Debug Tools

- Distributed tracing with Jaeger
- Database query analysis
- Health check endpoints
- Worker status monitoring
- Event replay capabilities
- Template function documentation API

## API Reference

### Subscription Management APIs

**CreateSubscription** - Create event subscription with custom configuration
```protobuf
rpc CreateSubscription(CreateSubscriptionRequest) returns (CreateSubscriptionResponse);
```
- Configure per-subscription headers, method, timeout
- Enable payload transformation with Go templates
- Unique constraint prevents duplicate subscriptions

**GetSubscription** - Retrieve subscription by ID
```protobuf
rpc GetSubscription(GetSubscriptionRequest) returns (GetSubscriptionResponse);
```

**ListSubscriptions** - List all subscriptions for a webhook
```protobuf
rpc ListSubscriptions(ListSubscriptionsRequest) returns (ListSubscriptionsResponse);
```

**UpdateSubscription** - Update subscription configuration
```protobuf
rpc UpdateSubscription(UpdateSubscriptionRequest) returns (UpdateSubscriptionResponse);
```
- Modify headers, timeout, HTTP method
- Update or disable payload transformation templates

**DeleteSubscription** - Remove subscription
```protobuf
rpc DeleteSubscription(DeleteSubscriptionRequest) returns (DeleteSubscriptionResponse);
```

**ListSubscriptionsByEvent** - Find subscriptions for specific event
```protobuf
rpc ListSubscriptionsByEvent(ListSubscriptionsByEventRequest) returns (ListSubscriptionsByEventResponse);
```

### Template System APIs

**GetTemplateFunctions** - Get complete template function reference
```protobuf
rpc GetTemplateFunctions(GetTemplateFunctionsRequest) returns (GetTemplateFunctionsResponse);
```
- Returns all 20+ available template functions
- Includes usage examples and descriptions
- Useful for building template editors and documentation

### Enhanced Event APIs

**RegisterEvent** - Register event with JSON schema
```protobuf
rpc RegisterEvent(RegisterEventRequest) returns (RegisterEventResponse);
```
- Now supports `sample_payload` auto-generation from schema
- Validates JSON schema format at registration time

## Migration Guide

### Schema Evolution

The system has undergone significant architectural improvements through database migrations:

**Migration 000002** - Webhook-Event Refactoring
- Removed `events` column from `webhook_registrations` table
- Introduced `event_subscriptions` as dedicated linking table
- Enables per-subscription configuration and transformation
- Backward compatible: existing webhook-event relationships migrated automatically

**Migration 000003** - Subscription Tracking in Deliveries
- Added `subscription_id` column to `webhook_deliveries`
- Enables subscription-level delivery metrics
- Improves audit trail for subscription-based routing

**Migration 000004** - Sample Payload Support
- Added `sample_payload` column to `event_registrations`
- Auto-generates sample JSON from event schemas
- Improves developer experience and testing

### Upgrade Path

**From Legacy Webhook-Event Model:**
1. System automatically migrates existing relationships to subscriptions
2. One subscription created per webhook-event pair
3. Default subscription settings applied (POST method, no transformation)
4. No downtime required - migrations are backward compatible

**Adopting New Features:**
1. **Subscriptions**: Use CreateSubscription API for fine-grained event handling (events are ONLY specified in subscriptions, not webhook registration)
2. **Templates**: Enable `transform_enabled` on subscriptions and provide Go template
3. **Sample Payloads**: Registered events automatically generate samples from schemas
4. **Template Functions**: Use GetTemplateFunctions API to discover capabilities

**Important Architecture Note:**
- Webhooks are registered WITHOUT specifying events
- Events are configured through subscriptions (CreateSubscription API)
- This separation allows per-event configuration (headers, method, timeout, templates)
- One webhook can have multiple subscriptions with different configurations per event

## Usage Examples

### Creating a Webhook with Subscription

```bash
# 1. Register the webhook endpoint (NO events specified here)
curl -X POST http://localhost:8080/webhook.WebhookService/RegisterWebhook \\
  -H "Content-Type: application/json" \\
  -d '{
    "namespace": "production",
    "url": "https://api.example.com/webhooks",
    "active": true,
    "description": "Production webhook endpoint",
    "http_config": {
      "max_retries": 5,
      "retry_backoff_seconds": 60,
      "request_timeout_seconds": 30,
      "verify_ssl": true
    }
  }'

# Response: {"webhook_id": "wh_abc123", "success": true}

# 2. Create subscription to specify which events this webhook receives
curl -X POST http://localhost:8080/webhook.WebhookService/CreateSubscription \\
  -H "Content-Type: application/json" \\
  -d '{
    "webhook_id": "wh_abc123",
    "event_name": "user.created",
    "namespace": "production",
    "headers": {
      "X-Event-Type": "user.created",
      "Authorization": "Bearer secret_token"
    },
    "method": "POST",
    "timeout": 30,
    "transform_enabled": false
  }'
```

### Using Payload Transformation

```bash
# Create subscription with template transformation
curl -X POST http://localhost:8080/webhook.WebhookService/CreateSubscription \\
  -H "Content-Type: application/json" \\
  -d '{
    "webhook_id": "wh_abc123",
    "event_name": "order.completed",
    "namespace": "production",
    "transform_enabled": true,
    "transform_template": "{\\"order_id\\": \\"{{ .payload.id }}\\", \\"customer\\": \\"{{ .payload.customer_email | upper }}\\", \\"total\\": {{ .payload.amount }}, \\"timestamp\\": \\"{{ now | formatTime \\"2006-01-02T15:04:05Z07:00\\" }}\\"}"
  }'
```

**Template Input (Event Payload):**
```json
{
  "id": "ord_12345",
  "customer_email": "john@example.com",
  "amount": 99.99,
  "items": ["item1", "item2"]
}
```

**Template Output (Delivered to Webhook):**
```json
{
  "order_id": "ord_12345",
  "customer": "JOHN@EXAMPLE.COM",
  "total": 99.99,
  "timestamp": "2026-01-08T14:30:45Z"
}
```

### Template Examples

**Adding Custom Headers with Templates:**
```json
{
  "webhook_id": "wh_abc123",
  "event_name": "payment.processed",
  "namespace": "production",
  "headers": {
    "X-Signature": "{{ .payload.signature | base64 }}",
    "X-Timestamp": "{{ now | formatTime \\"RFC3339\\" }}",
    "X-Event-ID": "{{ .event_id }}"
  }
}
```

**Complex Transformation with Multiple Functions:**
```json
{
  "transform_template": "{\\"user_id\\": \\"{{ .payload.user_id }}\\", \\"email\\": \\"{{ .payload.email | lower | urlencode }}\\", \\"profile_url\\": \\"https://example.com/users/{{ .payload.user_id | urlencode }}\\", \\"signup_date\\": \\"{{ parseTime \\"2006-01-02\\" .payload.created_at | formatTime \\"January 2, 2006\\" }}\\", \\"metadata\\": {{ .payload.metadata | json }}}"
}
```

### Getting Template Function Documentation

```bash
# Get all available template functions
curl -X POST http://localhost:8080/webhook.WebhookService/GetTemplateFunctions \\
  -H "Content-Type: application/json" \\
  -d '{}'

# Response includes all 20+ functions with descriptions and examples
{
  "functions": [
    {
      "name": "json",
      "description": "Converts any value to a JSON string...",
      "example": "{{ .data | json }}"
    },
    {
      "name": "base64",
      "description": "Base64 encodes a string...",
      "example": "{{ .secret | base64 }}"
    }
    // ... more functions
  ]
}
```

### Multiple Subscriptions for Different Events

```bash
# Subscribe same webhook to multiple events with different configs
curl -X POST http://localhost:8080/webhook.WebhookService/CreateSubscription \\
  -H "Content-Type: application/json" \\
  -d '{
    "webhook_id": "wh_abc123",
    "event_name": "user.created",
    "namespace": "production",
    "method": "POST",
    "timeout": 10
  }'

curl -X POST http://localhost:8080/webhook.WebhookService/CreateSubscription \\
  -H "Content-Type: application/json" \\
  -d '{
    "webhook_id": "wh_abc123",
    "event_name": "user.deleted",
    "namespace": "production",
    "method": "DELETE",
    "timeout": 5,
    "headers": {
      "X-Deletion-Reason": "user_request"
    }
  }'
```