package webhooks

import (
	"context"
	"errors"
	"testing"

	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

func TestListenSessionsAreOffUnlessEnabled(t *testing.T) {
	svc := NewWebhookService(nil, nil, nil)
	ctx := context.Background()

	calls := map[string]error{}
	_, calls["create"] = svc.CreateListenSession(ctx, ListenSessionRequest{Consumer: "c", Events: []string{"e"}})
	_, calls["claim"] = svc.ClaimListenDeliveries(ctx, "c", "00000000-0000-0000-0000-000000000001", 0)
	calls["respond"] = svc.RespondListenDelivery(ctx, "c", "00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002", 200, nil, nil)
	calls["delete"] = svc.DeleteListenSession(ctx, "c", "00000000-0000-0000-0000-000000000001")
	for name, err := range calls {
		var se *svcerrors.ServiceError
		if !errors.As(err, &se) || se.Status != svcerrors.PermissionDenied {
			t.Errorf("%s: err %v, want PermissionDenied while listen sessions are disabled", name, err)
		}
	}
}
