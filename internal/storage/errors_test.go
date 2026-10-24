package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestErrorTranslation(t *testing.T) {
	if Error(nil) != nil {
		t.Fatal("nil must stay nil")
	}
	if !IsNotFound(Error(fmt.Errorf("get: %w", sql.ErrNoRows))) {
		t.Fatal("sql.ErrNoRows must become ErrNotFound")
	}
	for code, want := range map[string]error{
		"23502": ErrNotNullViolation,
		"23505": ErrAlreadyExists,
		"23503": ErrForeignKeyViolation,
	} {
		pgErr := &pgconn.PgError{Code: code}
		got := Error(fmt.Errorf("insert: %w", pgErr))
		if !errors.Is(got, want) {
			t.Errorf("code %s: got %v, want %v", code, got, want)
		}
		var kept *pgconn.PgError
		if !errors.As(got, &kept) {
			t.Errorf("code %s: the Postgres error must stay reachable", code)
		}
	}
	other := &pgconn.PgError{Code: "42P01"}
	if got := Error(other); got != other {
		t.Errorf("an unmapped Postgres code must pass through, got %v", got)
	}
	plain := errors.New("boom")
	if Error(plain) != plain {
		t.Error("a non-SQL error must pass through")
	}
}
