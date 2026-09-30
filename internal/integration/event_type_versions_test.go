//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

type eventTypeResp struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"event_schema"`
	Active      bool           `json:"active"`
	Version     int            `json:"version"`
	Change      struct {
		Action          string   `json:"action"`
		Version         int      `json:"version"`
		PreviousVersion int      `json:"previous_version"`
		Changes         []string `json:"changes"`
	} `json:"change"`
}

type eventTypeVersionResp struct {
	Version         int            `json:"version"`
	Schema          map[string]any `json:"event_schema"`
	SchemaDefinedAt *string        `json:"schema_defined_at"`
}

func totalSchema(typ string) map[string]any {
	return map[string]any{
		"type":       "object",
		"required":   []string{"total"},
		"properties": map[string]any{"total": map[string]any{"type": typ}},
	}
}

func TestEventTypeVersions_Lifecycle(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()
	c := newRESTClient(t, env)
	const name = "order.created"

	// Register with no schema: v1.
	var created eventTypeResp
	resp, err := c.post(ctx, "/v1/event-types", map[string]any{"name": name}, &created)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, 1, created.Version)

	// Registering the same name again is a conflict, not an overwrite.
	resp, err = c.post(ctx, "/v1/event-types", map[string]any{"name": name}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	// Push under v1 before any schema exists.
	earlyID, _, _, _, err := env.webhookSvc.PushEvent(ctx, "shop", name, map[string]any{"total": 1}, 0, nil, nil, nil)
	require.NoError(t, err)

	// First schema fills in v1.
	var patched eventTypeResp
	resp, err = c.do(ctx, http.MethodPatch, "/v1/event-types/"+name, map[string]any{"event_schema": totalSchema("number")}, &patched)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "updated", patched.Change.Action)
	assert.Equal(t, 1, patched.Version)
	assert.Contains(t, patched.Change.Changes, "schema_defined")

	// Description change stays on v1.
	resp, err = c.do(ctx, http.MethodPatch, "/v1/event-types/"+name, map[string]any{"description": "An order"}, &patched)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, patched.Version)

	// Schema change creates v2.
	resp, err = c.do(ctx, http.MethodPatch, "/v1/event-types/"+name, map[string]any{"event_schema": totalSchema("string")}, &patched)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "new_version", patched.Change.Action)
	assert.Equal(t, 2, patched.Version)
	assert.Equal(t, 1, patched.Change.PreviousVersion)

	// History keeps both, newest first; v1 records when its schema was filled in.
	var versions struct {
		Items []eventTypeVersionResp `json:"items"`
	}
	resp, err = c.get(ctx, "/v1/event-types/"+name+"/versions", &versions)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, versions.Items, 2)
	assert.Equal(t, 2, versions.Items[0].Version)
	assert.Equal(t, 1, versions.Items[1].Version)
	assert.NotNil(t, versions.Items[1].SchemaDefinedAt)
	assert.Nil(t, versions.Items[0].SchemaDefinedAt)

	var v1 eventTypeVersionResp
	resp, err = c.get(ctx, "/v1/event-types/"+name+"/versions/1", &v1)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "number", v1.Schema["properties"].(map[string]any)["total"].(map[string]any)["type"])

	resp, err = c.get(ctx, "/v1/event-types/"+name+"/versions/9", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Events are pinned to the version they were accepted under.
	lateID, _, _, _, err := env.webhookSvc.PushEvent(ctx, "shop", name, map[string]any{"total": "9.50"}, 0, nil, nil, nil)
	require.NoError(t, err)
	var early, late struct {
		EventVersion int `json:"event_version"`
	}
	_, err = c.get(ctx, "/v1/events/"+earlyID, &early)
	require.NoError(t, err)
	_, err = c.get(ctx, "/v1/events/"+lateID, &late)
	require.NoError(t, err)
	assert.Equal(t, 1, early.EventVersion)
	assert.Equal(t, 2, late.EventVersion)
}

func TestEventTypeVersions_NoDelete(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()
	c := newRESTClient(t, env)

	_, err := c.post(ctx, "/v1/event-types", map[string]any{"name": "report.ready"}, nil)
	require.NoError(t, err)

	resp, err := c.do(ctx, http.MethodDelete, "/v1/event-types/report.ready", nil, nil)
	require.NoError(t, err)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed}, resp.StatusCode)

	// The database refuses too: a definition with history cannot be removed.
	_, err = env.sqlxDB.ExecContext(ctx, `DELETE FROM event_registrations WHERE tenant_id = $1 AND name = $2`, tenant.DefaultTenantID, "report.ready")
	require.Error(t, err, "the history foreign key must block deletion")

	// Deactivating is the way to retire a type; pushes are then refused.
	var patched eventTypeResp
	resp, err = c.do(ctx, http.MethodPatch, "/v1/event-types/report.ready", map[string]any{"active": false}, &patched)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.False(t, patched.Active)
	assert.Equal(t, 1, patched.Version)

	resp, err = c.post(ctx, "/v1/consumers/shop/events?event=report.ready", map[string]any{"payload": map[string]any{}}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestEventTypeVersions_ReservedPrefix(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()
	c := newRESTClient(t, env)

	for _, name := range []string{"sparrow.custom", "Sparrow.Custom"} {
		resp, err := c.post(ctx, "/v1/event-types", map[string]any{"name": name}, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "register %s", name)

		resp, err = c.post(ctx, "/v1/consumers/shop/events?event="+name, map[string]any{"payload": map[string]any{}}, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "push %s", name)
	}
}

func TestEventTypeVersions_ConcurrentSchemaChanges(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()
	c := newRESTClient(t, env)
	const name = "load.tested"

	_, err := c.post(ctx, "/v1/event-types", map[string]any{"name": name, "event_schema": totalSchema("number")}, nil)
	require.NoError(t, err)

	// Every writer changes the schema to something distinct, so each must
	// create exactly one new version and none may reuse a number.
	const writers = 8
	var wg sync.WaitGroup
	statuses := make([]int, writers)
	for i := range writers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			schema := totalSchema("number")
			schema["description"] = fmt.Sprintf("writer %d", i)
			resp, err := c.do(ctx, http.MethodPatch, "/v1/event-types/"+name, map[string]any{"event_schema": schema}, nil)
			if err == nil {
				statuses[i] = resp.StatusCode
			}
		}(i)
	}
	wg.Wait()
	for i, s := range statuses {
		assert.Equal(t, http.StatusOK, s, "writer %d", i)
	}

	repo := store.NewRepository(env.sqlxDB)
	versions, err := repo.ListEventTypeVersions(ctx, tenant.DefaultTenantID, name)
	require.NoError(t, err)
	require.Len(t, versions, writers+1)
	for i, v := range versions {
		assert.Equal(t, writers+1-i, v.Version, "versions are contiguous with no gaps or duplicates")
	}
	head, err := repo.GetEventByName(ctx, tenant.DefaultTenantID, name)
	require.NoError(t, err)
	assert.Equal(t, writers+1, head.Version)
}

func TestEventTypeVersions_AutoRegisterThenFillIn(t *testing.T) {
	env := setupEnv(t) // auto-register is on in the integration environment
	ctx := context.Background()
	c := newRESTClient(t, env)

	_, _, _, _, err := env.webhookSvc.PushEvent(ctx, "shop", "cart.abandoned", map[string]any{"total": 3}, 0, nil, nil, nil)
	require.NoError(t, err)

	var patched eventTypeResp
	resp, err := c.do(ctx, http.MethodPatch, "/v1/event-types/cart.abandoned", map[string]any{"event_schema": totalSchema("number")}, &patched)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, patched.Version, "the first real definition fills in the blank v1, so environments stay aligned")
}
