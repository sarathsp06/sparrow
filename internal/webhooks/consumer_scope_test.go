package webhooks

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/mock"

	svcerrors "github.com/sarathsp06/sparrow/internal/errors"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

func (m *mockRepo) GetBatchJob(ctx context.Context, tenantID uuid.UUID, batchID uuid.UUID) (*store.BatchJob, error) {
	args := m.Called(ctx, tenantID, batchID)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.(*store.BatchJob), args.Error(1)
}

func (m *mockRepo) UpdateBatchJobStatus(ctx context.Context, _ uuid.UUID, batchID uuid.UUID, from, to store.BatchJobStatus) error {
	return m.Called(ctx, batchID, from, to).Error(0)
}

func (m *mockRepo) GetDeliveryAttempts(ctx context.Context, tenantID uuid.UUID, deliveryID uuid.UUID) ([]*store.WebhookHealthEvent, error) {
	args := m.Called(ctx, tenantID, deliveryID)
	return args.Get(0).([]*store.WebhookHealthEvent), args.Error(1)
}

// A batch job is reachable only from its own consumer's path. A job
// snapshotted across all consumers is reachable from any consumer path by a
// tenant-wide caller, never by a consumer-scoped one (portal token).
func TestBatchJobsStayInTheirConsumer(t *testing.T) {
	cases := []struct {
		name        string
		jobConsumer string
		scope       BatchScope
		reachable   bool
	}{
		{"own consumer", "acme", BatchScope{Consumer: "acme"}, true},
		{"own consumer, portal", "acme", BatchScope{Consumer: "acme", ConsumerOnly: true}, true},
		{"other consumer", "acme", BatchScope{Consumer: "globex"}, false},
		{"other consumer, portal", "acme", BatchScope{Consumer: "globex", ConsumerOnly: true}, false},
		{"all-consumers job, tenant-wide caller", "", BatchScope{Consumer: "default"}, true},
		{"all-consumers job, portal", "", BatchScope{Consumer: "acme", ConsumerOnly: true}, false},
	}
	ops := []struct {
		name string
		call func(svc *WebhookService, scope BatchScope, id string) error
	}{
		{"GetRetryStatus", func(svc *WebhookService, scope BatchScope, id string) error {
			_, err := svc.GetRetryStatus(testContext(), scope, id)
			return err
		}},
		{"CancelRetry", func(svc *WebhookService, scope BatchScope, id string) error {
			return svc.CancelRetry(testContext(), scope, id)
		}},
		{"RetryDeliveries", func(svc *WebhookService, scope BatchScope, id string) error {
			return svc.RetryDeliveries(testContext(), scope, id)
		}},
	}
	for _, op := range ops {
		for _, tc := range cases {
			t.Run(op.name+"/"+tc.name, func(t *testing.T) {
				repo := &mockRepo{}
				inserter := &mockJobInserter{}
				svc := NewWebhookService(inserter, repo, nil)
				id := uuid.New()
				repo.On("GetBatchJob", mock.Anything, tenant.DefaultTenantID, id).Return(&store.BatchJob{
					ID:        id,
					Consumer:  tc.jobConsumer,
					JobType:   store.BatchTypeDeliveryRetry,
					Status:    store.BatchStatusPending,
					ExpiresAt: time.Now().Add(time.Hour),
				}, nil)
				repo.On("UpdateBatchJobStatus", mock.Anything, id, mock.Anything, mock.Anything).Return(nil)
				inserter.On("Insert", mock.Anything, mock.Anything).Return(&rivertype.JobInsertResult{}, nil)

				err := op.call(svc, tc.scope, id.String())
				if !tc.reachable {
					requireStatus(t, err, svcerrors.NotFound)
					repo.AssertNotCalled(t, "UpdateBatchJobStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
					inserter.AssertNotCalled(t, "Insert", mock.Anything, mock.Anything)
					return
				}
				if err != nil {
					t.Fatalf("in-scope job: %v", err)
				}
			})
		}
	}
}

// Attempts of another consumer's delivery answer not found and are never read.
func TestGetDeliveryAttempts_StaysInTheConsumer(t *testing.T) {
	id := uuid.New()
	attempts := []*store.WebhookHealthEvent{{DeliveryID: id}}

	t.Run("other consumer", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewWebhookService(&mockJobInserter{}, repo, nil)
		repo.On("GetDeliveryByID", mock.Anything, tenant.DefaultTenantID, id, "globex").Return(nil, nil)

		_, err := svc.GetDeliveryAttempts(testContext(), id.String(), "globex")
		requireStatus(t, err, svcerrors.NotFound)
		repo.AssertNotCalled(t, "GetDeliveryAttempts", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("own consumer", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewWebhookService(&mockJobInserter{}, repo, nil)
		repo.On("GetDeliveryByID", mock.Anything, tenant.DefaultTenantID, id, "acme").Return(&store.WebhookDelivery{ID: id}, nil)
		repo.On("GetDeliveryAttempts", mock.Anything, tenant.DefaultTenantID, id).Return(attempts, nil)

		got, err := svc.GetDeliveryAttempts(testContext(), id.String(), "acme")
		if err != nil || len(got) != 1 {
			t.Fatalf("got %v, %v; want the attempts", got, err)
		}
	})

	t.Run("no consumer looks up by id alone", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewWebhookService(&mockJobInserter{}, repo, nil)
		repo.On("GetDeliveryAttempts", mock.Anything, tenant.DefaultTenantID, id).Return(attempts, nil)

		if _, err := svc.GetDeliveryAttempts(testContext(), id.String(), ""); err != nil {
			t.Fatal(err)
		}
		repo.AssertNotCalled(t, "GetDeliveryByID", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}
