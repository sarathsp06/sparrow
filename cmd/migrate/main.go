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

	// Create webhook tables
	log.Info("Creating webhook tables...")
	if err := createWebhookTables(ctx, dbPool); err != nil {
		log.Error("Failed to create webhook tables", "error", err)
		os.Exit(1)
	}
	log.Info("Webhook tables created successfully")
}

func createWebhookTables(ctx context.Context, db *pgxpool.Pool) error {
	// Read and execute webhook schema (with multiple events support)
	schema := `
		-- Create webhook_registrations table with multiple events support
		CREATE TABLE IF NOT EXISTS webhook_registrations (
			id VARCHAR(36) PRIMARY KEY,
			namespace VARCHAR(255) NOT NULL,
			events JSONB NOT NULL,
			url TEXT NOT NULL,
			headers JSONB DEFAULT '{}',
			timeout INTEGER NOT NULL DEFAULT 30,
			active BOOLEAN NOT NULL DEFAULT true,
			description TEXT,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		-- Create indexes for efficient lookups using JSONB operators
		CREATE INDEX IF NOT EXISTS idx_webhook_registrations_namespace 
			ON webhook_registrations(namespace) WHERE active = true;
		CREATE INDEX IF NOT EXISTS idx_webhook_registrations_events 
			ON webhook_registrations USING GIN(events) WHERE active = true;

		-- Create event_records table
		CREATE TABLE IF NOT EXISTS event_records (
			id VARCHAR(36) PRIMARY KEY,
			namespace VARCHAR(255) NOT NULL,
			event VARCHAR(255) NOT NULL,
			payload TEXT NOT NULL,
			ttl BIGINT NOT NULL,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL
		);

		-- Create indexes for event records
		CREATE INDEX IF NOT EXISTS idx_event_records_namespace_event 
			ON event_records(namespace, event);
		CREATE INDEX IF NOT EXISTS idx_event_records_expires_at 
			ON event_records(expires_at);

		-- Create webhook_deliveries table
		CREATE TABLE IF NOT EXISTS webhook_deliveries (
			id VARCHAR(36) PRIMARY KEY,
			webhook_id VARCHAR(36) NOT NULL,
			event_id VARCHAR(36) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			attempt_count INTEGER NOT NULL DEFAULT 0,
			max_attempts INTEGER NOT NULL DEFAULT 3,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			last_attempted_at TIMESTAMP WITH TIME ZONE,
			next_retry_at TIMESTAMP WITH TIME ZONE,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			response_code INTEGER DEFAULT 0,
			response_body TEXT DEFAULT '',
			error_message TEXT DEFAULT ''
		);

		-- Create indexes for webhook deliveries
		CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id 
			ON webhook_deliveries(webhook_id);
		CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_event_id 
			ON webhook_deliveries(event_id);
		CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status 
			ON webhook_deliveries(status);
		CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_expires_at 
			ON webhook_deliveries(expires_at);
		CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_next_retry 
			ON webhook_deliveries(next_retry_at) WHERE next_retry_at IS NOT NULL;
	`

	// Execute the main schema
	if _, err := db.Exec(ctx, schema); err != nil {
		return err
	}

	// Create trigger function and trigger separately to avoid syntax issues
	triggerSQL := `
		CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		$$ language 'plpgsql';
	`

	if _, err := db.Exec(ctx, triggerSQL); err != nil {
		return err
	}

	triggerCreateSQL := `
		DROP TRIGGER IF EXISTS update_webhook_registrations_updated_at ON webhook_registrations;
		CREATE TRIGGER update_webhook_registrations_updated_at BEFORE UPDATE
			ON webhook_registrations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
	`

	_, err := db.Exec(ctx, triggerCreateSQL)
	return err
}
