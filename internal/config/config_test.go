package config

import (
	"strings"
	"testing"
)

func validConfig() *Config {
	return &Config{
		HTTPPort:      "8080",
		DatabaseURL:   "postgres://localhost/riverqueue?sslmode=disable",
		EncryptionKey: strings.Repeat("ab", 32),
		MaxBodyBytes:  5242880,
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
		{"missing encryption key", func(c *Config) { c.EncryptionKey = "" }, "SPARROW_ENCRYPTION_KEY"},
		{"non-hex encryption key", func(c *Config) { c.EncryptionKey = strings.Repeat("zz", 32) }, "SPARROW_ENCRYPTION_KEY"},
		{"short encryption key", func(c *Config) { c.EncryptionKey = "abcd" }, "SPARROW_ENCRYPTION_KEY"},
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
