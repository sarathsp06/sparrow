//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bundle struct {
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Stamp      map[string]any   `json:"stamp"`
	Items      []map[string]any `json:"items"`
}

type importResult struct {
	Applied    bool   `json:"applied"`
	DryRun     bool   `json:"dry_run"`
	ImportedAt string `json:"imported_at"`
	Stamp      struct {
		Status   string   `json:"status"`
		Warnings []string `json:"warnings"`
	} `json:"stamp"`
	BlockedBy []string `json:"blocked_by"`
	Items     []struct {
		Name          string   `json:"name"`
		Action        string   `json:"action"`
		Version       int      `json:"version"`
		Changes       []string `json:"changes"`
		Blocked       bool     `json:"blocked"`
		Compatibility *struct {
			Result  string   `json:"result"`
			Reasons []string `json:"reasons"`
		} `json:"compatibility"`
		Subscriptions *struct {
			Failures []struct {
				SubscriptionID string `json:"subscription_id"`
				Payload        string `json:"payload"`
				Error          string `json:"error"`
			} `json:"failures"`
			Passed           int `json:"passed"`
			WithoutTransform int `json:"without_transform"`
		} `json:"subscriptions"`
	} `json:"items"`
}

func exportBundle(t *testing.T, c *restClient, ctx context.Context, sel map[string]any) bundle {
	t.Helper()
	var b bundle
	resp, err := c.post(ctx, "/v1/event-types:export", sel, &b)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	return b
}

// importBundle posts an exported bundle as-is, plus options.
func importBundle(t *testing.T, c *restClient, ctx context.Context, b bundle, opts map[string]any) (int, importResult) {
	t.Helper()
	body := map[string]any{"items": b.Items}
	if b.APIVersion != "" {
		body["apiVersion"], body["kind"] = b.APIVersion, b.Kind
	}
	if b.Stamp != nil {
		body["stamp"] = b.Stamp
	}
	for k, v := range opts {
		body[k] = v
	}
	var out importResult
	resp, err := c.post(ctx, "/v1/event-types:import", body, &out)
	require.NoError(t, err)
	return resp.StatusCode, out
}

func TestEventTypeBundle_RoundTripBetweenEnvironments(t *testing.T) {
	ctx := context.Background()
	dev, prod := setupEnv(t), setupEnv(t)
	devC, prodC := newRESTClient(t, dev), newRESTClient(t, prod)

	for _, et := range []map[string]any{
		{"name": "order.created", "description": "An order", "event_schema": totalSchema("number"), "metadata": map[string]string{"owner": "payments"}},
		{"name": "order.shipped", "description": "Shipped"},
		{"name": "user.signup"},
	} {
		_, err := devC.post(ctx, "/v1/event-types", et, nil)
		require.NoError(t, err)
	}

	// Select by prefix: two of the three, sorted by name; system events never included.
	b := exportBundle(t, devC, ctx, map[string]any{"prefix": "order."})
	assert.Equal(t, "sparrow/v1", b.APIVersion)
	assert.Equal(t, "EventTypeList", b.Kind)
	require.Len(t, b.Items, 2)
	assert.Equal(t, "order.created", b.Items[0]["name"])
	assert.Equal(t, "order.shipped", b.Items[1]["name"])
	for _, it := range b.Items {
		assert.NotContains(t, it, "version", "version numbers are per environment and stay out of the file")
		assert.NotContains(t, it, "sample_payload")
	}
	all := exportBundle(t, devC, ctx, map[string]any{"all": true})
	for _, it := range all.Items {
		assert.NotContains(t, it["name"], "sparrow.")
	}

	// Dry run into prod: full preview, nothing written.
	status, res := importBundle(t, prodC, ctx, b, map[string]any{"dry_run": true})
	require.Equal(t, http.StatusOK, status)
	assert.False(t, res.Applied)
	assert.Equal(t, "valid", res.Stamp.Status)
	require.Len(t, res.Items, 2)
	assert.Equal(t, "created", res.Items[0].Action)
	resp, err := prodC.get(ctx, "/v1/event-types/order.created", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "a dry run writes nothing")

	// Real import.
	status, res = importBundle(t, prodC, ctx, b, nil)
	require.Equal(t, http.StatusOK, status)
	assert.True(t, res.Applied)
	assert.NotEmpty(t, res.ImportedAt)

	// Prod now exports byte-for-byte the same items as dev did.
	back := exportBundle(t, prodC, ctx, map[string]any{"prefix": "order."})
	assert.Equal(t, b.Items, back.Items)
	assert.Equal(t, b.Stamp["sha256"], back.Stamp["sha256"])

	// Importing again changes nothing.
	_, res = importBundle(t, prodC, ctx, b, nil)
	for _, it := range res.Items {
		assert.Equal(t, "unchanged", it.Action, it.Name)
	}
}

func TestEventTypeBundle_StampWarningsNeedAcknowledgement(t *testing.T) {
	ctx := context.Background()
	env := setupEnv(t)
	c := newRESTClient(t, env)
	_, err := c.post(ctx, "/v1/event-types", map[string]any{"name": "report.ready"}, nil)
	require.NoError(t, err)
	b := exportBundle(t, c, ctx, map[string]any{"names": []string{"report.ready"}})

	// Edit the file after export: the digest no longer matches.
	b.Items[0]["description"] = "edited by hand"
	_, res := importBundle(t, c, ctx, b, nil)
	assert.False(t, res.Applied)
	assert.Equal(t, []string{"items_changed"}, res.Stamp.Warnings)
	assert.Equal(t, []string{"items_changed"}, res.BlockedBy)

	// A different exporting version, too.
	b.Stamp["sparrow_version"] = "0.0.1"
	_, res = importBundle(t, c, ctx, b, map[string]any{"acknowledge": []string{"items_changed"}})
	assert.False(t, res.Applied)
	assert.Equal(t, []string{"version_differs"}, res.BlockedBy)

	_, res = importBundle(t, c, ctx, b, map[string]any{"acknowledge": []string{"items_changed", "version_differs"}})
	assert.True(t, res.Applied)
	assert.Equal(t, "updated", res.Items[0].Action)

	// A hand-written file has no stamp and imports with only a notice.
	hand := bundle{APIVersion: "sparrow/v1", Kind: "EventTypeList", Items: []map[string]any{{"name": "report.failed"}}}
	_, res = importBundle(t, c, ctx, hand, nil)
	assert.Equal(t, "unsigned", res.Stamp.Status)
	assert.True(t, res.Applied)
}

func TestEventTypeBundle_AllOrNothingAndBreakingChanges(t *testing.T) {
	ctx := context.Background()
	env := setupEnv(t)
	c := newRESTClient(t, env)

	_, err := c.post(ctx, "/v1/event-types", map[string]any{"name": "invoice.paid", "event_schema": totalSchema("number")}, nil)
	require.NoError(t, err)
	srv, _ := startBodyRecorder(t)
	_, subID := setupTemplateSubscription(t, c, ctx, "billing", "invoice.paid", srv.URL, `{"amount": {{.payload.total}}}`, nil)

	// One new type and one breaking change to a subscribed type.
	breaking := map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}}
	b := bundle{APIVersion: "sparrow/v1", Kind: "EventTypeList", Items: []map[string]any{
		{"name": "invoice.paid", "event_schema": breaking},
		{"name": "invoice.voided"},
	}}

	_, res := importBundle(t, c, ctx, b, nil)
	assert.False(t, res.Applied)
	assert.Equal(t, []string{"breaking"}, res.BlockedBy)
	require.Len(t, res.Items, 2)
	paid := res.Items[0]
	assert.Equal(t, "invoice.paid", paid.Name)
	assert.True(t, paid.Blocked)
	require.NotNil(t, paid.Compatibility)
	assert.Equal(t, "breaking", paid.Compatibility.Result)
	assert.Contains(t, paid.Compatibility.Reasons, "total: removed required property")
	require.NotNil(t, paid.Subscriptions)
	require.Len(t, paid.Subscriptions.Failures, 1, "the pre-check renders the subscription's template against the new version")
	assert.Equal(t, subID, paid.Subscriptions.Failures[0].SubscriptionID)
	assert.Contains(t, paid.Subscriptions.Failures[0].Error, "total")

	// All or nothing: the other item was not written either.
	resp, err := c.get(ctx, "/v1/event-types/invoice.voided", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	_, res = importBundle(t, c, ctx, b, map[string]any{"allow_breaking": true})
	assert.True(t, res.Applied)
	assert.Equal(t, 2, res.Items[0].Version)
	resp, err = c.get(ctx, "/v1/event-types/invoice.voided", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestEventTypeBundle_InvalidInput(t *testing.T) {
	ctx := context.Background()
	env := setupEnv(t)
	c := newRESTClient(t, env)

	// An unknown name fails the whole export.
	resp, err := c.post(ctx, "/v1/event-types:export", map[string]any{"names": []string{"does.not.exist"}}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp, err = c.post(ctx, "/v1/event-types:export", map[string]any{"all": true, "prefix": "x"}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	for name, items := range map[string][]map[string]any{
		"duplicate names": {{"name": "a.b"}, {"name": "a.b"}},
		"reserved prefix": {{"name": "sparrow.custom"}},
		"bad schema":      {{"name": "a.c", "event_schema": map[string]any{"type": 12}}},
	} {
		status, _ := importBundle(t, c, ctx, bundle{Items: items}, nil)
		assert.Equal(t, http.StatusBadRequest, status, name)
	}
	resp, err = c.get(ctx, "/v1/event-types/a.b", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "an invalid bundle imports nothing")
}
