package storage

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// DB defines an interface for a database connection
// with restricive minimum methods fro SQLX
type DB interface {
	// Ping checks if the database is alive
	Ping() error
	// Close closes the database connection
	Close() error
	// Begin starts a new transaction
	Beginx() (*sqlx.Tx, error)
	Reader
	Writer
}

// TX defines an interface for a database transaction
type TX interface {
	DB
	Commit() error
	Rollback() error
}

// Reader defines an interface for a database connection
// with read methods. This is used to define a read-only
// connection.
// This helps to prevent accidental writes to the database
// when a read-only connection is used and moving the read
// operations to a separate database for performance reasons like read replicas.
type Reader interface {
	// SelectContext gets a single row from the database
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	// SelectContext gets multiple rows from the database
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

// Writer defines an interface for a database connection
// with write methods. This is used to define a write-only
// connection.
// This helps with moving the write operations to a separate database
// for performance reasons.
type Writer interface {
	// NamedExecContext executes a query without returning any rows
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	// ExecContext executes a query without returning any rows
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// WithTransaction creates a new transaction and executes the given function
// inside the transaction
func WithTransaction(ctx context.Context, connection DB, fn func(ctx context.Context, tx *sqlx.Tx) error) error {
	tx, err := connection.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		// Rollback if the transaction is not committed
		tx.Rollback() // nolint:errcheck
	}()
	if err := fn(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}
