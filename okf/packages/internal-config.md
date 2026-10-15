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
    LogLevel               string   // debug | info (default) | warn | error (SPARROW_LOG_LEVEL)
    AllowPrivateNetworks   bool     // SPARROW_ALLOW_PRIVATE_NETWORKS
    EncryptionKeys         []string // "<key-id>=<64-char-hex>" entries (SPARROW_ENCRYPTION_KEYS)
    EncryptionPrimaryKeyID string   // selects the active key (SPARROW_ENCRYPTION_PRIMARY_KEY_ID)
    OTLPEndpoint           string   // OTLP collector URL; empty disables export (OTEL_EXPORTER_OTLP_ENDPOINT)
    OTLPProtocol           string   // "http/protobuf" (default) or "grpc" (OTEL_EXPORTER_OTLP_PROTOCOL)
    OTLPSignals            []string // subset of traces,metrics,logs to export over OTLP (SPARROW_OTLP_SIGNALS)
    CORSAllowedOrigins     []string // CORS_ALLOWED_ORIGINS
    MaxBodyBytes           int64    // default 5 MiB, min 1 MiB (SPARROW_MAX_BODY_BYTES)
    EventRetentionDays     int      // 0 = keep forever (SPARROW_EVENT_RETENTION_DAYS)
    ShutdownTimeout        time.Duration // whole graceful-shutdown budget, default 10s (SPARROW_SHUTDOWN_TIMEOUT)
    AIDrafting             bool     // the Draft with AI switch; provider vars are read only when true (SPARROW_AI_ENABLED)
    AIProvider             string   // anthropic | openai (SPARROW_AI_PROVIDER)
    AIAPIKey               string   // provider key; required for anthropic when AIDrafting (SPARROW_AI_API_KEY)
    AIModel                string   // model id; AIModelOrDefault() → claude-haiku-4-5 for anthropic (SPARROW_AI_MODEL)
    AIBaseURL              string   // required for openai, optional gateway for anthropic (SPARROW_AI_BASE_URL)
}
```

## Functions

- `Load() (*Config, error)` — reads env vars (no prefix; spans SPARROW_*, DATABASE_URL, ENVIRONMENT, OTEL_*, CORS_*)
- `(*Config).IsProduction() bool`
- `(*Config).Validate() error` — checks port, keyring, all-zeros key in production, DATABASE_URL, log level, production API key, body-size floor, retention range
- `(*Config).SlogLevel() (slog.Level, error)` — parses SPARROW_LOG_LEVEL (case-insensitive, `warning` = `warn`); `cmd/server/main.go` sets it on the JSON handler behind the OTel log bridge
- `(*Config).Warnings() []string` — non-fatal advisories (e.g. sslmode=disable on non-local host, all-zeros key outside production, AI key set with SPARROW_AI_ENABLED off, `AIConfigError`)
- `(*Config).EncryptionKeyring() (*crypto.Keyring, error)` — resolves SPARROW_ENCRYPTION_KEYS + primary ID into a crypto.Keyring

## Citations

- `internal/config/config.go`
