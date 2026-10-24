package migration

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
)

const unreachable = "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1"

// TestMigrationsReportConnectionProblems checks that a bad or unreachable
// database is an error naming the step that failed, never a hang or a panic.
func TestMigrationsReportConnectionProblems(t *testing.T) {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	for _, tc := range []struct {
		name string
		run  func() error
		want string
	}{
		{"river, malformed url", func() error { return RunRiverMigrations(ctx, "postgres://host:notaport/db", log) }, "database pool"},
		{"river, unreachable", func() error { return RunRiverMigrations(ctx, unreachable, log) }, "connect to database"},
		{"all, unreachable", func() error { return RunAllMigrations(ctx, unreachable, "up", 0, 0, log) }, "River migrations"},
		{"app, unreachable", func() error { return RunAppMigrations(ctx, unreachable, "up", 0, 0, log) }, "connection"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want an error containing %q", err, tc.want)
			}
		})
	}
}
