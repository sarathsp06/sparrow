package rest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/sarathsp06/sparrow/internal/middleware"
	"github.com/sarathsp06/sparrow/internal/rest"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// scopeRecorder records the consumer scope each batch-job and attempts call
// receives; every other method panics via the nil embedded interface.
type scopeRecorder struct {
	webhooks.WebhookServiceInterface
	scopes    []webhooks.BatchScope
	consumers []string
}

func (s *scopeRecorder) job(scope webhooks.BatchScope) (*store.BatchJob, error) {
	s.scopes = append(s.scopes, scope)
	return &store.BatchJob{JobType: store.BatchTypeDeliveryRetry, Status: store.BatchStatusPending}, nil
}

func (s *scopeRecorder) GetRetryStatus(_ context.Context, scope webhooks.BatchScope, _ string) (*store.BatchJob, error) {
	return s.job(scope)
}

func (s *scopeRecorder) GetRepushStatus(_ context.Context, scope webhooks.BatchScope, _ string) (*store.BatchJob, error) {
	return s.job(scope)
}

func (s *scopeRecorder) CancelRetry(_ context.Context, scope webhooks.BatchScope, _ string) error {
	_, err := s.job(scope)
	return err
}

func (s *scopeRecorder) RetryDeliveries(_ context.Context, scope webhooks.BatchScope, _ string) error {
	_, err := s.job(scope)
	return err
}

func (s *scopeRecorder) GetDeliveryAttempts(_ context.Context, _ string, consumer string) ([]*store.WebhookHealthEvent, error) {
	s.consumers = append(s.consumers, consumer)
	return nil, nil
}

// Portal requests reach batch jobs and attempts pinned to the token's consumer
// and limited to it; tenant-wide /v1 requests keep the path consumer and may
// also reach all-consumers jobs.
func TestBatchJobsAndAttemptsCarryTheConsumerScope(t *testing.T) {
	svc := &scopeRecorder{}
	r := chi.NewRouter()
	rest.Mount(r, svc, rest.AccessDeps{}, rest.AIDeps{})
	portal := middleware.PortalGateway(func(context.Context, string) (string, error) { return "acme", nil }, r)
	const id = "11111111-1111-1111-1111-111111111111"

	send := func(h http.Handler, method, path, body, token string) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code >= 300 {
			t.Fatalf("%s %s = %d %s", method, path, rec.Code, rec.Body)
		}
	}

	send(portal, http.MethodGet, "/portal/api/retry-jobs/"+id, "", "t")
	send(portal, http.MethodGet, "/portal/api/repush-jobs/"+id, "", "t")
	send(portal, http.MethodPost, "/portal/api/retry-jobs/"+id+":cancel", "", "t")
	send(portal, http.MethodPost, "/portal/api/deliveries:retryBatch", `{"repush_id":"`+id+`"}`, "t")
	for i, got := range svc.scopes {
		if want := (webhooks.BatchScope{Consumer: "acme", ConsumerOnly: true}); got != want {
			t.Fatalf("portal call %d scope = %+v, want %+v", i, got, want)
		}
	}

	svc.scopes = nil
	send(r, http.MethodGet, "/v1/consumers/globex/retry-jobs/"+id, "", "")
	if want := (webhooks.BatchScope{Consumer: "globex"}); len(svc.scopes) != 1 || svc.scopes[0] != want {
		t.Fatalf("tenant-wide scope = %+v, want %+v", svc.scopes, want)
	}

	send(portal, http.MethodGet, "/portal/api/deliveries/"+id+"/attempts", "", "t")
	send(r, http.MethodGet, "/v1/consumers/globex/deliveries/"+id+"/attempts", "", "")
	send(r, http.MethodGet, "/v1/deliveries/"+id+"/attempts", "", "")
	if want := []string{"acme", "globex", ""}; strings.Join(svc.consumers, ",") != strings.Join(want, ",") {
		t.Fatalf("attempts consumers = %q, want %q", svc.consumers, want)
	}
}
