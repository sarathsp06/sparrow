package main

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/sarathsp06/httpqueue/internal/config"
	"github.com/sarathsp06/httpqueue/internal/logger"
)

func main() {
	// Initialize logger
	log := logger.NewLogger("migration")

	// Load configuration
	cfg := config.Load()
	log.Info("Starting River database migration",
		"database_url", cfg.DatabaseURL,
	)

	ctx := context.Background()

	// Connect to database
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("Failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// Test database connection
	if err := dbPool.Ping(ctx); err != nil {
		log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	log.Info("Connected to database successfully")

	// Create migrator
	migrator, err := rivermigrate.New(riverpgxv5.New(dbPool), nil)
	if err != nil {
		log.Error("Failed to create migrator", "error", err)
		os.Exit(1)
	}

	// Get migration status
	res, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{})
	if err != nil {
		log.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	log.Info("Database migration completed successfully",
		"migrations_run", len(res.Versions),
	)

	// Log each migration that was applied
	for _, version := range res.Versions {
		log.Info("Applied migration",
			"version", version.Version,
			"name", version.Name,
		)
	}

	if len(res.Versions) == 0 {
		log.Info("No migrations needed - database is already up to date")
	}
}
