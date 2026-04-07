package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	DatabaseURL   string
	EncryptionKey string // SPARROW_ENCRYPTION_KEY: 64 hex chars (32 bytes) for AES-256-GCM. Optional.
	GRPCPort      string // SPARROW_GRPC_PORT: gRPC listen port. Default "50051".
	HTTPPort      string // SPARROW_HTTP_PORT: HTTP/Connect-RPC listen port. Default "8080".
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

	cfg.GRPCPort = os.Getenv("SPARROW_GRPC_PORT")
	if cfg.GRPCPort == "" {
		cfg.GRPCPort = "50051"
	}

	cfg.HTTPPort = os.Getenv("SPARROW_HTTP_PORT")
	if cfg.HTTPPort == "" {
		cfg.HTTPPort = "8080"
	}

	return cfg
}
