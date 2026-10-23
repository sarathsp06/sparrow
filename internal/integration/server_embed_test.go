//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/server"
)

// TestServerEmbedding runs Sparrow the way an embedding program does: through
// the public server package only, serving its handler behind the caller's own
// listener.
func TestServerEmbedding(t *testing.T) {
	ctx := context.Background()
	databaseURL, _ := setupTestDB(t, ctx)

	t.Setenv("DATABASE_URL", databaseURL)
	t.Setenv("SPARROW_ENCRYPTION_KEYS", "main=0000000000000000000000000000000000000000000000000000000000000000")
	t.Setenv("SPARROW_ENCRYPTION_PRIMARY_KEY_ID", "main")
	t.Setenv("SPARROW_METRICS_ENABLED", "false")
	t.Setenv("SPARROW_TLS_REVOCATION", "off")

	srv, err := server.New(ctx)
	require.NoError(t, err)
	require.NoError(t, srv.Start(ctx))

	ts := httptest.NewServer(srv.Handler())

	for _, path := range []string{"/health", "/ready", "/v1/capabilities"} {
		resp, err := http.Get(ts.URL + path)
		require.NoError(t, err, path)
		_ = resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, path)
	}

	srv.Drain()
	resp, err := http.Get(ts.URL + "/ready")
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "/ready while draining")

	ts.Close()
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	require.NoError(t, srv.Shutdown(shutdownCtx))
}
