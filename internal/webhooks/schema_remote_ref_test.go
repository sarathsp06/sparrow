package webhooks

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A schema must never make the server fetch a remote $ref: it would bypass
// the delivery network policy (SSRF) and read the body without a limit.
func TestSchemaCompile_RemoteRefIsNotFetched(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`{"type":"object"}`))
	}))
	defer srv.Close()

	schema := map[string]any{"$ref": srv.URL + "/schema.json"}

	require.Error(t, compileSchema(schema))
	require.Error(t, ValidateJSONSchema(schema, map[string]any{"a": 1}))
	assert.Zero(t, hits.Load(), "remote $ref was fetched")
}
