package config

import (
	"strings"
	"testing"
)

func validConfig() *Config {
	return &Config{
		HTTPPort:               "8080",
		DatabaseURL:            "postgres://localhost/riverqueue?sslmode=disable",
		EncryptionKeys:         []string{"new=" + strings.Repeat("ab", 32)},
		EncryptionPrimaryKeyID: "new",
		MaxBodyBytes:           5242880,
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
		{"production with api key", func(c *Config) { c.Environment = "production"; c.APIKey = "k" }, ""},
		{"missing encryption keyring", func(c *Config) { c.EncryptionKeys = nil }, "SPARROW_ENCRYPTION_KEYS"},
		{"legacy single encryption key rejected", func(c *Config) {
			c.EncryptionKeys = nil
			c.EncryptionPrimaryKeyID = ""
			c.EncryptionKey = strings.Repeat("ab", 32)
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
