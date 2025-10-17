package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	DatabaseURL string
}

// Load loads configuration from environment variables
func Load() *Config {
	cfg := &Config{}

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		// Default connection string for local development
		cfg.DatabaseURL = "postgres://localhost/riverqueue?sslmode=disable"
	}

	return cfg
}
