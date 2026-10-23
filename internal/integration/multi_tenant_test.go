//go:build integration

package integration

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/server"
)

// tenantHeader is how these tests pick a tenant. A real embedder decides the
// tenant from its own authentication; the mechanism into Sparrow is the same
// server.WithTenant call.
const tenantHeader = "X-Test-Tenant"

// embeddedSparrow is Sparrow run through the public server package with a
// test middleware that maps tenantHeader to server.WithTenant.
type embeddedSparrow struct {
	srv     *server.Server
	baseURL string
	db      *sqlx.DB
}

func startEmbeddedSparrow(t *testing.T) *embeddedSparrow {
	t.Helper()
	ctx := context.Background()
	databaseURL, _ := setupTestDB(t, ctx)

	t.Setenv("DATABASE_URL", databaseURL)
	t.Setenv("SPARROW_ENCRYPTION_KEYS", "main=0000000000000000000000000000000000000000000000000000000000000000")
	t.Setenv("SPARROW_ENCRYPTION_PRIMARY_KEY_ID", "main")
	t.Setenv("SPARROW_METRICS_ENABLED", "false")
	t.Setenv("SPARROW_TLS_REVOCATION", "off")
	t.Setenv("SPARROW_ALLOW_PRIVATE_NETWORKS", "true")
	t.Setenv("SPARROW_HEALTH_EVAL_INTERVAL", "1s")

	srv, err := server.New(ctx)
	require.NoError(t, err)
	require.NoError(t, srv.Start(ctx))

	withTenant := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if h := r.Header.Get(tenantHeader); h != "" {
				r = r.WithContext(server.WithTenant(r.Context(), uuid.MustParse(h)))
			}
			next.ServeHTTP(w, r)
		})
	}
	ts := httptest.NewServer(withTenant(srv.Handler()))

	db, err := sqlx.Connect("pgx", databaseURL)
	require.NoError(t, err)

	t.Cleanup(func() {
		ts.Close()
		_ = db.Close()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		require.NoError(t, srv.Shutdown(shutdownCtx))
	})
	return &embeddedSparrow{srv: srv, baseURL: ts.URL, db: db}
}

// addTenant creates a tenant through the public package.
func (e *embeddedSparrow) addTenant(t *testing.T, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, e.srv.CreateTenant(context.Background(), id, name))
	return id
}

// client returns a REST client acting as tenantID; uuid.Nil sends no tenant,
// as a self-hosted request does.
func (e *embeddedSparrow) client(t *testing.T, tenantID uuid.UUID) *restClient {
	c := &restClient{t: t, baseURL: e.baseURL, http: &http.Client{Timeout: 15 * time.Second}}
	if tenantID != uuid.Nil {
		c.http.Transport = headerTransport{name: tenantHeader, value: tenantID.String()}
	}
	return c
}

type headerTransport struct{ name, value string }

func (h headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set(h.name, h.value)
	return http.DefaultTransport.RoundTrip(r)
}

// countingReceiver counts webhook deliveries it accepts.
type countingReceiver struct {
	srv   *httptest.Server
	count atomic.Int32
}

func newCountingReceiver(t *testing.T) *countingReceiver {
	rcv := &countingReceiver{}
	rcv.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		rcv.count.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(rcv.srv.Close)
	return rcv
}

func (r *countingReceiver) waitFor(t *testing.T, n int32) {
	t.Helper()
	require.Eventually(t, func() bool { return r.count.Load() >= n }, 30*time.Second, 100*time.Millisecond,
		"receiver got %d deliveries, want %d", r.count.Load(), n)
}

// TestMultiTenantIsolation runs two tenants through one embedded Sparrow
// with identical consumer and event names, and checks that each tenant's
// webhooks, events and deliveries stay invisible and untouchable to the
// other. The first tenant is the default one (no tenant set), which is what
// every self-hosted request uses.
func TestMultiTenantIsolation(t *testing.T) {
	ctx := context.Background()
	sp := startEmbeddedSparrow(t)
	tenantB := sp.addTenant(t, "tenant-b")

	a := sp.client(t, uuid.Nil) // default tenant, as self-hosted
	b := sp.client(t, tenantB)

	const consumer = "acme"
	const event = "order.created"

	type setup struct {
		webhookID string
		receiver  *countingReceiver
	}
	register := func(c *restClient) setup {
		resp, err := c.post(ctx, "/v1/event-types", map[string]any{"name": event, "active": true}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode, "each tenant has its own event type namespace")

		rcv := newCountingReceiver(t)
		var out struct {
			WebhookID string `json:"webhook_id"`
		}
		resp, err = c.post(ctx, "/v1/consumers/"+consumer+"/webhooks", map[string]any{
			"events": []string{event}, "url": rcv.srv.URL + "/hook", "active": true,
		}, &out)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NotEmpty(t, out.WebhookID)
		return setup{webhookID: out.WebhookID, receiver: rcv}
	}
	sa, sb := register(a), register(b)

	push := func(c *restClient) string {
		var out struct {
			EventID string `json:"event_id"`
		}
		resp, err := c.post(ctx, "/v1/consumers/"+consumer+"/events?event="+event,
			map[string]any{"payload": map[string]any{"order_id": "1"}}, &out)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		return out.EventID
	}

	// An event pushed in one tenant reaches only that tenant's webhook.
	eventA := push(a)
	sa.receiver.waitFor(t, 1)
	eventB := push(b)
	sb.receiver.waitFor(t, 1)
	time.Sleep(time.Second) // give any wrong fan-out time to land
	require.EqualValues(t, 1, sa.receiver.count.Load(), "tenant A's receiver got tenant B's event")
	require.EqualValues(t, 1, sb.receiver.count.Load(), "tenant B's receiver got tenant A's event")

	// Lists show only the caller's own webhooks.
	type listWebhooks struct {
		Items []struct {
			WebhookID string `json:"webhook_id"`
		} `json:"items"`
	}
	for _, tc := range []struct {
		c    *restClient
		want string
	}{{a, sa.webhookID}, {b, sb.webhookID}} {
		var out listWebhooks
		resp, err := tc.c.get(ctx, "/v1/consumers/"+consumer+"/webhooks", &out)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Len(t, out.Items, 1)
		require.Equal(t, tc.want, out.Items[0].WebhookID)
	}

	// The other tenant's webhook and event are not found.
	resp, err := a.get(ctx, "/v1/consumers/"+consumer+"/webhooks/"+sb.webhookID, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "A can read B's webhook")
	resp, err = b.get(ctx, "/v1/consumers/"+consumer+"/webhooks/"+sa.webhookID, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "B can read A's webhook")

	// Deliveries: each tenant sees only its own, and cannot read or retry
	// the other's.
	type listDeliveries struct {
		Items []struct {
			DeliveryID string `json:"delivery_id"`
			EventID    string `json:"event_id"`
			Status     string `json:"status"`
		} `json:"items"`
	}
	deliveriesOf := func(c *restClient) listDeliveries {
		var out listDeliveries
		resp, err := c.get(ctx, "/v1/consumers/"+consumer+"/deliveries", &out)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		return out
	}
	var deliveryB string
	require.Eventually(t, func() bool {
		da, db := deliveriesOf(a), deliveriesOf(b)
		if len(da.Items) != 1 || len(db.Items) != 1 {
			return false
		}
		require.Equal(t, eventA, da.Items[0].EventID)
		require.Equal(t, eventB, db.Items[0].EventID)
		deliveryB = db.Items[0].DeliveryID
		return da.Items[0].Status == "success" && db.Items[0].Status == "success"
	}, 30*time.Second, 200*time.Millisecond, "each tenant should see exactly its own successful delivery")

	resp, err = a.get(ctx, "/v1/consumers/"+consumer+"/deliveries/"+deliveryB, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "A can read B's delivery")
	resp, err = a.post(ctx, "/v1/consumers/"+consumer+"/deliveries/"+deliveryB+":retry", nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "A can retry B's delivery")
	time.Sleep(time.Second)
	require.EqualValues(t, 1, sb.receiver.count.Load(), "a cross-tenant retry reached B's receiver")

	// Event history and event types are per tenant.
	resp, err = a.get(ctx, "/v1/events/"+eventB, nil)
	require.NoError(t, err)
	require.NotEqual(t, http.StatusOK, resp.StatusCode, "A can read B's event")

	// Deleting in one tenant leaves the other untouched.
	resp, err = a.do(ctx, http.MethodDelete, "/v1/consumers/"+consumer+"/webhooks/"+sb.webhookID, nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "A can delete B's webhook")
	var stillThere listWebhooks
	_, err = b.get(ctx, "/v1/consumers/"+consumer+"/webhooks", &stillThere)
	require.NoError(t, err)
	require.Len(t, stillThere.Items, 1, "B's webhook survived A's delete attempt")
}
