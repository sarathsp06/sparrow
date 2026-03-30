# Sparrow gRPC Client Libraries

This directory contains **generated** gRPC client stubs for Go, JavaScript/TypeScript, and Python. These clients connect to Sparrow's native gRPC endpoint (default: port 50051).

> **Note:** The files in `go/`, `js/`, and `python/` are auto-generated. Do not edit them manually.
> Regenerate with: `make generate`

## Generating Clients

```bash
# Generate all protobuf code (server + clients)
make generate

# Or run buf directly
buf generate
```

This produces:

| Directory        | Language              | Transport     |
|------------------|-----------------------|---------------|
| `client/go/`     | Go                    | gRPC          |
| `client/js/`     | JavaScript/TypeScript | gRPC-Web      |
| `client/python/` | Python                | gRPC          |

---

## Go

### Installation

```bash
go get github.com/sarathsp06/sparrow/client/go/proto
```

Or copy the generated files from `client/go/` into your project.

### Dependencies

```bash
go get google.golang.org/grpc
go get google.golang.org/protobuf
```

### Usage

```go
package main

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	structpb "google.golang.org/protobuf/types/known/structpb"

	pb "github.com/sarathsp06/sparrow/client/go/proto"
)

func main() {
	// Connect to gRPC server
	conn, err := grpc.NewClient("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()

	// Create service clients
	webhookClient := pb.NewWebhookServiceClient(conn)
	eventClient := pb.NewEventServiceClient(conn)

	// Register a webhook
	resp, err := webhookClient.RegisterWebhook(ctx, &pb.RegisterWebhookRequest{
		Namespace: "default",
		Events:    []string{"order.created"},
		Url:       "https://example.com/webhook",
		Active:    true,
	})
	if err != nil {
		log.Fatalf("failed to register webhook: %v", err)
	}
	log.Printf("webhook registered: %s", resp.WebhookId)

	// Push an event
	payload, _ := structpb.NewStruct(map[string]any{
		"order_id": "order_123",
		"total":    49.99,
	})

	pushResp, err := eventClient.PushEvent(ctx, &pb.PushEventRequest{
		Namespace:  "default",
		Event:      "order.created",
		Payload:    payload,
		TtlSeconds: 3600,
	})
	if err != nil {
		log.Fatalf("failed to push event: %v", err)
	}
	log.Printf("event pushed: %s", pushResp.EventId)
}
```

### Using TLS

```go
import "google.golang.org/grpc/credentials"

creds, err := credentials.NewClientTLSFromFile("ca-cert.pem", "")
if err != nil {
	log.Fatalf("failed to load TLS cert: %v", err)
}

conn, err := grpc.NewClient("sparrow.example.com:50051",
	grpc.WithTransportCredentials(creds),
)
```

---

## JavaScript / TypeScript (gRPC-Web)

The JavaScript client uses [gRPC-Web](https://github.com/nicedoc/grpc-web), which works in both browsers and Node.js (with a gRPC-Web proxy or a server that supports it).

### Installation

```bash
npm install grpc-web google-protobuf
```

Copy the generated files from `client/js/` into your project.

### Usage

```typescript
import { WebhookServiceClient } from './proto/webhook_grpc_web_pb';
import { RegisterWebhookRequest } from './proto/webhook_pb';

// Create client (gRPC-Web connects over HTTP)
const client = new WebhookServiceClient('http://localhost:8080');

// Register a webhook
const request = new RegisterWebhookRequest();
request.setNamespace('default');
request.setEventsList(['order.created']);
request.setUrl('https://example.com/webhook');
request.setActive(true);

client.registerWebhook(request, {}, (err, response) => {
  if (err) {
    console.error('Error:', err.message);
    return;
  }
  console.log('Webhook ID:', response.getWebhookId());
});
```

### Using Promises (async/await)

```typescript
function registerWebhook(
  client: WebhookServiceClient,
  request: RegisterWebhookRequest,
): Promise<RegisterWebhookResponse> {
  return new Promise((resolve, reject) => {
    client.registerWebhook(request, {}, (err, response) => {
      if (err) reject(err);
      else resolve(response);
    });
  });
}

// Usage
const response = await registerWebhook(client, request);
```

---

## Python

### Installation

```bash
pip install grpcio grpcio-tools protobuf
```

Copy the generated files from `client/python/` into your project.

### Usage

```python
import grpc
from proto import webhook_pb2
from proto import webhook_pb2_grpc


def main():
    # Connect to gRPC server
    channel = grpc.insecure_channel("localhost:50051")

    # Create service stubs
    webhook_stub = webhook_pb2_grpc.WebhookServiceStub(channel)
    event_stub = webhook_pb2_grpc.EventServiceStub(channel)

    # Register a webhook
    response = webhook_stub.RegisterWebhook(
        webhook_pb2.RegisterWebhookRequest(
            namespace="default",
            events=["order.created"],
            url="https://example.com/webhook",
            active=True,
        ),
    )
    print(f"Webhook registered: {response.webhook_id}")

    # Push an event
    from google.protobuf.struct_pb2 import Struct

    payload = Struct()
    payload.update({"order_id": "order_123", "total": 49.99})

    push_response = event_stub.PushEvent(
        webhook_pb2.PushEventRequest(
            namespace="default",
            event="order.created",
            payload=payload,
            ttl_seconds=3600,
        ),
    )
    print(f"Event pushed: {push_response.event_id}")


if __name__ == "__main__":
    main()
```

### Using TLS

```python
with open("ca-cert.pem", "rb") as f:
    creds = grpc.ssl_channel_credentials(f.read())

channel = grpc.secure_channel("sparrow.example.com:50051", creds)
```

---

## Available Services

| Service                      | Description                           | Proto? |
|------------------------------|---------------------------------------|--------|
| `WebhookService`             | Register, update, pause/resume webhooks | Yes |
| `EventService`               | Push events, register event types     | Yes |
| `SubscriptionService`        | Manage webhook-event subscriptions    | Yes |
| `DeliveryService`            | Query delivery history and attempts   | Yes |
| `HealthService`              | Webhook health metrics and summaries  | Yes |
| `NamespaceService`           | Create and manage namespaces          | No (Go-only, no generated client stubs) |

## Ports

| Protocol     | Default Port | Description                          |
|--------------|-------------|---------------------------------------|
| gRPC         | 50051       | Native gRPC (use these client stubs)  |
| Connect-RPC  | 8080        | HTTP-based RPC (used by the web UI)   |

## Further Reading

- [gRPC Go documentation](https://grpc.io/docs/languages/go/)
- [gRPC Python documentation](https://grpc.io/docs/languages/python/)
- [gRPC-Web documentation](https://github.com/nicedoc/grpc-web)
- [Connect-RPC documentation](https://connectrpc.com/)
