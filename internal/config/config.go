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
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
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

	// MaxCapturedResponseBytes caps the stored response body of webhooks
	// with capture_response_body enabled (others always store 1 KiB). Lower
	// it to bound what receivers can make Sparrow store.
	// Env: SPARROW_MAX_CAPTURED_RESPONSE_BYTES (default 1 MiB)
	MaxCapturedResponseBytes int64 `envconfig:"SPARROW_MAX_CAPTURED_RESPONSE_BYTES" default:"1048576"`

	// TokenDefaultTTL is the lifetime of a tenant-wide access token created
	// without an explicit TTL or never_expires. 0 restores tokens that never
	// expire by default.
	// Env: SPARROW_TOKEN_DEFAULT_TTL (Go duration, e.g. "2160h" = 90 days)
	TokenDefaultTTL time.Duration `envconfig:"SPARROW_TOKEN_DEFAULT_TTL" default:"2160h"`

	// AllowPrivateNetworks relaxes SSRF protection to allow localhost and
	// private IP addresses as webhook target URLs. Useful for local dev.
	// Env: SPARROW_ALLOW_PRIVATE_NETWORKS
	AllowPrivateNetworks bool `envconfig:"SPARROW_ALLOW_PRIVATE_NETWORKS" default:"false"`

	// AllowedNetworks lists CIDRs (or bare IPs) webhook deliveries may reach
	// in addition to public addresses — e.g. internal services on a VPN —
	// while loopback, cloud metadata, and the rest of the private address
	// space stay blocked. Prefer this over AllowPrivateNetworks in
	// production. Cloud metadata endpoints are reachable only if listed here.
	// Env: SPARROW_ALLOWED_NETWORKS (comma-separated, e.g. "10.20.0.0/16,fd12::/48")
	AllowedNetworks []string `envconfig:"SPARROW_ALLOWED_NETWORKS"`

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
	if c.IsProduction() && len(c.APIKey) < MinAPIKeyLength {
		return fmt.Errorf("SPARROW_API_KEY must be at least %d characters in production (generate one with: openssl rand -hex 32)", MinAPIKeyLength)
	}
	if _, err := client.ParseNetworks(c.AllowedNetworks); err != nil {
		return fmt.Errorf("SPARROW_ALLOWED_NETWORKS: %w", err)
	}
	if c.MaxCapturedResponseBytes < 1024 {
		return fmt.Errorf("SPARROW_MAX_CAPTURED_RESPONSE_BYTES: %d is below the 1024-byte minimum", c.MaxCapturedResponseBytes)
	}
	if c.TokenDefaultTTL < 0 {
		return fmt.Errorf("SPARROW_TOKEN_DEFAULT_TTL: must be >= 0, got %s", c.TokenDefaultTTL)
	}
	if c.MaxBodyBytes < 1<<20 {
		return fmt.Errorf("SPARROW_MAX_BODY_BYTES: %d is below the 1 MiB minimum", c.MaxBodyBytes)
	}
	if c.EventRetentionDays < 0 {
		return fmt.Errorf("SPARROW_EVENT_RETENTION_DAYS: must be >= 0, got %d", c.EventRetentionDays)
	}
	return nil
}

// MinAPIKeyLength is the shortest master key accepted in production. 32
// characters is 128+ bits for a random hex or base64 key.
const MinAPIKeyLength = 32

// AllowedNetworkList returns the parsed SPARROW_ALLOWED_NETWORKS entries.
// Validate reports parse errors, so callers after Validate can ignore them.
func (c *Config) AllowedNetworkList() []*net.IPNet {
	nets, _ := client.ParseNetworks(c.AllowedNetworks)
	return nets
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
	if c.APIKey != "" && len(c.APIKey) < MinAPIKeyLength && !c.IsProduction() {
		warnings = append(warnings, fmt.Sprintf("SPARROW_API_KEY is shorter than %d characters — fine for local testing, but production refuses it (generate one with: openssl rand -hex 32)", MinAPIKeyLength))
	}
	if _, set := os.LookupEnv("SPARROW_UI_INJECT_KEY"); set {
		warnings = append(warnings, "SPARROW_UI_INJECT_KEY is no longer supported and is ignored — the UI never receives the API key; sign in with the key, an access token, or an invite link")
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
