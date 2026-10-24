package postgres

import (
	"os"
	"testing"
)

func TestOpenUnreachable(t *testing.T) {
	db, err := Open("postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1", 1)
	if err == nil {
		_ = db.Close()
		t.Fatal("Open must fail when the database never answers")
	}
}

func TestOpen(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := Open(dsn, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck
	var one int
	if err := db.Get(&one, `SELECT 1`); err != nil || one != 1 {
		t.Fatalf("SELECT 1 = %d, %v", one, err)
	}
}
