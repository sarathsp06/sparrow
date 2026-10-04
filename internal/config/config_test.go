package config

import (
	"strings"
	"testing"
	"time"
)

func validConfig() *Config {
	return &Config{
		HTTPPort:                 "8080",
		DatabaseURL:              "postgres://localhost/riverqueue?sslmode=disable",
		EncryptionKeys:           []string{"new=" + strings.Repeat("ab", 32)},
		EncryptionPrimaryKeyID:   "new",
		MaxBodyBytes:             5242880,
		MaxCapturedResponseBytes: 1 << 20,
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{"valid", func(c *Config) {}, ""},
		{"production without api key", func(c *Config) { c.Environment = "production" }, "SPARROW_API_KEY"},
		{"production with short api key", func(c *Config) { c.Environment = "production"; c.APIKey = "k" }, "at least 32"},
		{"production with api key", func(c *Config) { c.Environment = "production"; c.APIKey = strings.Repeat("k", 32) }, ""},
		{"captured response cap too small", func(c *Config) { c.MaxCapturedResponseBytes = 10 }, "SPARROW_MAX_CAPTURED_RESPONSE_BYTES"},
		{"negative token ttl", func(c *Config) { c.TokenDefaultTTL = -time.Hour }, "SPARROW_TOKEN_DEFAULT_TTL"},
		{"invalid allowed networks", func(c *Config) { c.AllowedNetworks = []string{"nope"} }, "SPARROW_ALLOWED_NETWORKS"},
		{"valid allowed networks", func(c *Config) { c.AllowedNetworks = []string{"10.0.0.0/8", "fd00::1"} }, ""},
		{"missing encryption keyring", func(c *Config) { c.EncryptionKeys = nil }, "SPARROW_ENCRYPTION_KEYS"},
		{"legacy single encryption key no longer satisfies config", func(c *Config) {
			c.EncryptionKeys = nil
			c.EncryptionPrimaryKeyID = ""
		}, "SPARROW_ENCRYPTION_KEYS"},
		{"invalid keyring hex", func(c *Config) { c.EncryptionKeys = []string{"new=" + strings.Repeat("zz", 32)} }, "SPARROW_ENCRYPTION_KEYS"},
		{"short keyring key", func(c *Config) { c.EncryptionKeys = []string{"new=abcd"} }, "SPARROW_ENCRYPTION_KEYS"},
		{"valid encryption keyring", func(c *Config) {
			c.EncryptionKeys = []string{
				"old=" + strings.Repeat("ab", 32),
				"new=" + strings.Repeat("cd", 32),
			}
			c.EncryptionPrimaryKeyID = "new"
		}, ""},
		{"single-key primary missing", func(c *Config) {
			c.EncryptionKeys = []string{"main=" + strings.Repeat("ab", 32)}
			c.EncryptionPrimaryKeyID = ""
		}, "SPARROW_ENCRYPTION_PRIMARY_KEY_ID"},
		{"keyring primary missing", func(c *Config) {
			c.EncryptionKeys = []string{
				"old=" + strings.Repeat("ab", 32),
				"new=" + strings.Repeat("cd", 32),
			}
			c.EncryptionPrimaryKeyID = ""
		}, "SPARROW_ENCRYPTION_PRIMARY_KEY_ID"},
		{"keyring primary not found", func(c *Config) {
			c.EncryptionKeys = []string{
				"old=" + strings.Repeat("ab", 32),
				"new=" + strings.Repeat("cd", 32),
			}
			c.EncryptionPrimaryKeyID = "missing"
		}, "SPARROW_ENCRYPTION_PRIMARY_KEY_ID"},
		{"negative auto-disable window", func(c *Config) { c.AutoDisableAfter = -time.Hour }, "SPARROW_AUTO_DISABLE_AFTER"},
		{"auto-disable off", func(c *Config) { c.AutoDisableAfter = 0 }, ""},
		{"negative auto-disable failure minimum", func(c *Config) { c.AutoDisableMinFailures = -1 }, "SPARROW_AUTO_DISABLE_MIN_FAILURES"},
		{"body limit below minimum", func(c *Config) { c.MaxBodyBytes = 1024 }, "SPARROW_MAX_BODY_BYTES"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)
			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestWarnings(t *testing.T) {
	cfg := validConfig()
	if w := cfg.Warnings(); len(w) != 0 {
		t.Fatalf("Warnings() for localhost = %v, want none", w)
	}

	cfg.DatabaseURL = "postgres://db.internal.example.com/riverqueue?sslmode=disable"
	w := cfg.Warnings()
	if len(w) != 1 || !strings.Contains(w[0], "sslmode=disable") {
		t.Fatalf("Warnings() for remote sslmode=disable = %v, want one advisory", w)
	}
}

func TestUIInjectKeyIsRetired(t *testing.T) {
	t.Setenv("SPARROW_UI_INJECT_KEY", "true")
	cfg := &Config{DatabaseURL: "postgres://localhost/db"}
	w := cfg.Warnings()
	if len(w) != 1 || !strings.Contains(w[0], "SPARROW_UI_INJECT_KEY") {
		t.Fatalf("Warnings() with SPARROW_UI_INJECT_KEY set = %v, want one retirement notice", w)
	}
}

func TestAIConfig(t *testing.T) {
	base := func() *Config {
		return &Config{HTTPPort: "8080", DatabaseURL: "postgres://localhost/x", EncryptionKeys: []string{"main=" + strings.Repeat("00", 32)}, EncryptionPrimaryKeyID: "main", MaxCapturedResponseBytes: 1024, MaxBodyBytes: 1 << 20}
	}
	cases := []struct {
		name    string
		mut     func(c *Config)
		wantErr string
		enabled bool
		model   string
	}{
		{"unset is off", func(c *Config) {}, "", false, "claude-haiku-4-5"},
		{"anthropic key enables with default model", func(c *Config) { c.AIAPIKey = "k" }, "", true, "claude-haiku-4-5"},
		{"anthropic explicit model", func(c *Config) { c.AIAPIKey = "k"; c.AIModel = "claude-sonnet-5-5" }, "", true, "claude-sonnet-5-5"},
		{"openai without base url is off", func(c *Config) { c.AIProvider = "openai" }, "", false, ""},
		{"openai with base url needs model", func(c *Config) { c.AIProvider = "openai"; c.AIBaseURL = "http://localhost:11434/v1" }, "SPARROW_AI_MODEL is required", true, ""},
		{"openai local server, no key", func(c *Config) {
			c.AIProvider = "openai"
			c.AIBaseURL = "http://localhost:11434/v1"
			c.AIModel = "llama3.2"
		}, "", true, "llama3.2"},
		{"openai key without base url", func(c *Config) { c.AIProvider = "openai"; c.AIAPIKey = "k"; c.AIModel = "m" }, "SPARROW_AI_BASE_URL is required", true, "m"},
		{"unknown provider", func(c *Config) { c.AIProvider = "gemini" }, "not supported", false, ""},
		{"bad base url", func(c *Config) { c.AIAPIKey = "k"; c.AIBaseURL = "localhost:11434" }, "not an absolute URL", true, "claude-haiku-4-5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := base()
			tc.mut(c)
			err := c.Validate()
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("Validate() = %v", err)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Fatalf("Validate() = %v, want containing %q", err, tc.wantErr)
			}
			if c.AIEnabled() != tc.enabled {
				t.Fatalf("AIEnabled() = %v, want %v", c.AIEnabled(), tc.enabled)
			}
			if c.AIModelOrDefault() != tc.model {
				t.Fatalf("AIModelOrDefault() = %q, want %q", c.AIModelOrDefault(), tc.model)
			}
		})
	}
}
