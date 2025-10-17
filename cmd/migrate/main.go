package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/sarathsp06/httpqueue/internal/config"
	"github.com/sarathsp06/httpqueue/internal/logger"
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

	// Run River migrations first
	if err := runRiverMigrations(ctx, cfg.DatabaseURL, log); err != nil {
		log.Error("Failed to run River migrations", "error", err)
		os.Exit(1)
	}

	// Run application migrations
	if err := runAppMigrations(cfg.DatabaseURL, *direction, *steps, *version, log); err != nil {
		log.Error("Failed to run application migrations", "error", err)
		os.Exit(1)
	}

	log.Info("All migrations completed successfully")
}

func runRiverMigrations(ctx context.Context, databaseURL string, log *slog.Logger) error {
	log.Info("Running River queue migrations...")

	// Connect to database
	dbPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create database pool: %w", err)
	}
	defer dbPool.Close()

	// Test database connection
	if err := dbPool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Create River migrator
	migrator, err := rivermigrate.New(riverpgxv5.New(dbPool), nil)
	if err != nil {
		return fmt.Errorf("failed to create River migrator: %w", err)
	}

	// Run River migrations
	res, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{})
	if err != nil {
		return fmt.Errorf("failed to run River migrations: %w", err)
	}

	log.Info("River migrations completed",
		"migrations_run", len(res.Versions),
	)

	// Log each migration that was applied
	for _, version := range res.Versions {
		log.Info("Applied River migration",
			"version", version.Version,
			"name", version.Name,
		)
	}

	if len(res.Versions) == 0 {
		log.Info("No River migrations needed - database is already up to date")
	}

	return nil
}

func runAppMigrations(databaseURL, direction string, steps int, targetVersion uint, log *slog.Logger) error {
	log.Info("Running application migrations...")

	// Create database connection for golang-migrate using stdlib
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	// Test the connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create database driver for golang-migrate
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		"file://db/migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	// Get current version and dirty state
	currentVersion, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	if dirty {
		log.Warn("Database is in dirty state, forcing version", "version", currentVersion)
		if err := m.Force(int(currentVersion)); err != nil {
			return fmt.Errorf("failed to force version: %w", err)
		}
	}

	log.Info("Current migration state",
		"version", currentVersion,
		"dirty", dirty,
	)

	// Execute migrations based on direction
	switch direction {
	case "up":
		if targetVersion > 0 {
			log.Info("Migrating to specific version", "target_version", targetVersion)
			if err := m.Migrate(targetVersion); err != nil && err != migrate.ErrNoChange {
				return fmt.Errorf("failed to migrate to version %d: %w", targetVersion, err)
			}
		} else if steps > 0 {
			log.Info("Migrating up with steps", "steps", steps)
			if err := m.Steps(steps); err != nil && err != migrate.ErrNoChange {
				return fmt.Errorf("failed to migrate %d steps up: %w", steps, err)
			}
		} else {
			log.Info("Migrating to latest version")
			if err := m.Up(); err != nil && err != migrate.ErrNoChange {
				return fmt.Errorf("failed to migrate up: %w", err)
			}
		}

	case "down":
		if targetVersion > 0 {
			log.Info("Migrating down to specific version", "target_version", targetVersion)
			if err := m.Migrate(targetVersion); err != nil && err != migrate.ErrNoChange {
				return fmt.Errorf("failed to migrate to version %d: %w", targetVersion, err)
			}
		} else if steps > 0 {
			log.Info("Migrating down with steps", "steps", steps)
			if err := m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
				return fmt.Errorf("failed to migrate %d steps down: %w", steps, err)
			}
		} else {
			log.Info("Migrating down one step")
			if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
				return fmt.Errorf("failed to migrate down: %w", err)
			}
		}

	default:
		return fmt.Errorf("invalid direction: %s (must be 'up' or 'down')", direction)
	}

	// Get final version
	finalVersion, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get final migration version: %w", err)
	}

	log.Info("Application migrations completed",
		"final_version", finalVersion,
		"dirty", dirty,
	)

	return nil
}
