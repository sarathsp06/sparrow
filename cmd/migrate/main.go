package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

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

	ctx := context.Background()

	log := slog.Default().With("component", "migration")

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://localhost/riverqueue?sslmode=disable"
	}

	log.InfoContext(ctx, "Starting database migration",
		"database_url", databaseURL,
		"direction", *direction,
	)

	// Run all migrations using the migration package
	if err := migration.RunAllMigrations(ctx, databaseURL, *direction, *steps, *version, log); err != nil {
		log.ErrorContext(ctx, "Failed to run migrations", "error", err)
		os.Exit(1)
	}

	log.InfoContext(ctx, "All migrations completed successfully")
}
