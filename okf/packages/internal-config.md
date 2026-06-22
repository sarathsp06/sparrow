---
type: Go Package
title: internal/config
description: Server configuration loaded from environment variables via kelseyhightower/envconfig
tags: [config, leaf]
timestamp: 2026-06-22T00:00:00Z
---

# internal/config

Leaf package that loads all server configuration from environment variables.

## Config Struct

```go
type Config struct {
    Environment          string // "development" or "production"
    DatabaseURL          string
    GRPCPort             string // default "50051"
    HTTPPort             string // default "8080"
    APIKey               string // optional, SPARROW_API_KEY
    ServeUI              bool
    AllowPrivateNetworks bool
    EncryptionKey        string // 64-char hex (32 bytes), required
    OTLPEndpoint         string // OTLP HTTP exporter
    CORSAllowedOrigins   string // comma-separated
}
```

## Functions

- `Load() (*Config, error)` — reads env vars
- `(*Config).IsProduction() bool`
- `(*Config).Validate() error`

## Citations

- `internal/config/config.go`
