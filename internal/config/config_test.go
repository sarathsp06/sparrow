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
