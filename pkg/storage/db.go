package storage

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// DB defines an interface for a database connection
// with restrictive minimum methods for SQLX
type DB interface {
	// Ping checks if the database is alive
	Ping() error
	// Close closes the database connection
	Close() error
	// Beginx starts a new transaction
	Beginx() (*sqlx.Tx, error)
	// GetContext gets a single row from the database
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	// SelectContext gets multiple rows from the database
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	// NamedExecContext executes a query without returning any rows
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	// ExecContext executes a query without returning any rows
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
