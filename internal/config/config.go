// Package config provides structured configuration loading from environment
// variables using [kelseyhightower/envconfig].
//
// All server configuration is defined in the [Config] struct. Call [Load] to
// populate it from the environment. Non-SPARROW-prefixed variables (DATABASE_URL,
// ENVIRONMENT, OTEL_EXPORTER_OTLP_ENDPOINT, CORS_ALLOWED_ORIGINS) are loaded
// separately because envconfig works with a single prefix.
package config

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"

	"github.com/sarathsp06/sparrow/internal/crypto"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
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

	// LogLevel is the lowest level the server logs: debug, info, warn or
	// error. info logs every event and delivery; warn keeps only problems,
	// which saves the per-event log work on a busy instance.
	// Env: SPARROW_LOG_LEVEL
	LogLevel string `envconfig:"SPARROW_LOG_LEVEL" default:"info"`

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

	// OTLPEndpoint is the OpenTelemetry OTLP export endpoint, as a URL
	// (http://collector:4318, https://... for TLS). When empty, OTel export
	// is disabled. The other standard OTEL_EXPORTER_OTLP_* variables
	// (headers, TLS certificates, per-signal endpoints) are read directly by
	// the exporters.
	// Env: OTEL_EXPORTER_OTLP_ENDPOINT
	OTLPEndpoint string `envconfig:"OTEL_EXPORTER_OTLP_ENDPOINT" default:""`

	// OTLPProtocol selects the OTLP transport: "http/protobuf" (default) or
	// "grpc".
	// Env: OTEL_EXPORTER_OTLP_PROTOCOL
	OTLPProtocol string `envconfig:"OTEL_EXPORTER_OTLP_PROTOCOL" default:""`

	// OTLPSignals is a comma-separated subset of "traces", "metrics", "logs"
	// to send to the OTLP endpoint. Default: all three. Set it to "traces"
	// for a traces-only backend such as Jaeger or Tempo; metrics remain
	// scrapeable at /metrics either way.
	// Env: SPARROW_OTLP_SIGNALS
	OTLPSignals []string `envconfig:"SPARROW_OTLP_SIGNALS" default:"traces,metrics,logs"`

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

	// AutoDisableAfter pauses a webhook automatically once its receiver has
	// failed every attempt for this long (and at least
	// AutoDisableMinFailures attempts in a row). The webhook is paused, not
	// deleted: it is marked with the reason, a sparrow.webhook.disabled
	// system event is emitted, and resuming it re-enables delivery. 0
	// turns automatic disabling off.
	// Env: SPARROW_AUTO_DISABLE_AFTER (Go duration, default "120h" = 5 days)
	AutoDisableAfter time.Duration `envconfig:"SPARROW_AUTO_DISABLE_AFTER" default:"120h"`

	// AutoDisableMinFailures is the minimum run of consecutive failed
	// attempts before a webhook can be auto-disabled, so a receiver that saw
	// a single failure long ago is never paused by the next one. 0 means
	// the default.
	// Env: SPARROW_AUTO_DISABLE_MIN_FAILURES (default 10)
	AutoDisableMinFailures int `envconfig:"SPARROW_AUTO_DISABLE_MIN_FAILURES" default:"10"`

	// EventWorkers is the number of concurrent event fan-out jobs (one per
	// pushed event). Env: SPARROW_EVENT_WORKERS (default 20)
	EventWorkers int `envconfig:"SPARROW_EVENT_WORKERS" default:"20"`

	// WebhookWorkers is the number of concurrent webhook deliveries.
	// Env: SPARROW_WEBHOOK_WORKERS (default 20)
	WebhookWorkers int `envconfig:"SPARROW_WEBHOOK_WORKERS" default:"20"`

	// QueueFetchCooldown is the minimum interval between two job fetches on
	// a queue. Each fetch takes as many jobs as there are idle workers, so
	// workers/cooldown is the hard throughput ceiling of a queue (20 workers
	// at 20ms is about 1000 jobs/s). Lower values poll Postgres more often
	// under load; idle queues are not affected.
	// Env: SPARROW_QUEUE_FETCH_COOLDOWN (Go duration, default "20ms")
	QueueFetchCooldown time.Duration `envconfig:"SPARROW_QUEUE_FETCH_COOLDOWN" default:"20ms"`

	// HealthEvalInterval is how often delivery outcomes are folded into
	// webhook health: the health label, health_changed alerts and
	// auto-disable lag outcomes by at most this much. The evaluation is
	// incremental (only outcomes since the last run), so a busy webhook does
	// not make it more expensive. Metrics remain the real-time signal.
	// Env: SPARROW_HEALTH_EVAL_INTERVAL (Go duration, default "1m")
	HealthEvalInterval time.Duration `envconfig:"SPARROW_HEALTH_EVAL_INTERVAL" default:"1m"`

	// ShutdownTimeout is the whole budget for a graceful shutdown on SIGTERM
	// or SIGINT: /ready turns 503, in-flight HTTP requests finish, then the
	// queue stops taking jobs and waits for running ones. Jobs still running
	// near the end of the budget are cancelled and retried later. Keep it
	// below the orchestrator's kill deadline (Kubernetes:
	// terminationGracePeriodSeconds, 30s by default).
	// Env: SPARROW_SHUTDOWN_TIMEOUT (Go duration, default "10s")
	ShutdownTimeout time.Duration `envconfig:"SPARROW_SHUTDOWN_TIMEOUT" default:"10s"`

	// MetricsEnabled serves the OpenTelemetry metrics in Prometheus text
	// format at GET /metrics (unauthenticated, like /health), independent of
	// OTLP export.
	// Env: SPARROW_METRICS_ENABLED (default true)
	MetricsEnabled bool `envconfig:"SPARROW_METRICS_ENABLED" default:"true"`

	// AutoRegisterEvents lets a push to an unregistered event name create a
	// schema-less event type instead of returning 404. Off by default: event
	// types are never deleted, so in production a producer typo would become
	// a permanent name. Turn it on for local development.
	AutoRegisterEvents bool `envconfig:"SPARROW_AUTO_REGISTER_EVENTS" default:"false"`
	// AIProvider selects the chat API behind AI-assisted template drafting:
	// "anthropic" (default; needs SPARROW_AI_API_KEY) or "openai" for any
	// OpenAI-compatible /v1/chat/completions server such as Ollama, vLLM,
	// LM Studio, llama.cpp, OpenRouter or OpenAI itself (needs
	// SPARROW_AI_BASE_URL and SPARROW_AI_MODEL; the key is optional for
	// local servers).
	// Env: SPARROW_AI_PROVIDER
	AIProvider string `envconfig:"SPARROW_AI_PROVIDER" default:"anthropic"`

	// AIDrafting turns AI-assisted template drafting ("Draft with AI" and
	// POST /v1/subscriptions:draftTemplate) on. It is the only switch: the
	// provider settings below are read only when it is true, and a missing
	// or bad one is reported by the draft endpoint (AIConfigError), not at
	// startup. The copy-and-paste prompt works either way.
	// Env: SPARROW_AI_ENABLED (default false)
	AIDrafting bool `envconfig:"SPARROW_AI_ENABLED" default:"false"`

	// AIAPIKey is the provider API key: required for the anthropic provider
	// when AIDrafting is on, and sent as a Bearer token for the openai
	// provider when set.
	// Env: SPARROW_AI_API_KEY
	AIAPIKey string `envconfig:"SPARROW_AI_API_KEY" default:""`

	// AIModel is the model used for drafting. Drafts are short and every
	// draft is render-verified and repaired, so a light model is the
	// default; set a larger one if drafts need more than a couple of
	// repair rounds. Required for the openai provider.
	// Env: SPARROW_AI_MODEL
	AIModel string `envconfig:"SPARROW_AI_MODEL" default:""`

	// AIBaseURL is the API base URL. Required for the openai provider
	// (e.g. http://localhost:11434/v1 for Ollama); optional for anthropic
	// (an internal gateway or proxy).
	// Env: SPARROW_AI_BASE_URL
	AIBaseURL string `envconfig:"SPARROW_AI_BASE_URL" default:""`

	// AITimeout bounds one POST /v1/subscriptions:draftTemplate, every
	// draft → render → repair round included. The server's 30s write
	// timeout is lifted to match for that request only, so a slow model or
	// a few repair rounds are not cut off mid-draft.
	// Env: SPARROW_AI_TIMEOUT (Go duration, default "3m")
	AITimeout time.Duration `envconfig:"SPARROW_AI_TIMEOUT" default:"3m"`
}

// Load populates a Config struct from environment variables.
// envconfig does not use a prefix since the env vars span multiple consumers
// (SPARROW_*, DATABASE_URL, ENVIRONMENT, OTEL_*, CORS_*).
//
// A variable set to "" or "-" counts as unset, so its default applies: some
// platforms cannot unset a variable once it exists, or cannot store an empty
// value. Load unsets such variables in the process environment, so code that
// reads them directly (the OTel exporters) agrees.
func Load() (*Config, error) {
	unsetBlankEnv()
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("loading config from environment: %w", err)
	}
	return &cfg, nil
}

// isBlank reports whether an environment value means "unset": empty,
// whitespace, or a lone "-".
func isBlank(v string) bool {
	v = strings.TrimSpace(v)
	return v == "" || v == "-"
}

// unsetBlankEnv unsets every variable Config reads whose value is blank.
func unsetBlankEnv() {
	t := reflect.TypeOf(Config{})
	for i := range t.NumField() {
		key := t.Field(i).Tag.Get("envconfig")
		if v, ok := os.LookupEnv(key); key != "" && ok && isBlank(v) {
			_ = os.Unsetenv(key)
		}
	}
}

// SlogLevel parses LogLevel: debug, info, warn or error (any case).
func (c *Config) SlogLevel() (slog.Level, error) {
	var level slog.Level
	switch strings.ToLower(strings.TrimSpace(c.LogLevel)) {
	case "debug":
		level = slog.LevelDebug
	case "", "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return level, fmt.Errorf("SPARROW_LOG_LEVEL: %q is not one of debug, info, warn, error", c.LogLevel)
	}
	return level, nil
}

// DefaultShutdownTimeout is SPARROW_SHUTDOWN_TIMEOUT's default.
const DefaultShutdownTimeout = 10 * time.Second

// ShutdownTimeoutOrDefault returns ShutdownTimeout, or the default when it is
// unset (a Config built in code without envconfig defaults).
func (c *Config) ShutdownTimeoutOrDefault() time.Duration {
	if c.ShutdownTimeout <= 0 {
		return DefaultShutdownTimeout
	}
	return c.ShutdownTimeout
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
	if ids := c.zeroEncryptionKeyIDs(); c.IsProduction() && len(ids) > 0 {
		return fmt.Errorf("SPARROW_ENCRYPTION_KEYS: key %s is all zeros, the published development key; generate a real one with: openssl rand -hex 32", strings.Join(ids, ", "))
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if _, err := c.SlogLevel(); err != nil {
		return err
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
	if c.AutoDisableAfter < 0 {
		return fmt.Errorf("SPARROW_AUTO_DISABLE_AFTER: must be >= 0, got %s", c.AutoDisableAfter)
	}
	if c.AutoDisableMinFailures < 0 {
		return fmt.Errorf("SPARROW_AUTO_DISABLE_MIN_FAILURES: must be >= 0, got %d", c.AutoDisableMinFailures)
	}
	if c.ShutdownTimeout < 0 {
		return fmt.Errorf("SPARROW_SHUTDOWN_TIMEOUT: must be > 0, got %s", c.ShutdownTimeout)
	}
	if c.EventRetentionDays < 0 {
		return fmt.Errorf("SPARROW_EVENT_RETENTION_DAYS: must be >= 0, got %d", c.EventRetentionDays)
	}
	return nil
}

// AIConfigError reports what is missing or wrong in the provider settings
// when drafting is on, or nil when a draft can be attempted. The server
// still starts: the draft endpoint answers with this error, so a deploy
// with drafting on and a forgotten key keeps the copy-and-paste prompt.
func (c *Config) AIConfigError() error {
	if !c.AIDrafting {
		return nil
	}
	switch c.aiProvider() {
	case "anthropic":
		if c.AIAPIKey == "" {
			return fmt.Errorf("SPARROW_AI_ENABLED=true but SPARROW_AI_API_KEY is not set (required for SPARROW_AI_PROVIDER=anthropic)")
		}
	case "openai":
		if c.AIBaseURL == "" {
			return fmt.Errorf("SPARROW_AI_ENABLED=true but SPARROW_AI_BASE_URL is not set (required for SPARROW_AI_PROVIDER=openai, e.g. http://localhost:11434/v1)")
		}
		if strings.TrimSpace(c.AIModel) == "" {
			return fmt.Errorf("SPARROW_AI_ENABLED=true but SPARROW_AI_MODEL is not set (required for SPARROW_AI_PROVIDER=openai, e.g. llama3.2, qwen2.5-coder, gpt-4o-mini)")
		}
	default:
		return fmt.Errorf("SPARROW_AI_PROVIDER: %q is not supported (want anthropic or openai)", c.AIProvider)
	}
	if c.AIBaseURL != "" {
		if u, err := url.Parse(c.AIBaseURL); err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("SPARROW_AI_BASE_URL: %q is not an absolute URL", c.AIBaseURL)
		}
	}
	return nil
}

// MinAPIKeyLength is the shortest master key accepted in production. 32
// characters is 128+ bits for a random hex or base64 key.
const MinAPIKeyLength = 32

// DefaultAnthropicModel is the drafting model when SPARROW_AI_MODEL is unset
// with the anthropic provider. Drafts are short and render-verified, so a
// light model does the job; operators can raise it.
const DefaultAnthropicModel = "claude-haiku-4-5"

// aiProvider normalises an unset provider to the default, so a Config built
// in code without envconfig defaults behaves like one loaded from the env.
func (c *Config) aiProvider() string {
	if strings.TrimSpace(c.AIProvider) == "" {
		return "anthropic"
	}
	return strings.ToLower(strings.TrimSpace(c.AIProvider))
}

// AIProviderName returns the effective provider: "anthropic" or "openai".
func (c *Config) AIProviderName() string { return c.aiProvider() }

// AIEnabled reports whether SPARROW_AI_ENABLED turned drafting on.
func (c *Config) AIEnabled() bool { return c.AIDrafting }

// AIModelOrDefault returns the configured model, or the provider default.
func (c *Config) AIModelOrDefault() string {
	if m := strings.TrimSpace(c.AIModel); m != "" {
		return m
	}
	if c.aiProvider() == "anthropic" {
		return DefaultAnthropicModel
	}
	return ""
}

// DefaultAutoDisableMinFailures is the failure-run minimum when
// SPARROW_AUTO_DISABLE_MIN_FAILURES is unset or 0.
const DefaultAutoDisableMinFailures = 10

// AutoDisableMinFailuresOrDefault returns the configured minimum run of
// consecutive failures, or DefaultAutoDisableMinFailures.
func (c *Config) AutoDisableMinFailuresOrDefault() int {
	if c.AutoDisableMinFailures > 0 {
		return c.AutoDisableMinFailures
	}
	return DefaultAutoDisableMinFailures
}

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
	if ids := c.zeroEncryptionKeyIDs(); len(ids) > 0 && !c.IsProduction() {
		warnings = append(warnings, fmt.Sprintf("SPARROW_ENCRYPTION_KEYS: key %s is all zeros, the published development key — anyone can decrypt webhook secrets stored with it, and production refuses it (generate one with: openssl rand -hex 32)", strings.Join(ids, ", ")))
	}
	if !c.AIDrafting && (c.AIAPIKey != "" || c.AIBaseURL != "") {
		warnings = append(warnings, "SPARROW_AI_API_KEY or SPARROW_AI_BASE_URL is set but SPARROW_AI_ENABLED is not true, so Draft with AI is off (the copy-and-paste prompt still works); set SPARROW_AI_ENABLED=true to turn it on")
	}
	if err := c.AIConfigError(); err != nil {
		warnings = append(warnings, err.Error()+"; POST /v1/subscriptions:draftTemplate answers 503 with this message until it is fixed")
	}
	if v, set := os.LookupEnv("SPARROW_UI_INJECT_KEY"); set && !isBlank(v) {
		warnings = append(warnings, "SPARROW_UI_INJECT_KEY is no longer supported and is ignored — the UI never receives the API key; sign in with the key, an access token, or an invite link")
	}
	return warnings
}

// zeroEncryptionKeyIDs returns the IDs of keyring keys that are all zeros:
// the development default in scripts/dev-env.sh and the compose files,
// public and so no secret at all. Malformed entries are skipped; Validate
// reports them.
func (c *Config) zeroEncryptionKeyIDs() []string {
	var ids []string
	for _, entry := range c.EncryptionKeys {
		id, raw, ok := strings.Cut(strings.TrimSpace(entry), "=")
		if !ok {
			continue
		}
		key, err := crypto.ParseKey(strings.TrimSpace(raw))
		if err != nil || len(key) == 0 {
			continue
		}
		if bytes.Count(key, []byte{0}) == len(key) {
			ids = append(ids, strings.TrimSpace(id))
		}
	}
	return ids
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
