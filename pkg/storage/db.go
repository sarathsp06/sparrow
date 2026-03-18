package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// DBTX is the subset of methods shared by *sqlx.DB and *sqlx.Tx.
// Repositories use this for all query/exec operations, which allows them
// to work transparently with either a raw connection or a transaction.
type DBTX interface {
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// DB defines an interface for a database connection
// with restrictive minimum methods for SQLX.
// DB is a superset of DBTX — it adds lifecycle and transaction methods.
type DB interface {
	DBTX
	// Ping checks if the database is alive
	Ping() error
	// Close closes the database connection
	Close() error
	// Beginx starts a new transaction
	Beginx() (*sqlx.Tx, error)
}

// WithTransaction runs fn inside a database transaction. If fn returns an
// error the transaction is rolled back; otherwise it is committed. The
// deferred Rollback is a no-op after a successful Commit.
//
// The callback receives a DBTX so that any repository's WithConn method can
// accept it, enabling cross-repository transactional operations:
//
//	storage.WithTransaction(db, func(tx storage.DBTX) error {
//	    webhookRepo.WithConn(tx).RegisterWebhook(ctx, ...)
//	    auditRepo.WithConn(tx).RecordAction(ctx, ...)
//	    return nil
//	})
func WithTransaction(db DB, fn func(tx DBTX) error) error {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
