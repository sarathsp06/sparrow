# Sparrow Client Libraries

This directory contains **generated** client stubs for Go, JavaScript/TypeScript, and Python. Sparrow supports two RPC protocols:

- **Connect-RPC** (recommended) -- HTTP/JSON on port 8080, works with standard `net/http`, browsers, proxies
- **gRPC** -- Native gRPC on port 50051, for environments that already use gRPC infrastructure

> **Note:** All files in `go/`, `js/`, and `python/` are auto-generated. Do not edit them manually.
> Regenerate with: `make generate`

## Generating Clients

```bash
# Generate all protobuf code (server + clients)
make generate

# Or run buf directly
buf generate
```

This produces:

| Directory             | Language              | Transport    |
|-----------------------|-----------------------|--------------|
| `client/go/proto/protoconnect/` | Go             | Connect-RPC  |
| `client/go/proto/`   | Go                    | gRPC         |
| `client/js/connect/`  | JavaScript/TypeScript | Connect-RPC (ES modules) |
| `client/js/`          | JavaScript/TypeScript | gRPC-Web     |
| `client/python/`      | Python                | gRPC         |

---

## Go (Connect-RPC) -- Recommended

Connect-RPC uses standard `net/http` -- no special transport, no HTTP/2 requirement, works through any HTTP proxy. This is the recommended way to talk to Sparrow.

### Installation

```bash
go get github.com/sarathsp06/sparrow/client/go/proto
go get connectrpc.com/connect
```

### Full Example

```go
package main

import (
	"context"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/sarathsp06/sparrow/client/go/proto"
	pbconnect "github.com/sarathsp06/sparrow/client/go/proto/protoconnect"
)

func main() {
	// Connect-RPC uses standard net/http -- no special setup needed
	httpClient := http.DefaultClient

	webhookClient := pbconnect.NewWebhookServiceClient(httpClient, "http://localhost:8080")
	eventClient := pbconnect.NewEventServiceClient(httpClient, "http://localhost:8080")
	subscriptionClient := pbconnect.NewSubscriptionServiceClient(httpClient, "http://localhost:8080")

	ctx := context.Background()

	// 1. Register an event type
	eventResp, err := eventClient.RegisterEvent(ctx, connect.NewRequest(&pb.RegisterEventRequest{
		Name:        "order.created",
		Description: "New order placed",
		Active:      true,
	}))
	if err != nil {
		log.Fatalf("register event: %v", err)
	}
	log.Printf("Event registered: %s", eventResp.Msg.GetName())

	// 2. Register a webhook with automatic subscriptions
	webhookResp, err := webhookClient.RegisterWebhook(ctx, connect.NewRequest(&pb.RegisterWebhookRequest{
		Namespace: "default",
		Url:       "https://testhooks.sarathsadasivan.com/hooks",
		Events:    []string{"order.created"},
		Active:    true,
	}))
	if err != nil {
		log.Fatalf("register webhook: %v", err)
	}
	webhookID := webhookResp.Msg.GetWebhookId()
	log.Printf("Webhook registered: %s", webhookID)

	// 3. Create a subscription with label filters
	//    This subscription only receives events where plan=premium
	subResp, err := subscriptionClient.CreateSubscription(ctx, connect.NewRequest(&pb.CreateSubscriptionRequest{
		WebhookId: webhookID,
		EventName: "order.created",
		Namespace: "default",
		Active:    true,
		LabelFilters: map[string]string{
			"plan": "premium",
		},
	}))
	if err != nil {
		log.Fatalf("create subscription: %v", err)
	}
	log.Printf("Subscription created: %s (label filter: plan=premium)", subResp.Msg.GetSubscription().GetId())

	// 4. Push an event with labels
	//    Only subscriptions whose label_filters are a subset of these labels will match
	payload, _ := structpb.NewStruct(map[string]any{
		"order_id": "ord_456",
		"amount":   149.99,
	})
	pushResp, err := eventClient.PushEvent(ctx, connect.NewRequest(&pb.PushEventRequest{
		Namespace:  "default",
		Event:      "order.created",
		Payload:    payload,
		TtlSeconds: 3600,
		Labels: map[string]string{
			"plan":   "premium",
			"region": "us-east-1",
		},
	}))
	if err != nil {
		log.Fatalf("push event: %v", err)
	}
	log.Printf("Event pushed: %s", pushResp.Msg.GetEventId())

	// 5. Check delivery status
	deliveryClient := pbconnect.NewDeliveryServiceClient(httpClient, "http://localhost:8080")
	deliveries, err := deliveryClient.ListDeliveries(ctx, connect.NewRequest(&pb.ListDeliveriesRequest{
		WebhookId: webhookID,
		Namespace: "default",
		Limit:     10,
	}))
	if err != nil {
		log.Fatalf("list deliveries: %v", err)
	}
	for _, d := range deliveries.Msg.GetDeliveries() {
		log.Printf("Delivery %s: status=%s attempts=%d", d.GetId(), d.GetStatus(), d.GetAttemptCount())
	}

	// 6. Check webhook health
	healthClient := pbconnect.NewHealthServiceClient(httpClient, "http://localhost:8080")
	health, err := healthClient.GetWebhookHealth(ctx, connect.NewRequest(&pb.GetWebhookHealthRequest{
		WebhookId: webhookID,
		Namespace: "default",
	}))
	if err != nil {
		log.Fatalf("get health: %v", err)
	}
	log.Printf("Health: status=%s success_rate=%.1f%% consecutive_failures=%d",
		health.Msg.GetHealthStatus(),
		health.Msg.GetSuccessRate(),
		health.Msg.GetConsecutiveFailures(),
	)
}
```

### Using TLS / Custom HTTP Client

Since Connect-RPC uses standard `net/http`, you can use any HTTP client configuration:

```go
httpClient := &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			// your TLS config
		},
	},
	Timeout: 30 * time.Second,
}

client := pbconnect.NewWebhookServiceClient(httpClient, "https://sparrow.example.com:8080")
```

---

## Go (gRPC) -- Alternative

Use gRPC when you need HTTP/2 streaming, already have gRPC infrastructure, or need to connect to port 50051 directly.

### Installation

```bash
go get github.com/sarathsp06/sparrow/client/go/proto
go get google.golang.org/grpc
```

### Example

```go
package main

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/sarathsp06/sparrow/client/go/proto"
)

func main() {
	// gRPC requires explicit connection setup
	conn, err := grpc.NewClient("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()

	webhookClient := pb.NewWebhookServiceClient(conn)
	eventClient := pb.NewEventServiceClient(conn)

	// Register a webhook
	resp, err := webhookClient.RegisterWebhook(ctx, &pb.RegisterWebhookRequest{
		Namespace: "default",
		Events:    []string{"order.created"},
		Url:       "https://testhooks.sarathsadasivan.com/hooks",
		Active:    true,
	})
	if err != nil {
		log.Fatalf("register webhook: %v", err)
	}
	log.Printf("Webhook registered: %s", resp.WebhookId)

	// Push an event with labels
	payload, _ := structpb.NewStruct(map[string]any{
		"order_id": "ord_123",
		"total":    49.99,
	})
	pushResp, err := eventClient.PushEvent(ctx, &pb.PushEventRequest{
		Namespace:  "default",
		Event:      "order.created",
		Payload:    payload,
		TtlSeconds: 3600,
		Labels: map[string]string{
			"plan":   "premium",
			"region": "us-east-1",
		},
	})
	if err != nil {
		log.Fatalf("push event: %v", err)
	}
	log.Printf("Event pushed: %s", pushResp.EventId)
}
```

### Using TLS (gRPC)

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

## JavaScript / TypeScript (Connect-RPC) -- Recommended

The `client/js/connect/` directory contains ES modules generated by `protoc-gen-es`. Use these with `@connectrpc/connect-web` for browser and Node.js clients -- no gRPC-Web proxy needed.

### Installation

```bash
npm install @connectrpc/connect @connectrpc/connect-web @bufbuild/protobuf
```

### Usage

```typescript
import { createConnectTransport } from "@connectrpc/connect-web";
import { createClient } from "@connectrpc/connect";

// Import the generated service and message types
// (copy client/js/connect/proto/ into your project)

const transport = createConnectTransport({
  baseUrl: "http://localhost:8080",
});

// Use with your service definitions
```

---

## JavaScript / TypeScript (gRPC-Web) -- Alternative

The `client/js/` directory contains traditional gRPC-Web stubs. Use these if your infrastructure already has a gRPC-Web proxy (e.g., Envoy).

### Installation

```bash
npm install grpc-web google-protobuf
```

### Usage

```typescript
import { WebhookServiceClient } from './proto/webhook_grpc_web_pb';
import { RegisterWebhookRequest } from './proto/webhook_pb';

const client = new WebhookServiceClient('http://localhost:8080');

const request = new RegisterWebhookRequest();
request.setNamespace('default');
request.setEventsList(['order.created']);
request.setUrl('https://testhooks.sarathsadasivan.com/hooks');
request.setActive(true);

client.registerWebhook(request, {}, (err, response) => {
  if (err) {
    console.error('Error:', err.message);
    return;
  }
  console.log('Webhook ID:', response.getWebhookId());
});
```

---

## Python (gRPC)

No Connect-RPC plugin is available for Python yet. Use the standard gRPC stubs.

### Installation

```bash
pip install grpcio grpcio-tools protobuf
```

### Usage

```python
import grpc
from proto import webhook_pb2, webhook_pb2_grpc
from google.protobuf.struct_pb2 import Struct


def main():
    channel = grpc.insecure_channel("localhost:50051")

    webhook_stub = webhook_pb2_grpc.WebhookServiceStub(channel)
    event_stub = webhook_pb2_grpc.EventServiceStub(channel)

    # Register a webhook
    response = webhook_stub.RegisterWebhook(
        webhook_pb2.RegisterWebhookRequest(
            namespace="default",
            events=["order.created"],
            url="https://testhooks.sarathsadasivan.com/hooks",
            active=True,
        ),
    )
    print(f"Webhook registered: {response.webhook_id}")

    # Push an event with labels
    payload = Struct()
    payload.update({"order_id": "order_123", "total": 49.99})

    push_response = event_stub.PushEvent(
        webhook_pb2.PushEventRequest(
            namespace="default",
            event="order.created",
            payload=payload,
            ttl_seconds=3600,
            labels={"plan": "premium", "region": "us-east-1"},
        ),
    )
    print(f"Event pushed: {push_response.event_id}")


if __name__ == "__main__":
    main()
```

---

## Available Services

| Service                | Description                           |
|------------------------|---------------------------------------|
| `WebhookService`       | Register, update, pause/resume webhooks |
| `EventService`         | Push events, register event types     |
| `SubscriptionService`  | Manage webhook-event subscriptions    |
| `DeliveryService`      | Query delivery history and attempts   |
| `HealthService`        | Webhook health metrics and summaries  |

> **NamespaceService** is Go-only (no proto definition) and is accessed via Connect-RPC HTTP/JSON at `:8080`.

## Ports

| Protocol     | Port  | Recommended Client             |
|--------------|-------|--------------------------------|
| Connect-RPC  | 8080  | Go: `protoconnect`, JS: `@connectrpc/connect-web` |
| gRPC         | 50051 | Go: `grpc`, Python: `grpcio`, JS: `grpc-web` |

## Testing

Use [testhooks.sarathsadasivan.com](https://testhooks.sarathsadasivan.com/) as a webhook endpoint for testing. It accepts any payload and lets you inspect the requests.

## Further Reading

- [Connect-RPC documentation](https://connectrpc.com/) -- recommended protocol
- [gRPC Go documentation](https://grpc.io/docs/languages/go/)
- [gRPC Python documentation](https://grpc.io/docs/languages/python/)
- [gRPC-Web documentation](https://github.com/nicedoc/grpc-web)
- [webhook.proto](../webhook.proto) -- Service and message definitions
