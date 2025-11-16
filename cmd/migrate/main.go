package main

import (
	"context"
	"flag"
	"os"

	"github.com/sarathsp06/sparrow/internal/config"
	"github.com/sarathsp06/sparrow/internal/logger"
	"github.com/sarathsp06/sparrow/internal/migration"
)

func main() {
	// Parse command line flags
	var (
		direction = flag.String("direction", "up", "Migration direction: up, down")
		steps     = flag.Int("steps", 0, "Number of migration steps (0 for all)")
		version   = flag.Uint("version", 0, "Target migration version")
	)
	flag.Parse()

	// Initialize logger
	log := logger.NewLogger("migration")

	// Load configuration
	cfg := config.Load()
	log.Info("Starting database migration",
		"database_url", cfg.DatabaseURL,
		"direction", *direction,
	)

	ctx := context.Background()

	// Run all migrations using the migration package
	if err := migration.RunAllMigrations(ctx, cfg.DatabaseURL, *direction, *steps, *version, log); err != nil {
		log.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	log.Info("All migrations completed successfully")
}
