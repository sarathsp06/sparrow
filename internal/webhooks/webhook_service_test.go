package webhooks

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

type mockJobInserter struct {
	mock.Mock
}

func (m *mockJobInserter) Insert(ctx context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error) {
	callArgs := m.Called(ctx, args)
	res := callArgs.Get(0)
	if res == nil {
		return nil, callArgs.Error(1)
	}
	return res.(*rivertype.JobInsertResult), callArgs.Error(1)
}

func (m *mockJobInserter) BatchInsert(ctx context.Context, args []river.JobArgs) ([]*rivertype.JobInsertResult, error) {
	callArgs := m.Called(ctx, args)
	res := callArgs.Get(0)
	if res == nil {
		return nil, callArgs.Error(1)
	}
	return res.([]*rivertype.JobInsertResult), callArgs.Error(1)
}

type mockRepo struct {
	store.RepositoryInterface
	mock.Mock
}

func (m *mockRepo) ListWebhooksPaginated(ctx context.Context, namespace, event string, activeOnly bool, limit, offset int) ([]*store.WebhookRegistration, int, error) {
	args := m.Called(ctx, namespace, event, activeOnly, limit, offset)
	return args.Get(0).([]*store.WebhookRegistration), args.Int(1), args.Error(2)
}

func (m *mockRepo) ListEventsPaginated(ctx context.Context, activeOnly bool, limit, offset int) ([]*store.EventRegistration, int, error) {
	args := m.Called(ctx, activeOnly, limit, offset)
	return args.Get(0).([]*store.EventRegistration), args.Int(1), args.Error(2)
}

func (m *mockRepo) ListSubscriptions(ctx context.Context, webhookID uuid.UUID) ([]*store.EventSubscription, error) {
	args := m.Called(ctx, webhookID)
	return args.Get(0).([]*store.EventSubscription), args.Error(1)
}

func (m *mockRepo) GetWebhookByID(ctx context.Context, webhookID uuid.UUID, namespace string) (*store.WebhookRegistration, error) {
	args := m.Called(ctx, webhookID, namespace)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.(*store.WebhookRegistration), args.Error(1)
}

func (m *mockRepo) GetDeliveryByID(ctx context.Context, deliveryID uuid.UUID, namespace string) (*store.WebhookDelivery, error) {
	args := m.Called(ctx, deliveryID, namespace)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.(*store.WebhookDelivery), args.Error(1)
}

func (m *mockRepo) GetRetriableDeliveries(ctx context.Context, webhookID uuid.UUID, namespace string, force bool) ([]*store.WebhookDelivery, error) {
	args := m.Called(ctx, webhookID, namespace, force)
	return args.Get(0).([]*store.WebhookDelivery), args.Error(1)
}

func (m *mockRepo) ResetDeliveryForRetry(ctx context.Context, deliveryID uuid.UUID) error {
	args := m.Called(ctx, deliveryID)
	return args.Error(0)
}

func (m *mockRepo) GetDeliveriesByWebhookID(ctx context.Context, webhookID uuid.UUID, namespace string, limit, offset int) ([]*store.WebhookDelivery, int, error) {
	args := m.Called(ctx, webhookID, namespace, limit, offset)
	return args.Get(0).([]*store.WebhookDelivery), args.Int(1), args.Error(2)
}

func (m *mockRepo) ListDeliveriesPaginated(ctx context.Context, namespace string, limit, offset int) ([]*store.WebhookDelivery, int, error) {
	args := m.Called(ctx, namespace, limit, offset)
	return args.Get(0).([]*store.WebhookDelivery), args.Int(1), args.Error(2)
}

func (m *mockRepo) CreateSubscription(ctx context.Context, sub *store.EventSubscription) error {
	args := m.Called(ctx, sub)
	sub.ID = uuid.New()
	sub.CreatedAt = time.Now()
	return args.Error(0)
}

func (m *mockRepo) GetEventByName(ctx context.Context, name string) (*store.EventRegistration, error) {
	args := m.Called(ctx, name)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.(*store.EventRegistration), args.Error(1)
}

func TestWebhookService_ListWebhooks_Pagination(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo)

	ctx := context.Background()
	namespace := "default"
	limit := int32(10)
	offset := int32(0)

	expectedWebhooks := []*store.WebhookRegistration{
		{ID: uuid.New(), Namespace: namespace, URL: "http://example.com"},
	}

	repo.On("ListWebhooksPaginated", mock.Anything, namespace, "", false, int(limit), int(offset)).
		Return(expectedWebhooks, 1, nil)

	webhooks, totalCount, err := service.ListWebhooks(ctx, namespace, "", "", false, limit, offset)

	assert.NoError(t, err)
	assert.Equal(t, int32(1), totalCount)
	assert.Equal(t, len(expectedWebhooks), len(webhooks))
	repo.AssertExpectations(t)
}

func TestWebhookService_RetryDelivery(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo)

	ctx := context.Background()
	namespace := "default"
	deliveryID := uuid.New()
	webhookID := uuid.New()

	delivery := &store.WebhookDelivery{
		ID:        deliveryID,
		WebhookID: webhookID,
		EventID:   uuid.New(),
		Status:    store.StatusFailed,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	webhook := &store.WebhookRegistration{
		ID:        webhookID,
		Namespace: namespace,
	}

	repo.On("GetDeliveryByID", mock.Anything, deliveryID, namespace).Return(delivery, nil)
	repo.On("ResetDeliveryForRetry", mock.Anything, deliveryID).Return(nil)
	repo.On("GetWebhookByID", mock.Anything, webhookID, namespace).Return(webhook, nil)
	inserter.On("Insert", mock.Anything, mock.Anything).Return(&rivertype.JobInsertResult{}, nil)

	ids, count, err := service.RetryDelivery(ctx, namespace, deliveryID.String(), "", false)

	assert.NoError(t, err)
	assert.Equal(t, int32(1), count)
	assert.Contains(t, ids, deliveryID.String())
	repo.AssertExpectations(t)
	inserter.AssertExpectations(t)
}

func TestWebhookService_ListDeliveries_Pagination(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo)

	ctx := context.Background()
	namespace := "default"
	limit := int32(20)
	offset := int32(0)

	expectedDeliveries := []*store.WebhookDelivery{
		{ID: uuid.New(), WebhookID: uuid.New(), EventID: uuid.New()},
	}

	repo.On("ListDeliveriesPaginated", mock.Anything, namespace, int(limit), int(offset)).
		Return(expectedDeliveries, 1, nil)

	deliveries, totalCount, err := service.ListDeliveries(ctx, namespace, "", "", limit, offset)

	assert.NoError(t, err)
	assert.Equal(t, int32(1), totalCount)
	assert.Equal(t, len(expectedDeliveries), len(deliveries))
	repo.AssertExpectations(t)
}

func TestWebhookService_CreateSubscription(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo)

	ctx := context.Background()
	webhookID := uuid.New().String()
	namespace := "default"
	eventName := "user.created"

	repo.On("CreateSubscription", mock.Anything, mock.Anything).Return(nil)

	id, createdAt, err := service.CreateSubscription(ctx, webhookID, eventName, namespace, nil, "POST", 30, false, "")

	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.False(t, createdAt.IsZero())
	repo.AssertExpectations(t)
}

func TestWebhookService_GetEvent(t *testing.T) {
	repo := new(mockRepo)
	service := NewWebhookService(nil, repo)

	ctx := context.Background()
	eventName := "test.event"
	event := &store.EventRegistration{
		ID:   uuid.New(),
		Name: eventName,
	}

	repo.On("GetEventByName", mock.Anything, eventName).Return(event, nil)

	res, err := service.GetEvent(ctx, eventName)
	assert.NoError(t, err)
	assert.Equal(t, event, res)
	repo.AssertExpectations(t)
}

func TestWebhookService_TestSubscriptionTemplate(t *testing.T) {
	repo := new(mockRepo)
	service := NewWebhookService(nil, repo)

	ctx := context.Background()
	eventName := "test.event"
	event := &store.EventRegistration{
		ID:   uuid.New(),
		Name: eventName,
		SamplePayload: map[string]any{
			"id": "123",
		},
	}

	repo.On("GetEventByName", mock.Anything, eventName).Return(event, nil)

	template := `{"new_id": "{{ .Payload.id }}"}`
	res, err := service.TestSubscriptionTemplate(ctx, eventName, template, "default")
	assert.NoError(t, err)
	assert.Equal(t, `{"new_id": "123"}`, res)
	repo.AssertExpectations(t)
}
