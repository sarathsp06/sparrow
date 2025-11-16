package migration

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/sarathsp06/sparrow/db"
)

// RunRiverMigrations runs River queue migrations
func RunRiverMigrations(ctx context.Context, databaseURL string, log *slog.Logger) error {
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

// RunAppMigrations runs application schema migrations
func RunAppMigrations(databaseURL, direction string, steps int, targetVersion uint, log *slog.Logger) error {
	log.Info("Running application migrations...")

	// Create database connection for golang-migrate using stdlib
	dbConn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer dbConn.Close() //nolint:errcheck

	// Test the connection
	if err := dbConn.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	driver, err := iofs.New(db.GetMigrationsFS(), "migrations")
	if err != nil {
		return fmt.Errorf("failed to load embedded migration: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithSourceInstance(
		"iofs",
		driver,
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close() //nolint:errcheck

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

// RunAllMigrations runs both River and application migrations in the correct order
func RunAllMigrations(ctx context.Context, databaseURL, direction string, steps int, targetVersion uint, log *slog.Logger) error {
	// Run River migrations first
	if err := RunRiverMigrations(ctx, databaseURL, log); err != nil {
		return fmt.Errorf("failed to run River migrations: %w", err)
	}

	// Run application migrations
	if err := RunAppMigrations(databaseURL, direction, steps, targetVersion, log); err != nil {
		return fmt.Errorf("failed to run application migrations: %w", err)
	}

	return nil
}
