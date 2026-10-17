package postgres

import (
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"github.com/uptrace/opentelemetry-go-extra/otelsqlx"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Open opens a new database connection with the given dsn, retrying the
// first ping up to maxRetries times. Tune the pool with the *sqlx.DB setters.
func Open(dsn string, maxRetries int) (*sqlx.DB, error) {
	db, err := otelsqlx.Open(`pgx`, dsn, otelsql.WithAttributes(semconv.DBSystemPostgreSQL))
	if err != nil {
		return nil, err
	}
	// retry the connection if it fails
	for range maxRetries {
		if err = db.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	otelsql.ReportDBStatsMetrics(db.DB)
	return db, nil
}
