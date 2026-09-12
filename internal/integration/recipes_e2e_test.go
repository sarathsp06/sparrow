//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/sarathsp06/sparrow/satellites/recipes"
)

// startBodyCaptureTarget returns a test server that records request bodies
// and always returns 200.
func startBodyCaptureTarget(t *testing.T) (*httptest.Server, chan []byte) {
	t.Helper()
	bodies := make(chan []byte, 16)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		defer r.Body.Close() //nolint:errcheck
		bodies <- b
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, bodies
}

// TestE2E_SlackRecipeTransform applies the slack recipe's transform_template
// to a subscription and verifies the receiver gets destination-native Block
// Kit JSON, not the raw Sparrow envelope.
func TestE2E_SlackRecipeTransform(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "recipe-slack-test"
		eventName = "recipe.slack"
	)

	raw, err := os.ReadFile("../../satellites/recipes/slack.yaml")
	require.NoError(t, err, "read slack recipe")
	var recipe recipes.Recipe
	require.NoError(t, yaml.Unmarshal(raw, &recipe), "parse slack recipe")
	require.NotEmpty(t, recipe.Subscription.TransformTemplate)

	target, bodies := startBodyCaptureTarget(t)

	registerEventType(t, c, ctx, eventName)
	webhookID := registerWebhookPipeline(t, c, ctx, namespace, eventName, target.URL, 1)

	// Find the auto-created subscription and set the recipe's transform on it.
	var subs struct {
		Items []struct {
			SubscriptionID string `json:"subscription_id"`
		} `json:"items"`
	}
	_, err = c.get(ctx, "/v1/namespaces/"+namespace+"/subscriptions?webhook_id="+webhookID, &subs)
	require.NoError(t, err)
	require.Len(t, subs.Items, 1)

	enabled := true
	resp, err := c.do(ctx, http.MethodPatch, "/v1/namespaces/"+namespace+"/subscriptions/"+subs.Items[0].SubscriptionID, map[string]any{
		"transform_enabled":  enabled,
		"transform_template": recipe.Subscription.TransformTemplate,
	}, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Push an event with payload and labels.
	var pushed struct {
		EventID string `json:"event_id"`
	}
	resp, err = c.post(ctx, "/v1/namespaces/"+namespace+"/events?event="+eventName, map[string]any{
		"payload": map[string]any{"user_id": 42, "email": "ada@example.com"},
		"labels":  map[string]string{"env": "test"},
	}, &pushed)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotEmpty(t, pushed.EventID)

	// The receiver must get rendered Block Kit JSON, not the envelope.
	select {
	case body := <-bodies:
		require.True(t, json.Valid(body), "delivered body is not valid JSON:\n%s", body)
		var got map[string]any
		require.NoError(t, json.Unmarshal(body, &got))
		blocks, ok := got["blocks"].([]any)
		require.True(t, ok, "expected top-level \"blocks\" array, got:\n%s", body)
		assert.NotEmpty(t, blocks)
		// Envelope markers must be absent: transform replaced the body.
		assert.NotContains(t, got, "event_id")
		assert.NotContains(t, got, "payload")
	case <-time.After(60 * time.Second):
		t.Fatal("timed out waiting for transformed delivery")
	}
}
