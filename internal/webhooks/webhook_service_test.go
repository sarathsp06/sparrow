package webhooks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// MockRepository is a mock implementation of the store.RepositoryInterface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) RegisterWebhook(ctx context.Context, registration *store.WebhookRegistration) error {
	args := m.Called(ctx, registration)
	return args.Error(0)
}

// Implement the rest of the store.RepositoryInterface methods
func (m *MockRepository) UpdateWebhookHealthState(ctx context.Context, webhookID string, success bool, eventTimestamp time.Time) error {
	args := m.Called(ctx, webhookID, success, eventTimestamp)
	return args.Error(0)
}

func (m *MockRepository) CalculateWebhookHealth(ctx context.Context, webhookID string, lookbackHours int) (string, error) {
	args := m.Called(ctx, webhookID, lookbackHours)
	return args.String(0), args.Error(1)
}

func (m *MockRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

func (m *MockRepository) StoreEventTx(ctx context.Context, tx pgx.Tx, event *store.EventRecord) error {
	args := m.Called(ctx, tx, event)
	return args.Error(0)
}

func (m *MockRepository) GetWebhooksByEventTx(ctx context.Context, tx pgx.Tx, namespace, event string) ([]*store.WebhookRegistration, error) {
	args := m.Called(ctx, tx, namespace, event)
	return args.Get(0).([]*store.WebhookRegistration), args.Error(1)
}

func (m *MockRepository) CreateDeliveryTx(ctx context.Context, tx pgx.Tx, delivery *store.WebhookDelivery) error {
	args := m.Called(ctx, tx, delivery)
	return args.Error(0)
}

func (m *MockRepository) UnregisterWebhook(ctx context.Context, webhookID string) error {
	args := m.Called(ctx, webhookID)
	return args.Error(0)
}

func (m *MockRepository) GetWebhooksByEvent(ctx context.Context, namespace, event string) ([]*store.WebhookRegistration, error) {
	args := m.Called(ctx, namespace, event)
	return args.Get(0).([]*store.WebhookRegistration), args.Error(1)
}

func (m *MockRepository) ListWebhooks(ctx context.Context, namespace string, activeOnly bool) ([]*store.WebhookRegistration, error) {
	args := m.Called(ctx, namespace, activeOnly)
	return args.Get(0).([]*store.WebhookRegistration), args.Error(1)
}

func (m *MockRepository) StoreEvent(ctx context.Context, event *store.EventRecord) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockRepository) CreateDelivery(ctx context.Context, delivery *store.WebhookDelivery) error {
	args := m.Called(ctx, delivery)
	return args.Error(0)
}

func (m *MockRepository) UpdateDeliveryStatus(ctx context.Context, deliveryID string, status store.WebhookDeliveryStatus, responseCode int, responseBody, errorMessage string) error {
	args := m.Called(ctx, deliveryID, status, responseCode, responseBody, errorMessage)
	return args.Error(0)
}

func (m *MockRepository) UpdateDeliveryRequestBody(ctx context.Context, deliveryID string, requestBody string) error {
	args := m.Called(ctx, deliveryID, requestBody)
	return args.Error(0)
}

func (m *MockRepository) GetDeliveriesByWebhook(ctx context.Context, webhookID string) ([]*store.WebhookDelivery, error) {
	args := m.Called(ctx, webhookID)
	return args.Get(0).([]*store.WebhookDelivery), args.Error(1)
}

func (m *MockRepository) GetDeliveriesByEvent(ctx context.Context, eventID string) ([]*store.WebhookDelivery, error) {
	args := m.Called(ctx, eventID)
	return args.Get(0).([]*store.WebhookDelivery), args.Error(1)
}

func (m *MockRepository) GetWebhookByID(ctx context.Context, webhookID, namespace string) (*store.WebhookRegistration, error) {
	args := m.Called(ctx, webhookID, namespace)
	return args.Get(0).(*store.WebhookRegistration), args.Error(1)
}

func (m *MockRepository) GetWebhooksByNamespace(ctx context.Context, namespace string, activeOnly bool) ([]*store.WebhookRegistration, error) {
	args := m.Called(ctx, namespace, activeOnly)
	return args.Get(0).([]*store.WebhookRegistration), args.Error(1)
}

func (m *MockRepository) UpdateWebhook(ctx context.Context, webhook *store.WebhookRegistration) error {
	args := m.Called(ctx, webhook)
	return args.Error(0)
}

func (m *MockRepository) GetDeliveryByID(ctx context.Context, deliveryID, namespace string) (*store.WebhookDelivery, error) {
	args := m.Called(ctx, deliveryID, namespace)
	return args.Get(0).(*store.WebhookDelivery), args.Error(1)
}

func (m *MockRepository) GetDeliveriesByWebhookID(ctx context.Context, webhookID, namespace string, limit, offset int) ([]*store.WebhookDelivery, int, error) {
	args := m.Called(ctx, webhookID, namespace, limit, offset)
	return args.Get(0).([]*store.WebhookDelivery), args.Int(1), args.Error(2)
}

func (m *MockRepository) RegisterEvent(ctx context.Context, event *store.EventRegistration) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockRepository) GetEventByName(ctx context.Context, eventName string) (*store.EventRegistration, error) {
	args := m.Called(ctx, eventName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.EventRegistration), args.Error(1)
}

func (m *MockRepository) ListEvents(ctx context.Context, activeOnly bool) ([]*store.EventRegistration, error) {
	args := m.Called(ctx, activeOnly)
	return args.Get(0).([]*store.EventRegistration), args.Error(1)
}

func (m *MockRepository) UpdateEvent(ctx context.Context, event *store.EventRegistration) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockRepository) DeleteEvent(ctx context.Context, eventName string) error {
	args := m.Called(ctx, eventName)
	return args.Error(0)
}

func (m *MockRepository) RecordWebhookHealthEvent(ctx context.Context, webhookID, deliveryID string, success bool, responseTime, responseCode int, errorMessage string) error {
	args := m.Called(ctx, webhookID, deliveryID, success, responseTime, responseCode, errorMessage)
	return args.Error(0)
}

func (m *MockRepository) GetWebhookHealthState(ctx context.Context, webhookID string) (*store.WebhookHealthMetrics, error) {
	args := m.Called(ctx, webhookID)
	return args.Get(0).(*store.WebhookHealthMetrics), args.Error(1)
}

func (m *MockRepository) GetWebhookHealthSummary(ctx context.Context, webhookID string, hours int) (*store.WebhookHealthSummary, error) {
	args := m.Called(ctx, webhookID, hours)
	return args.Get(0).(*store.WebhookHealthSummary), args.Error(1)
}

func (m *MockRepository) GetWebhookHealthTimeSeries(ctx context.Context, webhookID string, hours int, bucketSize string) ([]*store.WebhookHealthEvent, error) {
	args := m.Called(ctx, webhookID, hours, bucketSize)
	return args.Get(0).([]*store.WebhookHealthEvent), args.Error(1)
}

func (m *MockRepository) AggregateHealthSummaries(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetWebhooksByHealth(ctx context.Context, health store.WebhookHealth) ([]*store.WebhookRegistration, error) {
	args := m.Called(ctx, health)
	return args.Get(0).([]*store.WebhookRegistration), args.Error(1)
}

func (m *MockRepository) GetHealthSummary(ctx context.Context) (map[store.WebhookHealth]int, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[store.WebhookHealth]int), args.Error(1)
}

func (m *MockRepository) GetRetriableDeliveries(ctx context.Context, webhookID, namespace string, force bool) ([]*store.WebhookDelivery, error) {
	args := m.Called(ctx, webhookID, namespace, force)
	return args.Get(0).([]*store.WebhookDelivery), args.Error(1)
}

func (m *MockRepository) ResetDeliveryForRetry(ctx context.Context, deliveryID string) error {
	args := m.Called(ctx, deliveryID)
	return args.Error(0)
}

func (m *MockRepository) GetEventByID(ctx context.Context, eventID string) (*store.EventRecord, error) {
	args := m.Called(ctx, eventID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.EventRecord), args.Error(1)
}

func (m *MockRepository) ListEventReportsWithStats(ctx context.Context, namespace string, eventName *string, limit, offset int) ([]*store.EventReportWithStats, int, error) {
	args := m.Called(ctx, namespace, eventName, limit, offset)
	return args.Get(0).([]*store.EventReportWithStats), args.Int(1), args.Error(2)
}

func (m *MockRepository) ListEventReports(ctx context.Context, namespace string, eventName *string, limit, offset int) ([]*store.EventReportWithStats, int, error) {
	args := m.Called(ctx, namespace, eventName, limit, offset)
	return args.Get(0).([]*store.EventReportWithStats), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetEventDeliveryStats(ctx context.Context, eventID string) (int32, int32, int32, int32, error) {
	args := m.Called(ctx, eventID)
	return args.Get(0).(int32), args.Get(1).(int32), args.Get(2).(int32), args.Get(3).(int32), args.Error(4)
}

// MockQueueManager is a mock implementation of the queue.QueueManagerInterface

func TestRegisterWebhook(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	service := NewWebhookService(nil, mockRepo)

	t.Run("successful registration", func(t *testing.T) {
		mockRepo.On("RegisterWebhook", mock.Anything, mock.Anything).Return(nil).Once()
		_, _, err := service.RegisterWebhook(ctx, "test-namespace", []string{"event1"}, "http://test.url", nil, 30, true, "description")
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("missing namespace", func(t *testing.T) {
		_, _, err := service.RegisterWebhook(ctx, "", []string{"event1"}, "http://test.url", nil, 30, true, "description")
		assert.Error(t, err)
	})

	t.Run("no events", func(t *testing.T) {
		_, _, err := service.RegisterWebhook(ctx, "test-namespace", []string{}, "http://test.url", nil, 30, true, "description")
		assert.Error(t, err)
	})

	t.Run("empty url", func(t *testing.T) {
		_, _, err := service.RegisterWebhook(ctx, "test-namespace", []string{"event1"}, "", nil, 30, true, "description")
		assert.Error(t, err)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo.On("RegisterWebhook", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()
		_, _, err := service.RegisterWebhook(ctx, "test-namespace", []string{"event1"}, "http://test.url", nil, 30, true, "description")
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestDeleteEvent(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	service := NewWebhookService(nil, mockRepo)

	t.Run("successful deletion", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(&store.EventRegistration{}, nil).Once()
		mockRepo.On("DeleteEvent", mock.Anything, "event1").Return(nil).Once()
		err := service.DeleteEvent(ctx, "event1")
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("event not found", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(nil, nil).Once()
		err := service.DeleteEvent(ctx, "event1")
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error on get", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(nil, errors.New("db error")).Once()
		err := service.DeleteEvent(ctx, "event1")
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error on delete", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(&store.EventRegistration{}, nil).Once()
		mockRepo.On("DeleteEvent", mock.Anything, "event1").Return(errors.New("db error")).Once()
		err := service.DeleteEvent(ctx, "event1")
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestUpdateEvent(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	service := NewWebhookService(nil, mockRepo)

	t.Run("successful update", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(&store.EventRegistration{}, nil).Once()
		mockRepo.On("UpdateEvent", mock.Anything, mock.Anything).Return(nil).Once()
		err := service.UpdateEvent(ctx, "event1", "new description", map[string]any{}, nil, true)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("event not found", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(nil, nil).Once()
		err := service.UpdateEvent(ctx, "event1", "new description", map[string]any{}, nil, true)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error on get", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(nil, errors.New("db error")).Once()
		err := service.UpdateEvent(ctx, "event1", "new description", map[string]any{}, nil, true)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error on update", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(&store.EventRegistration{}, nil).Once()
		mockRepo.On("UpdateEvent", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()
		err := service.UpdateEvent(ctx, "event1", "new description", map[string]any{}, nil, true)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestRegisterEvent(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	service := NewWebhookService(nil, mockRepo)

	t.Run("successful registration", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(nil, nil).Once()
		mockRepo.On("RegisterEvent", mock.Anything, mock.Anything).Return(nil).Once()
		_, _, err := service.RegisterEvent(ctx, "event1", "description", map[string]any{}, nil, true)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("event already exists", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(&store.EventRegistration{}, nil).Once()
		_, _, err := service.RegisterEvent(ctx, "event1", "description", map[string]any{}, nil, true)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error on get", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(nil, errors.New("db error")).Once()
		_, _, err := service.RegisterEvent(ctx, "event1", "description", map[string]any{}, nil, true)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error on register", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(nil, nil).Once()
		mockRepo.On("RegisterEvent", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()
		_, _, err := service.RegisterEvent(ctx, "event1", "description", map[string]any{}, nil, true)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

// MockQueueManager is a mock implementation of the queue.QueueManagerInterface
type MockQueueManager struct {
	mock.Mock
}

func (m *MockQueueManager) Insert(ctx context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error) {
	argsMock := m.Called(ctx, args)
	return argsMock.Get(0).(*rivertype.JobInsertResult), argsMock.Error(1)
}

func (m *MockQueueManager) BatchInsert(ctx context.Context, args []river.JobArgs) ([]*rivertype.JobInsertResult, error) {
	argsMock := m.Called(ctx, args)
	return argsMock.Get(0).([]*rivertype.JobInsertResult), argsMock.Error(1)
}

func TestPushEvent(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockQueue := new(MockQueueManager)
	service := NewWebhookService(mockQueue, mockRepo)

	t.Run("successful push", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(&store.EventRegistration{Active: true}, nil).Once()
		mockRepo.On("StoreEvent", mock.Anything, mock.Anything).Return(nil).Once()
		mockQueue.On("Insert", mock.Anything, mock.Anything).Return(&rivertype.JobInsertResult{}, nil).Once()

		payload := map[string]any{"key": "value"}
		_, err := service.PushEvent(ctx, "test-namespace", "event1", payload, 3600, nil)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockQueue.AssertExpectations(t)
	})

	t.Run("missing namespace", func(t *testing.T) {
		payload := map[string]any{"key": "value"}
		_, err := service.PushEvent(ctx, "", "event1", payload, 3600, nil)
		assert.Error(t, err)
	})

	t.Run("unregistered event", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(nil, errors.New("not found")).Once()
		payload := map[string]any{"key": "value"}
		_, err := service.PushEvent(ctx, "test-namespace", "event1", payload, 3600, nil)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("schema validation fail", func(t *testing.T) {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key": map[string]any{"type": "string"},
			},
			"required": []string{"key"},
		}
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(&store.EventRegistration{Active: true, Schema: schema}, nil).Once()

		payload := map[string]any{"key": 123}
		_, err := service.PushEvent(ctx, "test-namespace", "event1", payload, 3600, nil)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("queue error", func(t *testing.T) {
		mockRepo.On("GetEventByName", mock.Anything, "event1").Return(&store.EventRegistration{Active: true}, nil).Once()
		mockRepo.On("StoreEvent", mock.Anything, mock.Anything).Return(nil).Once()
		mockQueue.On("Insert", mock.Anything, mock.Anything).Return(&rivertype.JobInsertResult{}, errors.New("queue error")).Once()

		payload := map[string]any{"key": "value"}
		_, err := service.PushEvent(ctx, "test-namespace", "event1", payload, 3600, nil)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
		mockQueue.AssertExpectations(t)
	})
}

func TestUnregisterWebhook(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	service := NewWebhookService(nil, mockRepo)

	t.Run("successful unregistration", func(t *testing.T) {
		mockRepo.On("UnregisterWebhook", ctx, "webhook-id").Return(nil).Once()
		err := service.UnregisterWebhook(ctx, "webhook-id")
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty webhook id", func(t *testing.T) {
		err := service.UnregisterWebhook(ctx, "")
		assert.Error(t, err)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo.On("UnregisterWebhook", ctx, "webhook-id").Return(errors.New("db error")).Once()
		err := service.UnregisterWebhook(ctx, "webhook-id")
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}
