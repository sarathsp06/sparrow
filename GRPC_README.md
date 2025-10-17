# gRPC Webhook Service

This service provides a gRPC API for scheduling webhooks to be processed by the HTTP Queue system.

## Features

- **Schedule Single Webhooks**: Send webhooks immediately or with delay
- **Batch Webhook Scheduling**: Schedule multiple webhooks at once
- **Webhook Status Checking**: Query the status of scheduled webhooks
- **Flexible Configuration**: Support for custom headers, methods, timeouts, and queues

## API Reference

### Service: `WebhookService`

#### 1. `ScheduleWebhook`
Schedules a single webhook to be sent.

**Request:**
```protobuf
message ScheduleWebhookRequest {
  string url = 1;                           // Target URL (required)
  string method = 2;                        // HTTP method (default: POST)
  map<string, string> headers = 3;          // HTTP headers
  string payload = 4;                       // JSON payload as string
  int32 timeout = 5;                        // Timeout in seconds (default: 30)
  string queue = 6;                         // Queue name (default: "webhooks")
  int64 scheduled_at = 7;                   // Unix timestamp for scheduling
  int32 priority = 8;                       // Job priority
}
```

**Response:**
```protobuf
message ScheduleWebhookResponse {
  int64 job_id = 1;                         // Unique job identifier
  bool success = 2;                         // Whether scheduling was successful
  string message = 3;                       // Success or error message
  int64 scheduled_at = 4;                   // When the job was scheduled for
}
```

#### 2. `ScheduleWebhookBatch`
Schedules multiple webhooks to be sent.

**Request:**
```protobuf
message ScheduleWebhookBatchRequest {
  repeated ScheduleWebhookRequest webhooks = 1;
}
```

**Response:**
```protobuf
message ScheduleWebhookBatchResponse {
  repeated ScheduleWebhookResponse results = 1;
  int32 total_scheduled = 2;
  int32 total_failed = 3;
}
```

#### 3. `GetWebhookStatus`
Gets the status of a webhook job.

**Request:**
```protobuf
message GetWebhookStatusRequest {
  int64 job_id = 1;
}
```

**Response:**
```protobuf
message GetWebhookStatusResponse {
  int64 job_id = 1;
  WebhookJobStatus status = 2;              // PENDING, RUNNING, COMPLETED, FAILED, etc.
  string message = 3;
  int64 created_at = 4;
  int64 scheduled_at = 5;
  int64 attempted_at = 6;
  int32 attempt_count = 7;
  int32 max_attempts = 8;
}
```

## Running the Service

### With Docker Compose

1. **Start all services including gRPC:**
```bash
docker-compose -f docker-compose.grpc.yml up --build
```

2. **Services will be available at:**
   - gRPC Server: `localhost:50051`
   - HTTP Queue App: `localhost:8080` (placeholder)
   - River UI: `localhost:8082`
   - pgAdmin: `localhost:8081`
   - PostgreSQL: `localhost:5432`

### Standalone

1. **Build the gRPC server:**
```bash
GOTOOLCHAIN=go1.25.0 go build -o grpc-server ./cmd/grpc-server
```

2. **Run with environment variables:**
```bash
export DATABASE_URL="postgres://riveruser:riverpass@localhost:5432/riverqueue?sslmode=disable"
export GRPC_PORT="50051"
./grpc-server
```

## Testing the Service

### Using the Go client example:
```bash
GOTOOLCHAIN=go1.25.0 go run examples/grpc_client.go
```

### Using grpcurl (if you have it installed):

1. **List services:**
```bash
grpcurl -plaintext localhost:50051 list
```

2. **Schedule a webhook:**
```bash
grpcurl -plaintext \
  -d '{
    "url": "https://httpbin.org/post",
    "method": "POST",
    "payload": "{\"event\":\"test\",\"data\":\"hello world\"}",
    "headers": {"Content-Type": "application/json"},
    "timeout": 30,
    "queue": "webhooks"
  }' \
  localhost:50051 webhook.WebhookService/ScheduleWebhook
```

3. **Schedule batch webhooks:**
```bash
grpcurl -plaintext \
  -d '{
    "webhooks": [
      {
        "url": "https://httpbin.org/post",
        "payload": "{\"batch_id\":1}"
      },
      {
        "url": "https://httpbin.org/post", 
        "payload": "{\"batch_id\":2}"
      }
    ]
  }' \
  localhost:50051 webhook.WebhookService/ScheduleWebhookBatch
```

## Client Examples

### Go Client
See `examples/grpc_client.go` for a complete Go client implementation.

### Python Client
```python
import grpc
import webhook_pb2
import webhook_pb2_grpc
import json

# Connect to server
channel = grpc.insecure_channel('localhost:50051')
client = webhook_pb2_grpc.WebhookServiceStub(channel)

# Schedule webhook
response = client.ScheduleWebhook(webhook_pb2.ScheduleWebhookRequest(
    url="https://httpbin.org/post",
    method="POST",
    payload=json.dumps({"event": "test", "data": "hello"}),
    headers={"Content-Type": "application/json"},
    timeout=30
))

print(f"Job ID: {response.job_id}")
print(f"Success: {response.success}")
```

### Node.js Client
```javascript
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');

// Load protobuf
const packageDefinition = protoLoader.loadSync('proto/webhook.proto');
const webhook = grpc.loadPackageDefinition(packageDefinition).webhook;

// Create client
const client = new webhook.WebhookService('localhost:50051', 
  grpc.credentials.createInsecure());

// Schedule webhook
client.ScheduleWebhook({
  url: "https://httpbin.org/post",
  method: "POST",
  payload: JSON.stringify({event: "test", data: "hello"}),
  headers: {"Content-Type": "application/json"},
  timeout: 30
}, (error, response) => {
  if (error) {
    console.error('Error:', error);
  } else {
    console.log('Job ID:', response.job_id);
    console.log('Success:', response.success);
  }
});
```

## Configuration

### Environment Variables

- `DATABASE_URL`: PostgreSQL connection string
- `GRPC_PORT`: Port for gRPC server (default: 50051)

### Queue Configuration

The service supports multiple queues for organizing webhooks:
- `webhooks` (default): General webhook queue
- `priority`: High-priority webhooks
- Custom queue names can be specified per request

## Monitoring

- **River UI**: View job status, retry attempts, and queue health at `http://localhost:8082`
- **Logs**: Structured JSON logs with request tracing
- **Database**: Direct PostgreSQL access for advanced monitoring

## Error Handling

- **Invalid requests**: Returns gRPC error with details
- **JSON parsing errors**: Invalid payload format
- **Database errors**: Connection or insertion failures
- **Automatic retries**: Failed webhooks are retried by River queue system

## Production Considerations

1. **Security**: Add authentication/authorization to gRPC endpoints
2. **TLS**: Enable TLS for production deployments
3. **Rate Limiting**: Implement request rate limiting
4. **Monitoring**: Add metrics collection (Prometheus, etc.)
5. **Load Balancing**: Scale multiple gRPC server instances