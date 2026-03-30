package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	DatabaseURL   string
	EncryptionKey string // SPARROW_ENCRYPTION_KEY: 64 hex chars (32 bytes) for AES-256-GCM. Optional.
}

// Load loads configuration from environment variables
func Load() *Config {
	cfg := &Config{}

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		// Default connection string for local development
		cfg.DatabaseURL = "postgres://localhost/riverqueue?sslmode=disable"
	}

	cfg.EncryptionKey = os.Getenv("SPARROW_ENCRYPTION_KEY")

	return cfg
}
