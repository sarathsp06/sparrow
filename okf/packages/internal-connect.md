---
type: Go Package
title: internal/connect
description: Thin Connect-RPC adapter that delegates to gRPC service implementations
tags: [connect-rpc, http, api]
timestamp: 2026-06-22T00:00:00Z
---

# internal/connect

Thin adapter that exposes the gRPC service implementations as Connect-RPC HTTP endpoints on port 8080.

Each RPC method on `WebhookConnectServer` unwraps `connect.Request[T]`, calls the underlying gRPC handler (via `protoconnect.WebhookServiceHandler` etc.), and wraps the response.

[Routes to](/packages/cmd-server.md) via chi on `/webhook.WebhookService/*`, `/webhook.EventService/*`, etc.

## Citations

- `internal/connect/connect.go`
