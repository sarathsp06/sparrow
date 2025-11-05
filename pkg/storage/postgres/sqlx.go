package postgres

import (
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"github.com/uptrace/opentelemetry-go-extra/otelsqlx"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// SQLXDB is a wrapper around sqlx.DB
type SQLXDB struct {
	*sqlx.DB
}

// Open opens a new database connection with the given dsn
// It applies options after opening the connection
func Open(dsn string, maxRetries int, options ...OpenConnectionOption) (*SQLXDB, error) {
	db, err := otelsqlx.Open(`pgx`, dsn, otelsql.WithAttributes(semconv.DBSystemPostgreSQL))
	if err != nil {
		return nil, err
	}
	// retry the connection if it fails
	for i := 1; i < maxRetries; i++ {
		if err := db.Ping(); err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	for _, option := range options {
		if err := option(db); err != nil {
			return nil, err
		}
	}
	return &SQLXDB{db}, nil
}

type OpenConnectionOption func(*sqlx.DB) error

func WithConnectionMaxLifeTime(d time.Duration) OpenConnectionOption {
	return func(db *sqlx.DB) error {
		// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
		// Expired connections may be closed lazily before reuse.
		// If d <= 0, connections are reused forever.
		// The default is 0 (forever).
		db.SetConnMaxLifetime(d)
		return nil
	}
}

func WithMaxOpenConnections(n int) OpenConnectionOption {
	return func(db *sqlx.DB) error {
		// configure the database connection with some sane defaults
		// SetMaxOpenConns sets the maximum number of open connections to the database.
		// If n <= 0, there is no limit on the number of open connections.
		// The default is 0 (unlimited).
		db.SetMaxOpenConns(n)
		return nil
	}
}

func WithMaxIdleConnections(n int) OpenConnectionOption {
	return func(db *sqlx.DB) error {
		// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
		// If MaxIdleConns is greater than the configured MaxOpenConns, then the new MaxIdleConns will be reduced to match the MaxOpenConns limit.
		// If n <= 0, no idle connections are retained.
		// The default is 2.
		db.SetMaxIdleConns(n)
		return nil
	}
}

func WithSetConnMaxIdleTime(d time.Duration) OpenConnectionOption {
	return func(db *sqlx.DB) error {
		// SetConnMaxIdleTime sets the maximum amount of time a connection may be idle before being closed.
		// Expired connections may be closed lazily before reuse.
		// If d <= 0, connections are reused forever.
		// The default is 0 (unlimited).
		db.SetConnMaxIdleTime(d)
		return nil
	}
}
