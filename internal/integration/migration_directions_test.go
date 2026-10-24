//go:build integration

package integration

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/migration"
)

// TestAppMigrationDirections drives RunAppMigrations the way the -migrate
// flag of sparrow-server can: up and down by steps and to a target version,
// recovery from a dirty version left by a failed migration, and a bad
// direction.
func TestAppMigrationDirections(t *testing.T) {
	ctx := context.Background()
	databaseURL, _ := setupTestDB(t, ctx)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	db, err := sqlx.Connect("pgx", databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	state := func() (version uint, dirty bool) {
		t.Helper()
		var row struct {
			Version uint `db:"version"`
			Dirty   bool `db:"dirty"`
		}
		require.NoError(t, db.Get(&row, `SELECT version, dirty FROM schema_migrations`))
		return row.Version, row.Dirty
	}
	run := func(direction string, steps int, target uint) {
		t.Helper()
		require.NoError(t, migration.RunAllMigrations(ctx, databaseURL, direction, steps, target, log))
	}

	run("up", 0, 0)
	latest, dirty := state()
	require.False(t, dirty)
	require.Greater(t, latest, uint(3))

	run("up", 0, 0) // nothing to do is not an error
	v, _ := state()
	require.Equal(t, latest, v)

	run("down", 0, 0) // one step by default
	v, _ = state()
	require.Equal(t, latest-1, v)

	run("down", 2, 0)
	v, _ = state()
	require.Equal(t, latest-3, v)

	run("up", 1, 0)
	v, _ = state()
	require.Equal(t, latest-2, v)

	run("up", 0, latest)
	v, _ = state()
	require.Equal(t, latest, v)

	run("down", 0, latest-1)
	v, _ = state()
	require.Equal(t, latest-1, v)

	// A migration that failed half-way leaves its version dirty. The next run
	// steps back one version and applies it again.
	_, err = db.Exec(`UPDATE schema_migrations SET version = $1, dirty = true`, latest)
	require.NoError(t, err)
	run("up", 0, 0)
	v, dirty = state()
	require.Equal(t, latest, v)
	require.False(t, dirty)

	err = migration.RunAllMigrations(ctx, databaseURL, "sideways", 0, 0, log)
	require.ErrorContains(t, err, "invalid direction")

	err = migration.RunAppMigrations(ctx, databaseURL, "up", 0, latest+100, log)
	require.Error(t, err, "a target version with no migration file is an error")
	v, dirty = state()
	require.Equal(t, latest, v)
	require.False(t, dirty)

	err = migration.RunAppMigrations(ctx, "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1", "up", 0, 0, log)
	require.Error(t, err)
}
