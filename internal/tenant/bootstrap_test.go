package tenant

import (
	"context"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func TestBootstrapReportsAnUnreachableDatabase(t *testing.T) {
	db, err := sqlx.Open("pgx", "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck
	err = Bootstrap(context.Background(), db)
	if err == nil || !strings.Contains(err.Error(), "check default tenant") {
		t.Fatalf("Bootstrap = %v, want a check default tenant error", err)
	}
}
