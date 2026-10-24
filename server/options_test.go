package server

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/tenant"
)

func TestWithTenantReachesTheServiceLayer(t *testing.T) {
	if got := tenant.FromContext(context.Background()); got != tenant.DefaultTenantID {
		t.Fatalf("no tenant set: got %s, want the default tenant", got)
	}
	id := uuid.New()
	if got := tenant.FromContext(WithTenant(context.Background(), id)); got != id {
		t.Fatalf("server.WithTenant(%s) read back as %s", id, got)
	}
}

func TestWithTenantLimitsOption(t *testing.T) {
	o := &options{}
	if o.limits != nil || o.queues != nil {
		t.Fatal("no option must leave limits and queues unset (self-hosted)")
	}

	paid := uuid.New()
	WithTenantLimits(map[string]int{"free": 2}, func(_ context.Context, id uuid.UUID) TenantLimits {
		if id == paid {
			return TenantLimits{}
		}
		return TenantLimits{MaxInFlight: 1, Queue: "free"}
	})(o)

	if o.queues["free"] != 2 {
		t.Fatalf("queues = %v", o.queues)
	}
	if got := o.limits(context.Background(), uuid.New()); got != (TenantLimits{MaxInFlight: 1, Queue: "free"}) {
		t.Fatalf("limits for a free tenant = %+v", got)
	}
	if got := o.limits(context.Background(), paid); got != (TenantLimits{}) {
		t.Fatalf("limits for a paid tenant = %+v", got)
	}
}
