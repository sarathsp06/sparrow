package storage

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound            = errors.New("storage: not found")
	ErrForeignKeyViolation = errors.New("storage: foreign key violation")
	ErrAlreadyExists       = errors.New("storage: already exists")
	ErrNotNullViolation    = errors.New("storage: not null violation")
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// Error checks the error with SQL error codes and returns a storage error
// if it is nill it returns nil
func Error(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	pgxErr := &pgconn.PgError{}
	if errors.As(err, &pgxErr) {
		switch pgxErr.Code {
		case "23502":
			return errors.Join(ErrNotNullViolation, pgxErr)
		case "23505":
			return errors.Join(ErrAlreadyExists, pgxErr)
		case "23503":
			return errors.Join(ErrForeignKeyViolation, pgxErr)
		}
	}
	return err
}
