package tenant

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestFromContext(t *testing.T) {
	if got := FromContext(context.Background()); got != DefaultTenantID {
		t.Fatalf("no tenant set: got %s, want the default tenant", got)
	}
	if got := FromContext(WithTenant(context.Background(), uuid.Nil)); got != DefaultTenantID {
		t.Fatalf("nil tenant: got %s, want the default tenant", got)
	}
	id := uuid.New()
	ctx := WithTenant(context.Background(), id)
	if got := FromContext(ctx); got != id {
		t.Fatalf("got %s, want %s", got, id)
	}
	if got := FromContext(context.WithoutCancel(ctx)); got != id {
		t.Fatalf("detached context lost the tenant: got %s", got)
	}
}
