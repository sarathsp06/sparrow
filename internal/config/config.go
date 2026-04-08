// Package config provides structured configuration loading from environment
// variables using [kelseyhightower/envconfig].
//
// All server configuration is defined in the [Config] struct. Call [Load] to
// populate it from the environment. Non-SPARROW-prefixed variables (DATABASE_URL,
// ENVIRONMENT, OTEL_EXPORTER_OTLP_ENDPOINT, CORS_ALLOWED_ORIGINS) are loaded
// separately because envconfig works with a single prefix.
package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all server configuration populated from environment variables.
// See Load() for the mapping between env vars and struct fields.
type Config struct {
	// Environment is the deployment environment (e.g. "production", "development").
	// Controls .env loading, CORS defaults, and OTel environment tag.
	// Env: ENVIRONMENT
	Environment string `envconfig:"ENVIRONMENT" default:""`

	// DatabaseURL is the PostgreSQL connection string.
	// Env: DATABASE_URL
	DatabaseURL string `envconfig:"DATABASE_URL" default:"postgres://localhost/riverqueue?sslmode=disable"`

	// GRPCPort is the port the gRPC server listens on.
	// Env: SPARROW_GRPC_PORT
	GRPCPort string `envconfig:"SPARROW_GRPC_PORT" default:"50051"`

	// HTTPPort is the port the HTTP/Connect-RPC server listens on.
	// Env: SPARROW_HTTP_PORT
	HTTPPort string `envconfig:"SPARROW_HTTP_PORT" default:"8080"`

	// APIKey is the shared-secret API key for authentication.
	// When set, all API requests must include this key via the X-API-Key header.
	// When empty, all endpoints are open (no authentication).
	// Env: SPARROW_API_KEY
	APIKey string `envconfig:"SPARROW_API_KEY" default:""`

	// ServeUI enables the embedded SvelteKit web UI.
	// Env: SPARROW_SERVE_UI
	ServeUI bool `envconfig:"SPARROW_SERVE_UI" default:"false"`

	// AllowPrivateNetworks relaxes SSRF protection to allow localhost and
	// private IP addresses as webhook target URLs. Useful for local dev.
	// Env: SPARROW_ALLOW_PRIVATE_NETWORKS
	AllowPrivateNetworks bool `envconfig:"SPARROW_ALLOW_PRIVATE_NETWORKS" default:"false"`

	// EncryptionKey is a 64-character hex string (32 bytes) used as the KEK
	// for envelope encryption of webhook secrets. If empty, a temporary key
	// is generated at startup (data encrypted with it will be unreadable
	// after restart).
	// Env: SPARROW_ENCRYPTION_KEY
	EncryptionKey string `envconfig:"SPARROW_ENCRYPTION_KEY" default:""`

	// OTLPEndpoint is the OpenTelemetry OTLP HTTP export endpoint.
	// When empty, OTel export is disabled.
	// Env: OTEL_EXPORTER_OTLP_ENDPOINT
	OTLPEndpoint string `envconfig:"OTEL_EXPORTER_OTLP_ENDPOINT" default:""`

	// CORSAllowedOrigins is a comma-separated list of allowed CORS origins.
	// When empty in production, cross-origin requests are blocked.
	// When empty in development, all origins are allowed.
	// Env: CORS_ALLOWED_ORIGINS
	CORSAllowedOrigins []string `envconfig:"CORS_ALLOWED_ORIGINS" default:""`
}

// Load populates a Config struct from environment variables.
// envconfig does not use a prefix since the env vars span multiple namespaces
// (SPARROW_*, DATABASE_URL, ENVIRONMENT, OTEL_*, CORS_*).
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("loading config from environment: %w", err)
	}
	return &cfg, nil
}

// IsProduction returns true when Environment is set to "production".
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}
