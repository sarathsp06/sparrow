---
type: Go Package
title: internal/config
description: Server configuration loaded from environment variables via kelseyhightower/envconfig
tags: [config, leaf]
timestamp: 2026-08-29T00:00:00Z
---

# internal/config

Leaf package that loads all server configuration from environment variables.

## Config Struct

```go
type Config struct {
    Environment            string   // "development" or "production" (ENVIRONMENT)
    DatabaseURL            string   // DATABASE_URL
    HTTPPort               string   // default "8080" (SPARROW_HTTP_PORT)
    APIKey                 string   // optional (SPARROW_API_KEY)
    ServeUI                bool     // SPARROW_SERVE_UI
    AllowPrivateNetworks   bool     // SPARROW_ALLOW_PRIVATE_NETWORKS
    EncryptionKeys         []string // "<key-id>=<64-char-hex>" entries (SPARROW_ENCRYPTION_KEYS)
    EncryptionPrimaryKeyID string   // selects the active key (SPARROW_ENCRYPTION_PRIMARY_KEY_ID)
    OTLPEndpoint           string   // OTLP collector URL; empty disables export (OTEL_EXPORTER_OTLP_ENDPOINT)
    OTLPProtocol           string   // "http/protobuf" (default) or "grpc" (OTEL_EXPORTER_OTLP_PROTOCOL)
    OTLPSignals            []string // subset of traces,metrics,logs to export over OTLP (SPARROW_OTLP_SIGNALS)
    CORSAllowedOrigins     []string // CORS_ALLOWED_ORIGINS
    MaxBodyBytes           int64    // default 5 MiB, min 1 MiB (SPARROW_MAX_BODY_BYTES)
    EventRetentionDays     int      // 0 = keep forever (SPARROW_EVENT_RETENTION_DAYS)
    AIProvider             string   // anthropic | openai (SPARROW_AI_PROVIDER)
    AIAPIKey               string   // provider key; enables drafting for anthropic (SPARROW_AI_API_KEY)
    AIModel                string   // model id; AIModelOrDefault() → claude-haiku-4-5 for anthropic (SPARROW_AI_MODEL)
    AIBaseURL              string   // required for openai, optional gateway for anthropic (SPARROW_AI_BASE_URL)
}
```

## Functions

- `Load() (*Config, error)` — reads env vars (no prefix; spans SPARROW_*, DATABASE_URL, ENVIRONMENT, OTEL_*, CORS_*)
- `(*Config).IsProduction() bool`
- `(*Config).Validate() error` — checks port, keyring, DATABASE_URL, production API key, body-size floor, retention range
- `(*Config).Warnings() []string` — non-fatal advisories (e.g. sslmode=disable on non-local host)
- `(*Config).EncryptionKeyring() (*crypto.Keyring, error)` — resolves SPARROW_ENCRYPTION_KEYS + primary ID into a crypto.Keyring

## Citations

- `internal/config/config.go`
