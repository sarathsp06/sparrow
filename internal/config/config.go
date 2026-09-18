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
	"net/url"
	"strconv"
	"strings"

	"github.com/kelseyhightower/envconfig"

	"github.com/sarathsp06/sparrow/pkg/crypto"
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

	// HTTPPort is the port the HTTP/REST server listens on.
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

	// EncryptionKeys is the required keyring configuration. Each entry
	// must be "<key-id>=<64-char-hex-key>" where the key is a cryptographically
	// random 32-byte (256-bit) value encoded as 64 hex characters. New
	// encryption uses the primary key; all configured keys remain valid for
	// decryption.
	// Env: SPARROW_ENCRYPTION_KEYS
	EncryptionKeys []string `envconfig:"SPARROW_ENCRYPTION_KEYS" default:""`

	// EncryptionPrimaryKeyID selects which key from EncryptionKeys is primary
	// for new encryption. Required whenever EncryptionKeys is set.
	// Env: SPARROW_ENCRYPTION_PRIMARY_KEY_ID
	EncryptionPrimaryKeyID string `envconfig:"SPARROW_ENCRYPTION_PRIMARY_KEY_ID" default:""`

	// OTLPEndpoint is the OpenTelemetry OTLP HTTP export endpoint.
	// When empty, OTel export is disabled.
	// Env: OTEL_EXPORTER_OTLP_ENDPOINT
	OTLPEndpoint string `envconfig:"OTEL_EXPORTER_OTLP_ENDPOINT" default:""`

	// CORSAllowedOrigins is a comma-separated list of allowed CORS origins.
	// When empty in production, cross-origin requests are blocked.
	// When empty in development, all origins are allowed.
	// Env: CORS_ALLOWED_ORIGINS
	CORSAllowedOrigins []string `envconfig:"CORS_ALLOWED_ORIGINS" default:""`

	// MaxBodyBytes caps the size of incoming HTTP request bodies in bytes.
	// Defaults to 5 MiB. Must be at least 1 MiB (Huma's per-operation limit)
	// so oversized bodies surface as 413 rather than 500.
	// Env: SPARROW_MAX_BODY_BYTES
	MaxBodyBytes int64 `envconfig:"SPARROW_MAX_BODY_BYTES" default:"5242880"`

	// SendGridAPIKey is the SendGrid API key for the bootstrapped alert
	// webhook (system events -> email). The webhook stays inactive until this
	// and a non-placeholder AlertFromEmail are configured.
	// Env: SPARROW_SENDGRID_API_KEY
	SendGridAPIKey string `envconfig:"SPARROW_SENDGRID_API_KEY" default:""`

	// AlertFromEmail is the verified sender address used by the bootstrapped
	// SendGrid alert webhook. Env: SPARROW_ALERT_FROM_EMAIL
	AlertFromEmail string `envconfig:"SPARROW_ALERT_FROM_EMAIL" default:"alerts@example.com"`

	// AlertFromName is the sender display name used by the bootstrapped
	// SendGrid alert webhook. Env: SPARROW_ALERT_FROM_NAME
	AlertFromName string `envconfig:"SPARROW_ALERT_FROM_NAME" default:"Sparrow"`

	// EventRetentionDays purges events (and their deliveries, via cascade)
	// older than this many days. 0 (default) disables retention: data is
	// kept forever. Runs hourly as a background job.
	// Env: SPARROW_EVENT_RETENTION_DAYS
	EventRetentionDays int `envconfig:"SPARROW_EVENT_RETENTION_DAYS" default:"0"`
}

// Load populates a Config struct from environment variables.
// envconfig does not use a prefix since the env vars span multiple consumers
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

// Validate checks configuration values for sanity.
// Call after Load() and before using the config.
func (c *Config) Validate() error {
	if err := validatePort(c.HTTPPort, "SPARROW_HTTP_PORT"); err != nil {
		return err
	}
	if _, err := c.EncryptionKeyring(); err != nil {
		return err
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.IsProduction() && c.APIKey == "" {
		return fmt.Errorf("SPARROW_API_KEY is required in production (without it all endpoints are unauthenticated)")
	}
	if c.MaxBodyBytes < 1<<20 {
		return fmt.Errorf("SPARROW_MAX_BODY_BYTES: %d is below the 1 MiB minimum", c.MaxBodyBytes)
	}
	if c.EventRetentionDays < 0 {
		return fmt.Errorf("SPARROW_EVENT_RETENTION_DAYS: must be >= 0, got %d", c.EventRetentionDays)
	}
	return nil
}

// Warnings returns non-fatal configuration advisories the caller should log.
func (c *Config) Warnings() []string {
	var warnings []string
	if u, err := url.Parse(c.DatabaseURL); err == nil {
		host := u.Hostname()
		if u.Query().Get("sslmode") == "disable" && host != "" && host != "localhost" && host != "127.0.0.1" && host != "::1" {
			warnings = append(warnings, fmt.Sprintf("DATABASE_URL uses sslmode=disable with non-local host %q — database traffic is unencrypted", host))
		}
	}
	return warnings
}

// validatePort checks that a port string is a valid TCP port number (1-65535).
// EncryptionKeyring resolves the configured KEK keyring from
// SPARROW_ENCRYPTION_KEYS and SPARROW_ENCRYPTION_PRIMARY_KEY_ID.
func (c *Config) EncryptionKeyring() (*crypto.Keyring, error) {
	entries := make([]string, 0, len(c.EncryptionKeys))
	for _, entry := range c.EncryptionKeys {
		entry = strings.TrimSpace(entry)
		if entry != "" {
			entries = append(entries, entry)
		}
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("SPARROW_ENCRYPTION_KEYS is required (format: <key-id>=<64-char-hex-key>)")
	}

	keys := make([]crypto.Key, 0, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("SPARROW_ENCRYPTION_KEYS: invalid entry %q (want <key-id>=<64-char-hex-key>)", entry)
		}
		id := strings.TrimSpace(parts[0])
		raw := strings.TrimSpace(parts[1])
		if id == "" {
			return nil, fmt.Errorf("SPARROW_ENCRYPTION_KEYS: invalid entry %q (empty key id)", entry)
		}
		key, err := crypto.ParseKey(raw)
		if err != nil {
			return nil, fmt.Errorf("SPARROW_ENCRYPTION_KEYS: invalid key for %q: %w", id, err)
		}
		keys = append(keys, crypto.Key{ID: id, Material: key})
	}

	primaryID := strings.TrimSpace(c.EncryptionPrimaryKeyID)
	if primaryID == "" {
		return nil, fmt.Errorf("SPARROW_ENCRYPTION_PRIMARY_KEY_ID is required when SPARROW_ENCRYPTION_KEYS is set")
	}

	keyring, err := crypto.NewKeyring(keys, primaryID)
	if err != nil {
		if strings.Contains(err.Error(), "primary key id") {
			return nil, fmt.Errorf("SPARROW_ENCRYPTION_PRIMARY_KEY_ID: %w", err)
		}
		return nil, err
	}
	return keyring, nil
}

func validatePort(port, envVar string) error {
	n, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("%s: invalid port %q: %w", envVar, port, err)
	}
	if n < 1 || n > 65535 {
		return fmt.Errorf("%s: port %d out of range (1-65535)", envVar, n)
	}
	return nil
}
