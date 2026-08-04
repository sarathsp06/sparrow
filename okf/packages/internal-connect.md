---
type: Go Package
title: internal/connect
description: Connect-RPC adapter that forwards typed Connect requests to existing gRPC service implementations
tags: [connect-rpc, http, api]
timestamp: 2026-07-06T16:46:48Z
---

# internal/connect

Adapter that exposes the gRPC service implementations as Connect-RPC HTTP endpoints on port 8080.

`WebhookConnectServer` implements the generated `protoconnect.*ServiceHandler` interfaces. Those generated interfaces require one typed method per RPC, so each RPC method remains as a thin typed adapter. The repeated forwarding behavior is centralized in `forwardUnary`, which unwraps `connect.Request[T]`, calls the underlying gRPC handler, and wraps the response with `connect.NewResponse`.

[Routes to](/packages/cmd-server.md) via chi on `/webhook.WebhookService/*`, `/webhook.EventService/*`, etc.

[Delegates to](/packages/internal-grpc.md) for the actual RPC handler behavior.

## Citations

- `internal/connect/webhook_server.go`
- `proto/protoconnect/webhook.connect.go` — generated Connect handler interfaces
