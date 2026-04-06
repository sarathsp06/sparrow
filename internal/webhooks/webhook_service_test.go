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

func (m *mockRepo) ListWebhooksPaginated(ctx context.Context, tenantID uuid.UUID, namespace, event string, activeOnly bool, limit, offset int) ([]*store.WebhookRegistration, int, error) {
	args := m.Called(ctx, tenantID, namespace, event, activeOnly, limit, offset)
	return args.Get(0).([]*store.WebhookRegistration), args.Int(1), args.Error(2)
}

func (m *mockRepo) ListEventsPaginated(ctx context.Context, tenantID uuid.UUID, activeOnly bool, limit, offset int) ([]*store.EventRegistration, int, error) {
	args := m.Called(ctx, tenantID, activeOnly, limit, offset)
	return args.Get(0).([]*store.EventRegistration), args.Int(1), args.Error(2)
}

func (m *mockRepo) ListSubscriptions(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID) ([]*store.EventSubscription, error) {
	args := m.Called(ctx, tenantID, webhookID)
	return args.Get(0).([]*store.EventSubscription), args.Error(1)
}

func (m *mockRepo) GetWebhookByID(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, namespace string) (*store.WebhookRegistration, error) {
	args := m.Called(ctx, tenantID, webhookID, namespace)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.(*store.WebhookRegistration), args.Error(1)
}

func (m *mockRepo) GetDeliveryByID(ctx context.Context, tenantID uuid.UUID, deliveryID uuid.UUID, namespace string) (*store.WebhookDelivery, error) {
	args := m.Called(ctx, tenantID, deliveryID, namespace)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.(*store.WebhookDelivery), args.Error(1)
}

func (m *mockRepo) GetRetriableDeliveries(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, namespace string, force bool) ([]*store.WebhookDelivery, error) {
	args := m.Called(ctx, tenantID, webhookID, namespace, force)
	return args.Get(0).([]*store.WebhookDelivery), args.Error(1)
}

func (m *mockRepo) ResetDeliveryForRetry(ctx context.Context, deliveryID uuid.UUID) error {
	args := m.Called(ctx, deliveryID)
	return args.Error(0)
}

func (m *mockRepo) GetDeliveriesByWebhookID(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, namespace string, limit, offset int) ([]*store.WebhookDelivery, int, error) {
	args := m.Called(ctx, tenantID, webhookID, namespace, limit, offset)
	return args.Get(0).([]*store.WebhookDelivery), args.Int(1), args.Error(2)
}

func (m *mockRepo) ListDeliveriesPaginated(ctx context.Context, tenantID uuid.UUID, namespace string, limit, offset int) ([]*store.WebhookDelivery, int, error) {
	args := m.Called(ctx, tenantID, namespace, limit, offset)
	return args.Get(0).([]*store.WebhookDelivery), args.Int(1), args.Error(2)
}

func (m *mockRepo) ListDeliveriesFiltered(ctx context.Context, tenantID uuid.UUID, filter store.DeliveryFilter) ([]*store.WebhookDelivery, int, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]*store.WebhookDelivery), args.Int(1), args.Error(2)
}

func (m *mockRepo) ListEventReportsFiltered(ctx context.Context, tenantID uuid.UUID, filter store.EventReportFilter) ([]*store.EventReportWithStats, int, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]*store.EventReportWithStats), args.Int(1), args.Error(2)
}

func (m *mockRepo) CreateSubscription(ctx context.Context, tenantID uuid.UUID, sub *store.EventSubscription) error {
	args := m.Called(ctx, tenantID, sub)
	sub.ID = uuid.New()
	sub.CreatedAt = time.Now()
	return args.Error(0)
}

func (m *mockRepo) GetEventByName(ctx context.Context, tenantID uuid.UUID, eventName string) (*store.EventRegistration, error) {
	args := m.Called(ctx, tenantID, eventName)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.(*store.EventRegistration), args.Error(1)
}

func (m *mockRepo) RegisterEvent(ctx context.Context, tenantID uuid.UUID, event *store.EventRegistration) error {
	args := m.Called(ctx, tenantID, event)
	event.TenantID = tenantID
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()
	return args.Error(0)
}

func (m *mockRepo) StoreEvent(ctx context.Context, tenantID uuid.UUID, event *store.EventRecord) error {
	args := m.Called(ctx, tenantID, event)
	return args.Error(0)
}

// testContext returns a context for testing.
func testContext() context.Context {
	return context.Background()
}

func TestWebhookService_ListWebhooks_Pagination(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()
	namespace := "default"
	limit := int32(10)
	offset := int32(0)

	expectedWebhooks := []*store.WebhookRegistration{
		{ID: uuid.New(), Namespace: namespace, URL: "http://example.com"},
	}

	repo.On("ListWebhooksPaginated", mock.Anything, mock.Anything, namespace, "", false, int(limit), int(offset)).
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
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()
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

	repo.On("GetDeliveryByID", mock.Anything, mock.Anything, deliveryID, namespace).Return(delivery, nil)
	repo.On("ResetDeliveryForRetry", mock.Anything, deliveryID).Return(nil)
	repo.On("GetWebhookByID", mock.Anything, mock.Anything, webhookID, namespace).Return(webhook, nil)
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
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()
	namespace := "default"

	expectedDeliveries := []*store.WebhookDelivery{
		{ID: uuid.New(), WebhookID: uuid.New(), EventID: uuid.New()},
	}

	// The service normalises limit/offset, so match the filter as built by the service.
	repo.On("ListDeliveriesFiltered", mock.Anything, mock.Anything, mock.MatchedBy(func(f store.DeliveryFilter) bool {
		return f.Namespace == namespace && f.Limit == 20 && f.Offset == 0
	})).Return(expectedDeliveries, 1, nil)

	filter := store.DeliveryFilter{
		Namespace: namespace,
		Limit:     20,
		Offset:    0,
	}
	deliveries, totalCount, _, err := service.ListDeliveries(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, int32(1), totalCount)
	assert.Equal(t, len(expectedDeliveries), len(deliveries))
	repo.AssertExpectations(t)
}

func TestWebhookService_CreateSubscription(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()
	webhookID := uuid.New().String()
	namespace := "default"
	eventName := "user.created"

	repo.On("CreateSubscription", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	id, createdAt, err := service.CreateSubscription(ctx, webhookID, eventName, namespace, nil, "POST", 30, false, "", nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.False(t, createdAt.IsZero())
	repo.AssertExpectations(t)
}

func TestWebhookService_GetEvent(t *testing.T) {
	repo := new(mockRepo)
	service := NewWebhookService(nil, repo, nil)

	ctx := testContext()
	eventName := "test.event"
	event := &store.EventRegistration{
		Name: eventName,
	}

	repo.On("GetEventByName", mock.Anything, mock.Anything, eventName).Return(event, nil)

	res, err := service.GetEvent(ctx, eventName)
	assert.NoError(t, err)
	assert.Equal(t, event, res)
	repo.AssertExpectations(t)
}

func TestWebhookService_TestSubscriptionTemplate(t *testing.T) {
	repo := new(mockRepo)
	service := NewWebhookService(nil, repo, nil)

	ctx := testContext()
	eventName := "test.event"
	event := &store.EventRegistration{
		Name: eventName,
		SamplePayload: map[string]any{
			"id": "123",
		},
	}

	repo.On("GetEventByName", mock.Anything, mock.Anything, eventName).Return(event, nil)

	template := `{"new_id": "{{ .payload.id }}"}`
	res, err := service.TestSubscriptionTemplate(ctx, eventName, template, "default")
	assert.NoError(t, err)
	assert.Equal(t, `{"new_id": "123"}`, res)
	repo.AssertExpectations(t)
}

func TestWebhookService_PushEvent_AutoRegister(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()
	namespace := "default"
	eventName := "user.signup"
	payload := map[string]any{"user_id": "123"}

	// Event does not exist — should be auto-registered
	repo.On("GetEventByName", mock.Anything, mock.Anything, eventName).Return(nil, nil)
	repo.On("RegisterEvent", mock.Anything, mock.Anything, mock.MatchedBy(func(e *store.EventRegistration) bool {
		return e.Name == eventName && e.Active == true
	})).Return(nil)
	repo.On("StoreEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	inserter.On("Insert", mock.Anything, mock.Anything).Return(&rivertype.JobInsertResult{}, nil)

	eventID, _, err := service.PushEvent(ctx, namespace, eventName, payload, 0, nil, nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, eventID)
	repo.AssertExpectations(t)
	inserter.AssertExpectations(t)
}

func TestWebhookService_PushEvent_ExistingEvent(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()
	namespace := "default"
	eventName := "user.created"
	payload := map[string]any{"user_id": "456"}

	// Event already registered and active
	repo.On("GetEventByName", mock.Anything, mock.Anything, eventName).Return(&store.EventRegistration{
		Name:   eventName,
		Active: true,
	}, nil)
	repo.On("StoreEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	inserter.On("Insert", mock.Anything, mock.Anything).Return(&rivertype.JobInsertResult{}, nil)

	eventID, _, err := service.PushEvent(ctx, namespace, eventName, payload, 0, nil, nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, eventID)
	// RegisterEvent should NOT be called since event already exists
	repo.AssertNotCalled(t, "RegisterEvent", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
	inserter.AssertExpectations(t)
}

func TestWebhookService_PushEvent_InactiveEvent(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()

	// Event exists but inactive
	repo.On("GetEventByName", mock.Anything, mock.Anything, "user.deleted").Return(&store.EventRegistration{
		Name:   "user.deleted",
		Active: false,
	}, nil)

	_, _, err := service.PushEvent(ctx, "default", "user.deleted", nil, 0, nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inactive")
	repo.AssertExpectations(t)
}

func TestWebhookService_CreateSubscription_CatchAll(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()
	webhookID := uuid.New().String()
	namespace := "default"

	repo.On("CreateSubscription", mock.Anything, mock.Anything, mock.MatchedBy(func(sub *store.EventSubscription) bool {
		return sub.EventName == store.CatchAllEventName
	})).Return(nil)

	id, createdAt, err := service.CreateSubscription(ctx, webhookID, store.CatchAllEventName, namespace, nil, "POST", 30, false, "", nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.False(t, createdAt.IsZero())
	repo.AssertExpectations(t)
}
