//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// restClient is a minimal JSON REST client for the integration suite,
// replacing the old Connect-RPC generated clients now that Sparrow's
// interface is REST/OpenAPI only.
type restClient struct {
	t       *testing.T
	baseURL string
	apiKey  string
	http    *http.Client
}

func newRESTClient(t *testing.T, env *testEnv) *restClient {
	return &restClient{t: t, baseURL: env.baseURL, http: http.DefaultClient}
}

func (c *restClient) do(ctx context.Context, method, path string, body any, out any) (*http.Response, error) {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(c.t, err)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	require.NoError(c.t, err)
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	if out != nil && resp.StatusCode < 300 {
		if decErr := json.NewDecoder(resp.Body).Decode(out); decErr != nil && decErr != io.EOF {
			return resp, fmt.Errorf("decode response: %w", decErr)
		}
	}
	return resp, nil
}

func (c *restClient) post(ctx context.Context, path string, body, out any) (*http.Response, error) {
	return c.do(ctx, http.MethodPost, path, body, out)
}

func (c *restClient) get(ctx context.Context, path string, out any) (*http.Response, error) {
	return c.do(ctx, http.MethodGet, path, nil, out)
}
