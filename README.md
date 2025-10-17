# HTTP Queue with River & gRPC

A production-ready job queue system built with [River Queue](https://riverqueue.com/docs) and Go, featuring a gRPC API for webhook scheduling, structured logging, and comprehensive Docker integration.

## 🚀 Features

### Core Queue System
- **River Queue Integration**: Fast, robust job queue with PostgreSQL backend
- **Multiple Workers**: Data processing and webhook workers with configurable concurrency
- **Queue Management**: Multiple queues (default, webhooks) with separate worker pools
- **Graceful Shutdown**: Proper cleanup and signal handling
- **Structured Logging**: JSON logging with slog throughout the application

### gRPC API Service
- **Webhook Scheduling API**: Schedule single or batch webhook requests
- **Protocol Buffers**: Type-safe API definitions with generated Go code
- **Status Tracking**: Query webhook job status and execution details
- **Error Handling**: Comprehensive error responses and validation

### Infrastructure
- **Docker Compose**: Multi-service orchestration with dependency management
- **Database Migrations**: Automatic River schema setup
- **Monitoring**: River UI dashboard for job visualization
- **Development Tools**: pgAdmin for database management

## 📋 Prerequisites

- Go 1.25 or later
- Docker and Docker Compose
- Protocol Buffers compiler (protoc) for gRPC development

## 🚀 Quick Start

### Option 1: Docker Compose (Recommended)

Start the complete system with one command:

```bash
# Clone and start all services
git clone <repository>
cd httpqueue
docker-compose -f docker-compose.grpc.yml up -d
```

This starts:
- PostgreSQL database
- Database migrations
- River queue processor  
- gRPC API server (port 50051)
- River UI dashboard (port 8080)
- pgAdmin (port 8081)

### Option 2: Local Development

```bash
# Start PostgreSQL in Docker
docker-compose up postgres -d

# Set database URL
export DATABASE_URL="postgres://riveruser:riverpass@localhost:5432/riverqueue?sslmode=disable"

# Run migrations
go run cmd/migrate/main.go

# Start queue processor
go run cmd/grpc-server/main.go

# In another terminal, start gRPC server
go run cmd/grpc-server/main.go
```

## 🔧 gRPC API Usage

### Test the gRPC Service

```bash
# Run the provided client examples
go run examples/grpc_client.go
```

### gRPC Methods

#### Schedule Single Webhook
```go
req := &pb.ScheduleWebhookRequest{
    Url:    "https://httpbin.org/post",
    Method: "POST",
    Payload: `{"event": "user_signup", "user_id": "12345"}`,
    DelaySeconds: 0,
}
response, err := client.ScheduleWebhook(ctx, req)
```

#### Schedule Batch Webhooks
```go
req := &pb.ScheduleWebhookBatchRequest{
    Webhooks: []*pb.WebhookRequest{
        {Url: "https://api1.com/webhook", Payload: `{"data": "test1"}`},
        {Url: "https://api2.com/webhook", Payload: `{"data": "test2"}`},
    },
}
response, err := client.ScheduleWebhookBatch(ctx, req)
```

#### Check Webhook Status
```go
req := &pb.GetWebhookStatusRequest{JobId: 123}
response, err := client.GetWebhookStatus(ctx, req)
```

## 🏗️ Project Structure

```
httpqueue/
├── cmd/                          # Applications
│   ├── grpc-server/             # gRPC API server with queue processing
│   └── migrate/                 # Database migrations
├── internal/                    # Private application code
│   ├── config/                  # Configuration
│   ├── grpc/                    # gRPC service implementation
│   ├── jobs/                    # Job argument types
│   ├── logger/                  # Structured logging
│   ├── queue/                   # Queue management
│   └── workers/                 # Job workers
├── proto/                       # Protocol buffer definitions
├── examples/                    # Usage examples
├── docker-compose.grpc.yml      # Docker orchestration
└── GRPC_README.md              # gRPC documentation
```

## 🐳 Docker Services

- **PostgreSQL**: `localhost:5432` (riveruser/riverpass)
- **gRPC Server**: `localhost:50051` (webhook scheduling API)
- **River UI**: `http://localhost:8080` (job monitoring)
- **pgAdmin**: `http://localhost:8081` (admin@example.com/admin123)

## 🔄 How It Works

### Workers

- **DataProcessingWorker**: Handles background data processing tasks with 3-second simulation
- **WebhookWorker**: Sends HTTP requests with JSON payloads to external APIs
- **Configurable Concurrency**: 10 workers for default queue, 8 for webhook queue

### Job Types

1. **Webhook Jobs**: HTTP requests via gRPC API or direct insertion  
2. **Data Processing Jobs**: Background processing tasks
3. **Scheduled Jobs**: Delayed execution using `ScheduledAt`
4. **Batch Jobs**: Multiple jobs inserted at once
5. **Periodic Jobs**: Automatically created every 30 seconds for cleanup and health checks

### Webhook Features

- **Multiple HTTP Methods**: GET, POST, PUT, DELETE, PATCH
- **Custom Headers**: Authentication, content-type, custom headers
- **Configurable Timeouts**: Per-request timeout settings (5-30 seconds)
- **JSON Payloads**: Automatic JSON marshaling and validation
- **Error Handling**: Comprehensive error responses and retry logic
- **Status Tracking**: Monitor job progress and completion status

## 📊 Example Output

### gRPC Client Test

```bash
$ go run examples/grpc_client.go

=== Example 1: Simple Webhook ===
Webhook scheduled successfully:
  Job ID: 134
  Success: true
  Message: Webhook scheduled successfully
  Scheduled At: 2025-10-17 12:12:56 +0200 CEST

=== Example 3: Batch Webhooks ===
Batch webhooks scheduled:
  Total Scheduled: 3
  Total Failed: 0
  Webhook 1: Job ID 136, Success: true
  Webhook 2: Job ID 137, Success: true
  Webhook 3: Job ID 138, Success: true
```

### Queue Processing Logs

```json
{"time":"2025-10-17T10:12:56.415Z","level":"INFO","msg":"Processing webhook job","component":"webhook-worker","job_id":138,"url":"https://httpbin.org/post","method":"POST","payload_keys":3,"timeout":20}
{"time":"2025-10-17T10:12:58.568Z","level":"INFO","msg":"Webhook response received","component":"webhook-worker","job_id":138,"status_code":200,"duration_ms":2153}
{"time":"2025-10-17T10:12:58.568Z","level":"INFO","msg":"Webhook sent successfully","component":"webhook-worker","job_id":138,"status_code":200}
```

## 🔧 Technical Implementation

### Job Arguments

```go
type WebhookArgs struct {
    URL     string            `json:"url"`
    Method  string            `json:"method"`
    Payload json.RawMessage   `json:"payload"`
    Headers map[string]string `json:"headers,omitempty"`
    Timeout int               `json:"timeout,omitempty"`
}

func (WebhookArgs) Kind() string { return "webhook" }
```

### gRPC Service

```go
func (s *WebhookServer) ScheduleWebhook(ctx context.Context, req *pb.ScheduleWebhookRequest) (*pb.ScheduleWebhookResponse, error) {
    args := jobs.WebhookArgs{
        URL:     req.Url,
        Method:  req.Method,
        Payload: json.RawMessage(req.Payload),
        Headers: req.Headers,
        Timeout: int(req.TimeoutSeconds),
    }
    
    opts := river.InsertOpts{}
    if req.DelaySeconds > 0 {
        opts.ScheduledAt = time.Now().Add(time.Duration(req.DelaySeconds) * time.Second)
    }
    
    job, err := s.queueManager.GetClient().Insert(ctx, args, &opts)
    // Handle response...
}
```

## 🚀 Advanced Features

- **Structured Logging**: JSON logs with slog throughout the application
- **Graceful Shutdown**: Proper signal handling and resource cleanup
- **Docker Integration**: Complete containerized deployment
- **gRPC API**: Type-safe Protocol Buffer API with validation
- **Job Monitoring**: River UI for real-time job tracking
- **Database Management**: Automatic migrations and connection pooling
- **Error Handling**: Comprehensive error responses and retry logic

## 📚 Useful Commands

```bash
# View all running containers
docker ps

# Check gRPC server logs
docker logs httpqueue-grpc --tail 20

# Access River UI
open http://localhost:8080

# Monitor database
docker exec -it httpqueue-postgres psql -U riveruser -d riverqueue -c "SELECT * FROM river_job ORDER BY created_at DESC LIMIT 5;"

# Test gRPC service
go run examples/grpc_client.go

# Stop all services
docker-compose -f docker-compose.grpc.yml down
```

## 📖 Next Steps

1. **Implement job status tracking** - Add database queries for GetWebhookStatus
2. **Add authentication** - Secure the gRPC API with tokens or mTLS
3. **Enhance monitoring** - Add Prometheus metrics and health checks
4. **Job prioritization** - Implement priority queues for urgent webhooks
5. **Retry policies** - Add exponential backoff and dead letter queues
6. **Rate limiting** - Control webhook delivery rates per endpoint

## 📋 Documentation

- [River Documentation](https://riverqueue.com/docs)
- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)
- [Protocol Buffers Guide](https://developers.google.com/protocol-buffers)
- [Docker Compose Reference](https://docs.docker.com/compose/)
