# Sparrow Webhooks Python Client

Python gRPC client for the [Sparrow](https://github.com/sarathsp06/sparrow) webhook delivery platform.

## Installation

### From source (local development)

```bash
cd client/python
pip install -e .
```

### Dependencies only (use proto stubs directly)

```bash
pip install grpcio protobuf
```

## Quick Start

```python
import grpc
from proto import webhook_pb2, webhook_pb2_grpc
from google.protobuf.struct_pb2 import Struct

channel = grpc.insecure_channel("localhost:50051")

webhook_stub = webhook_pb2_grpc.WebhookServiceStub(channel)
event_stub = webhook_pb2_grpc.EventServiceStub(channel)

# Register a webhook
response = webhook_stub.RegisterWebhook(
    webhook_pb2.RegisterWebhookRequest(
        namespace="default",
        events=["order.created"],
        url="https://example.com/webhook",
        active=True,
    )
)
print(f"Webhook registered: {response.webhook_id}")

# Push an event
payload = Struct()
payload.update({"order_id": "ord_123", "total": 49.99})

push_response = event_stub.PushEvent(
    webhook_pb2.PushEventRequest(
        namespace="default",
        event="order.created",
        payload=payload,
        ttl_seconds=3600,
    )
)
print(f"Event pushed: {push_response.event_id}")
```

## API Key Authentication

If the Sparrow server has `SPARROW_API_KEY` set, include the key in gRPC metadata:

```python
metadata = [("x-api-key", "your-api-key")]

response = webhook_stub.RegisterWebhook(
    webhook_pb2.RegisterWebhookRequest(...),
    metadata=metadata,
)
```

## Available Services

| Stub Class | Description |
|---|---|
| `WebhookServiceStub` | Register, update, pause/resume webhooks |
| `EventServiceStub` | Push events, register event types |
| `SubscriptionServiceStub` | Manage webhook-event subscriptions |
| `DeliveryServiceStub` | Query delivery history and retry |
| `HealthServiceStub` | Webhook health metrics |

## Documentation

- [Sparrow Docs](https://sarathsp06.github.io/sparrow)
- [Python SDK Guide](https://sarathsp06.github.io/sparrow/guides/python/)
