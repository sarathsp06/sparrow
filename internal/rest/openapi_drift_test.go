package rest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/rest"
)

// TestOpenAPISpecMatchesCommitted regenerates the OpenAPI document from the
// Huma-registered operations and asserts it is byte-identical to the
// committed api/openapi.yaml. Huma code is the canonical contract (per the
// approved design); this test is the drift guard that keeps the committed
// spec — and everything generated from it (clients, e2e) — honest.
func TestOpenAPISpecMatchesCommitted(t *testing.T) {
	api := rest.Mount(chi.NewRouter(), nil)

	got, err := api.OpenAPI().YAML()
	require.NoError(t, err)

	committedPath := filepath.Join("..", "..", "api", "openapi.yaml")
	want, err := os.ReadFile(committedPath)
	require.NoError(t, err, "api/openapi.yaml must exist — run `make generate` (or `go run ./cmd/openapi-export api`) after changing internal/rest")

	require.Equal(t, string(want), string(got),
		"api/openapi.yaml is stale — regenerate with `go run ./cmd/openapi-export api` and commit the result")
}
