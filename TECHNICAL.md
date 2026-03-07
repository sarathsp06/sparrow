# Sparrow - Technical Documentation

## What Is Sparrow?

Sparrow is an event-driven webhook delivery platform. It sits between your application and external HTTP endpoints, providing reliable, observable webhook delivery with retries, health tracking, and payload transformation.

**Problem it solves:** When your app needs to notify external systems (e.g., "user created", "order completed"), direct HTTP calls are fragile. Sparrow acts as a reliable intermediary that guarantees delivery, retries on failure, and provides full audit trails.

---

## Architecture Overview

```
Your App                    Sparrow                         External Endpoints
   |                          |                                    |
   |-- PushEvent -----------> |                                    |
   |   (HTTP/gRPC)            |-- Store event in DB                |
   |                          |-- Queue EventArgs job              |
   |                          |                                    |
   |                          |-- EventWorker picks up job         |
   |                          |   - Find matching subscriptions    |
   |                          |   - Create delivery records        |
   |                          |   - Queue WebhookArgs jobs         |
   |                          |                                    |
   |                          |-- WebhookWorker picks up job       |
   |                          |   - Load webhook config from DB    |
   |                          |   - Apply payload transformation   |
   |                          |   - HTTP POST to endpoint -------> |
   |                          |   - Record success/failure         |
   |                          |   - Update health metrics          |
   |                          |   - River retries on failure       |
```

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go |
| Database | PostgreSQL |
| Job Queue | River (Postgres-based) |
| API Protocols | gRPC (:50051) + Connect-RPC/HTTP (:8080) |
| Web UI | SvelteKit 5 + TypeScript (:5173) |
| Observability | OpenTelemetry (traces, metrics, logs) |
| Protobuf | buf.build toolchain |

## Core Concepts

### 1. Events
Event types that your system can produce (e.g., `user.created`, `order.completed`). Each event has:
- **Name**: Unique identifier (e.g., `user.created`)
- **Schema**: Optional JSON Schema for payload validation
- **Sample Payload**: Auto-generated from schema for testing templates

### 2. Webhooks
HTTP endpoints registered to receive event notifications. Each webhook has:
- **Namespace**: Logical isolation for multi-tenancy (e.g., `production`, `staging`)
- **URL**: Target HTTP endpoint
- **HTTP Config**: Retries, timeouts, SSL settings, expected status codes, etc.
- **Health**: Automatically tracked (healthy/degraded/unhealthy/unknown)

### 3. Subscriptions
The link between webhooks and events. A subscription defines:
- Which **event** a **webhook** should receive
- Per-event **headers**, **HTTP method**, **timeout** overrides
- **Payload transformation** via Go templates

### 4. Deliveries
Individual delivery attempts. Each delivery tracks:
- Status (pending/sending/success/failed/retrying/expired)
- Request/response bodies
- HTTP status codes and error messages
- Attempt count and retry schedule

## The Delivery Flow

1. **Register Event** - Define event types with optional JSON schemas
2. **Register Webhook** - Configure an HTTP endpoint with delivery settings
3. **Create Subscription** - Link a webhook to specific events (with optional transformations)
4. **Push Event** - Trigger an event with a payload
5. **Automatic Processing**:
   - Event is stored in `event_records` table
   - `EventArgs` job is queued on the `events` River queue
   - `EventProcessingWorker` finds all matching subscriptions
   - Creates `webhook_deliveries` records
   - Queues `WebhookArgs` jobs on the `webhooks` River queue
   - `WebhookWorker` applies transformations, sends HTTP request
   - Records delivery status, response, and health metrics
   - River handles retries with configurable max attempts

## API Surface

### Services & RPCs (22 total)

**WebhookService** (8 RPCs)
| RPC | Purpose |
|-----|---------|
| `RegisterWebhook` | Register a webhook URL with events and HTTP config |
| `UnregisterWebhook` | Remove a webhook |
| `ListWebhooks` | List webhooks with namespace/event/ID filters |
| `UpdateWebhookConfig` | Update webhook URL, headers, events, etc. |
| `PauseWebhook` | Temporarily disable a webhook |
| `ResumeWebhook` | Re-enable a paused webhook |
| `GetNamespaceStats` | Get delivery statistics for a namespace |
| `GetTemplateFunctions` | List available Go template functions |

**EventService** (7 RPCs)
| RPC | Purpose |
|-----|---------|
| `RegisterEvent` | Register a new event type with schema |
| `ListEvents` | List all event types |
| `GetEvent` | Get a single event by name |
| `UpdateEvent` | Update event schema/description/status |
| `DeleteEvent` | Delete an event type |
| `PushEvent` | Trigger an event (the main action) |
| `ListEventReports` | List pushed event instances with delivery stats |

**SubscriptionService** (6 RPCs)
| RPC | Purpose |
|-----|---------|
| `CreateSubscription` | Link a webhook to an event |
| `GetSubscription` | Get subscription details |
| `ListSubscriptions` | List subscriptions by webhook or event |
| `UpdateSubscription` | Update subscription config/template |
| `DeleteSubscription` | Remove a subscription |
| `TestSubscriptionTemplate` | Dry-run a Go template transformation |

**DeliveryService** (3 RPCs)
| RPC | Purpose |
|-----|---------|
| `GetDeliveryStatus` | Get details of a specific delivery |
| `ListDeliveries` | List deliveries with webhook/event filters |
| `RetryDelivery` | Manually retry failed deliveries |

**HealthService** (3 RPCs)
| RPC | Purpose |
|-----|---------|
| `GetWebhookHealth` | Get health metrics for a webhook |
| `ListWebhooksByHealth` | List webhooks by health status |
| `GetHealthSummary` | Get system-wide health counts |

### API Access

All RPCs are accessible via two protocols:
- **gRPC** on `:50051` (for high-performance programmatic access)
- **Connect-RPC (HTTP/JSON)** on `:8080` (for curl, browsers, any HTTP client)

Connect-RPC URLs follow the pattern: `POST http://localhost:8080/{service}/{method}`

Example: `POST http://localhost:8080/webhook.EventService/PushEvent`

## Database Schema (8 tables)

| Table | Purpose |
|-------|---------|
| `event_registrations` | Registered event types with schemas |
| `webhook_registrations` | Registered webhooks with HTTP config |
| `event_subscriptions` | Webhook-to-event mappings with per-subscription config |
| `event_records` | Pushed event instances (payloads stored here) |
| `webhook_deliveries` | Individual delivery attempts with status/response |
| `webhook_health_events` | Per-delivery health events |
| `webhook_health_summaries` | Aggregated health stats over time windows |
| `webhook_health_state` | Current health state per webhook (consecutive failures, etc.) |

## Queue Architecture

Uses River (Postgres-based job queue) with 3 queues:
- **`default`**: General purpose (5 max workers)
- **`events`**: Event processing (20 max workers, 30s poll)
- **`webhooks`**: Webhook delivery (20 max workers, 30s poll)

Retry behavior is controlled per-webhook via `max_retries` (0-10, default 3) and `retry_backoff_seconds` (1-3600, default 60). River handles the retry scheduling.

## Web UI Pages (13 routes)

| Route | Purpose |
|-------|---------|
| `/` | Landing page with namespace input |
| `/webhooks` | List all webhooks (with namespace filter) |
| `/webhooks/register` | Register a new webhook with full config |
| `/webhooks/[id]` | Webhook detail: health, deliveries, pause/resume |
| `/webhooks/[id]/subscriptions` | Manage event subscriptions & templates |
| `/events` | List all event types |
| `/events/register` | Register a new event type |
| `/events/push` | Push a test event with JSON payload |
| `/events/[id]/reports` | View pushed event instances & delivery stats |
| `/events/[id]/update` | Update event schema/description |
| `/deliveries/[id]` | View delivery details |
| `/health` | System-wide health dashboard |

## Quick Start

```bash
# 1. Start infrastructure
make docker-dev     # PostgreSQL + River + OpenTelemetry
make migrate        # Run database migrations

# 2. Start servers
make run            # Go server (gRPC :50051, HTTP :8080)
make run-web        # SvelteKit UI (:5173)

# 3. Test the system
make example        # Run example gRPC client
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://localhost/riverqueue?sslmode=disable` | PostgreSQL connection |
| `HTTP_PORT` | `8080` | Connect-RPC HTTP port |
| `GRPC_PORT` | `50051` | gRPC port |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OpenTelemetry collector |
| `ENVIRONMENT` | `development` | Environment name for OTEL |
| `PUBLIC_API_URL` | `http://localhost:8080` | API URL for web UI |
