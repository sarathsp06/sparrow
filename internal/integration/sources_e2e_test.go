//go:build integration

package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/cmd/sparrow-sources/sources"
)

// startIngest mounts the sparrow-sources ingest handler on an httptest
// server pointed at the harness Sparrow.
func startIngest(t *testing.T, env *testEnv, namespace string, providers sources.ProvidersConfig) *httptest.Server {
	t.Helper()
	pusher := sources.NewClient(sources.SparrowConfig{URL: env.baseURL, Namespace: namespace})
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httptest.NewServer(sources.NewIngestHandler(sources.IngestConfig{Providers: providers}, pusher, log))
	t.Cleanup(srv.Close)
	return srv
}

// findEventOccurrence polls listEventOccurrences until an occurrence of the
// given event name appears, returning its event id.
func findEventOccurrence(t *testing.T, c *restClient, ctx context.Context, namespace, eventName string) string {
	t.Helper()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for event occurrence %s", eventName)
			return ""
		case <-ticker.C:
			var out struct {
				Items []struct {
					EventID string `json:"event_id"`
				} `json:"items"`
			}
			_, err := c.get(ctx, "/v1/namespaces/"+namespace+"/events?event="+eventName, &out)
			if err == nil && len(out.Items) > 0 {
				return out.Items[0].EventID
			}
		}
	}
}

// TestE2E_SourcesStripeIngest verifies the full path: signed Stripe webhook
// -> sparrow-sources ingest -> Sparrow event (auto-created event type) ->
// delivery to a subscribed webhook.
func TestE2E_SourcesStripeIngest(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "sources-stripe"
		secret    = "whsec_e2e_test"
		eventName = "stripe.payment_intent.succeeded"
	)

	targetSrv, requestCount := startCountingTarget(t)
	registerEventType(t, c, ctx, eventName)
	registerWebhookPipeline(t, c, ctx, namespace, eventName, targetSrv.URL, 3)

	ingest := startIngest(t, env, namespace, sources.ProvidersConfig{
		Stripe: &sources.StripeConfig{SigningSecret: secret, EventPrefix: "stripe"},
	})

	body := `{
		"id": "evt_1Nv8xY2eZvKYlo2C",
		"object": "event",
		"type": "payment_intent.succeeded",
		"livemode": false,
		"created": 1757600000,
		"data": {"object": {"id": "pi_3Nv8xX2eZvKYlo2C", "object": "payment_intent", "amount": 2000, "currency": "usd", "status": "succeeded"}}
	}`
	ts := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.%s", ts, body)
	sig := fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ingest.URL+"/ingest/stripe", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Stripe-Signature", sig)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close() //nolint:errcheck
	require.Equal(t, http.StatusOK, resp.StatusCode, "ingest should 2xx after successful push")

	pollCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	eventID := findEventOccurrence(t, c, pollCtx, namespace, eventName)
	pollDeliveryStatus(t, c, pollCtx, namespace, eventID, func(d deliveryItem) bool {
		return d.Status == "success"
	})
	require.GreaterOrEqual(t, int(requestCount.Load()), 1, "subscribed webhook should have been delivered to")
}

// TestE2E_SourcesGitHubIngest verifies the same path for a signed GitHub
// push webhook.
func TestE2E_SourcesGitHubIngest(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "sources-github"
		secret    = "gh_e2e_test"
		eventName = "github.push"
	)

	targetSrv, requestCount := startCountingTarget(t)
	registerEventType(t, c, ctx, eventName)
	registerWebhookPipeline(t, c, ctx, namespace, eventName, targetSrv.URL, 3)

	ingest := startIngest(t, env, namespace, sources.ProvidersConfig{
		GitHub: &sources.GitHubConfig{Secret: secret, EventPrefix: "github"},
	})

	body := `{
		"ref": "refs/heads/main",
		"before": "0000000000000000000000000000000000000000",
		"after": "84fb41b5f2a1c1a37c95a4a25b3f6d1c2d3e4f5a",
		"repository": {"full_name": "acme/site", "default_branch": "main"},
		"pusher": {"name": "octocat"}
	}`
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ingest.URL+"/ingest/github", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "push")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close() //nolint:errcheck
	require.Equal(t, http.StatusOK, resp.StatusCode, "ingest should 2xx after successful push")

	pollCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	eventID := findEventOccurrence(t, c, pollCtx, namespace, eventName)
	pollDeliveryStatus(t, c, pollCtx, namespace, eventID, func(d deliveryItem) bool {
		return d.Status == "success"
	})
	require.GreaterOrEqual(t, int(requestCount.Load()), 1, "subscribed webhook should have been delivered to")
}
