//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/server"
)

const zeroKeyring = "main=0000000000000000000000000000000000000000000000000000000000000000"

// TestServerErrorsInsteadOfExiting checks that New and Migrate report bad
// configuration and an unreachable database as errors. The sparrow-server
// binary used to log.Fatalf here; an embedding program must get an error it
// can handle instead of a dead process.
func TestServerErrorsInsteadOfExiting(t *testing.T) {
	ctx := context.Background()

	t.Run("missing encryption keyring", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://nobody@127.0.0.1:1/none?sslmode=disable")
		t.Setenv("SPARROW_ENCRYPTION_KEYS", "")
		t.Setenv("SPARROW_ENCRYPTION_PRIMARY_KEY_ID", "")
		_, err := server.New(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "configuration")
	})

	t.Run("invalid port", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://nobody@127.0.0.1:1/none?sslmode=disable")
		t.Setenv("SPARROW_ENCRYPTION_KEYS", zeroKeyring)
		t.Setenv("SPARROW_ENCRYPTION_PRIMARY_KEY_ID", "main")
		t.Setenv("SPARROW_HTTP_PORT", "not-a-port")
		_, err := server.New(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "SPARROW_HTTP_PORT")
	})

	t.Run("unreachable database", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=2")
		t.Setenv("SPARROW_ENCRYPTION_KEYS", zeroKeyring)
		t.Setenv("SPARROW_ENCRYPTION_PRIMARY_KEY_ID", "main")
		t.Setenv("SPARROW_METRICS_ENABLED", "false")
		_, err := server.New(ctx)
		require.Error(t, err, "New must return an error, not exit")
		require.Contains(t, err.Error(), "migrations", "New should fail at the database step")
		err = server.Migrate(ctx)
		require.Error(t, err, "Migrate must return an error, not exit")
		require.Contains(t, err.Error(), "migrations")
	})
}

// TestServerLifecycle runs Migrate, then New with the binary's defaults
// (TLS revocation on, so Start runs the CRL listener), and checks Addr,
// ShutdownTimeout, the CRL endpoint, CreateTenant's argument check, and a
// clean Shutdown.
func TestServerLifecycle(t *testing.T) {
	ctx := context.Background()
	databaseURL, _ := setupTestDB(t, ctx)
	t.Setenv("DATABASE_URL", databaseURL)
	t.Setenv("SPARROW_ENCRYPTION_KEYS", zeroKeyring)
	t.Setenv("SPARROW_ENCRYPTION_PRIMARY_KEY_ID", "main")
	t.Setenv("SPARROW_METRICS_ENABLED", "false")
	t.Setenv("SPARROW_HTTP_PORT", "9187")
	t.Setenv("SPARROW_SHUTDOWN_TIMEOUT", "7s")

	// Migrate alone creates the schema, as a pre-deploy job does.
	require.NoError(t, server.Migrate(ctx))
	db, err := sqlx.Connect("pgx", databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	var tables int
	require.NoError(t, db.Get(&tables, `SELECT count(*) FROM information_schema.tables WHERE table_name IN ('webhook_registrations', 'river_job')`))
	require.Equal(t, 2, tables)

	srv, err := server.New(ctx)
	require.NoError(t, err)
	require.Equal(t, ":9187", srv.Addr())
	require.Equal(t, 7*time.Second, srv.ShutdownTimeout())
	require.NoError(t, srv.Start(ctx))

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	// TLS revocation is on by default: its status endpoint answers.
	resp, err := http.Get(ts.URL + "/v1/tls/crls")
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	require.Error(t, srv.CreateTenant(ctx, uuid.Nil, "nameless"), "a tenant needs an id")

	ts.Close()
	shutdownCtx, cancel := context.WithTimeout(ctx, srv.ShutdownTimeout())
	defer cancel()
	require.NoError(t, srv.Shutdown(shutdownCtx))
}
