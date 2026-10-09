package webhooks

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	cryptosvc "github.com/sarathsp06/sparrow/pkg/crypto"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
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

func (m *mockRepo) ListWebhooksPaginated(ctx context.Context, tenantID uuid.UUID, consumer, event string, activeOnly bool, health store.WebhookHealth, limit, offset int) ([]*store.WebhookRegistration, int, error) {
	args := m.Called(ctx, tenantID, consumer, event, activeOnly, health, limit, offset)
	return args.Get(0).([]*store.WebhookRegistration), args.Int(1), args.Error(2)
}

func (m *mockRepo) ListEventsPaginated(ctx context.Context, tenantID uuid.UUID, filter store.EventTypeFilter, limit, offset int) ([]*store.EventRegistration, int, error) {
	args := m.Called(ctx, tenantID, filter, limit, offset)
	return args.Get(0).([]*store.EventRegistration), args.Int(1), args.Error(2)
}

func (m *mockRepo) ListSubscriptions(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID) ([]*store.EventSubscription, error) {
	args := m.Called(ctx, tenantID, webhookID)
	return args.Get(0).([]*store.EventSubscription), args.Error(1)
}

func (m *mockRepo) GetWebhookByID(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, consumer string) (*store.WebhookRegistration, error) {
	args := m.Called(ctx, tenantID, webhookID, consumer)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.(*store.WebhookRegistration), args.Error(1)
}

func (m *mockRepo) LockWebhook(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, consumer string, exclusive bool) (*store.WebhookRegistration, error) {
	args := m.Called(ctx, tenantID, webhookID, consumer, exclusive)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.(*store.WebhookRegistration), args.Error(1)
}

func (m *mockRepo) UpdateWebhook(ctx context.Context, tenantID uuid.UUID, webhook *store.WebhookRegistration) error {
	args := m.Called(ctx, tenantID, webhook)
	return args.Error(0)
}

func (m *mockRepo) RunInTransaction(fn func(store.RepositoryInterface) error) error {
	return fn(m)
}

func (m *mockRepo) UpsertRateLimitState(ctx context.Context, webhookID uuid.UUID) error {
	args := m.Called(ctx, webhookID)
	return args.Error(0)
}

func (m *mockRepo) DeleteRateLimitState(ctx context.Context, webhookID uuid.UUID) error {
	args := m.Called(ctx, webhookID)
	return args.Error(0)
}

func (m *mockRepo) GetDeliveryByID(ctx context.Context, tenantID uuid.UUID, deliveryID uuid.UUID, consumer string) (*store.WebhookDelivery, error) {
	args := m.Called(ctx, tenantID, deliveryID, consumer)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.(*store.WebhookDelivery), args.Error(1)
}

func (m *mockRepo) GetRetriableDeliveries(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, consumer string, force bool) ([]*store.WebhookDelivery, error) {
	args := m.Called(ctx, tenantID, webhookID, consumer, force)
	return args.Get(0).([]*store.WebhookDelivery), args.Error(1)
}

func (m *mockRepo) ResetDeliveryForRetry(ctx context.Context, deliveryID uuid.UUID) error {
	args := m.Called(ctx, deliveryID)
	return args.Error(0)
}

func (m *mockRepo) ResetDeliveriesForRetry(ctx context.Context, deliveryIDs []uuid.UUID) error {
	args := m.Called(ctx, deliveryIDs)
	return args.Error(0)
}

func (m *mockRepo) GetDeliveriesByWebhookID(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, consumer string, limit, offset int) ([]*store.WebhookDelivery, int, error) {
	args := m.Called(ctx, tenantID, webhookID, consumer, limit, offset)
	return args.Get(0).([]*store.WebhookDelivery), args.Int(1), args.Error(2)
}

func (m *mockRepo) ListDeliveriesPaginated(ctx context.Context, tenantID uuid.UUID, consumer string, limit, offset int) ([]*store.WebhookDelivery, int, error) {
	args := m.Called(ctx, tenantID, consumer, limit, offset)
	return args.Get(0).([]*store.WebhookDelivery), args.Int(1), args.Error(2)
}

func (m *mockRepo) ListDeliveriesFiltered(ctx context.Context, tenantID uuid.UUID, filter store.DeliveryFilter) ([]*store.WebhookDelivery, bool, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]*store.WebhookDelivery), args.Bool(1), args.Error(2)
}

func (m *mockRepo) CountDeliveries(ctx context.Context, tenantID uuid.UUID, filter store.DeliveryFilter) (int, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Int(0), args.Error(1)
}

func (m *mockRepo) ListEventReportsFiltered(ctx context.Context, tenantID uuid.UUID, filter store.EventReportFilter) ([]*store.EventReportWithStats, bool, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]*store.EventReportWithStats), args.Bool(1), args.Error(2)
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

func (m *mockRepo) GetEventByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, consumer, idempotencyKey string) (*store.EventRecord, error) {
	args := m.Called(ctx, tenantID, consumer, idempotencyKey)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.(*store.EventRecord), args.Error(1)
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
	consumer := "default"
	limit := int32(10)
	offset := int32(0)

	expectedWebhooks := []*store.WebhookRegistration{
		{ID: uuid.New(), Consumer: consumer, URL: "http://example.com"},
	}

	repo.On("ListWebhooksPaginated", mock.Anything, mock.Anything, consumer, "", false, store.WebhookHealth(""), int(limit), int(offset)).
		Return(expectedWebhooks, 1, nil)

	webhooks, totalCount, err := service.ListWebhooks(ctx, consumer, "", "", false, "", limit, offset)

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
	consumer := "default"
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
		ID:       webhookID,
		Consumer: consumer,
	}

	repo.On("GetDeliveryByID", mock.Anything, mock.Anything, deliveryID, consumer).Return(delivery, nil)
	repo.On("ResetDeliveriesForRetry", mock.Anything, []uuid.UUID{deliveryID}).Return(nil)
	repo.On("GetWebhookByID", mock.Anything, mock.Anything, webhookID, consumer).Return(webhook, nil).Once()
	inserter.On("BatchInsert", mock.Anything, mock.MatchedBy(func(jobs []river.JobArgs) bool { return len(jobs) == 1 })).
		Return([]*rivertype.JobInsertResult{{}}, nil)

	ids, count, err := service.RetryDelivery(ctx, consumer, deliveryID.String(), "", false)

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
	consumer := "default"

	expectedDeliveries := []*store.WebhookDelivery{
		{ID: uuid.New(), WebhookID: uuid.New(), EventID: uuid.New()},
	}

	// The service normalises limit/offset, so match the filter as built by the service.
	repo.On("ListDeliveriesFiltered", mock.Anything, mock.Anything, mock.MatchedBy(func(f store.DeliveryFilter) bool {
		return f.Consumer == consumer && f.Limit == 20 && f.Offset == 0
	})).Return(expectedDeliveries, true, nil)

	filter := store.DeliveryFilter{
		Consumer: consumer,
		Limit:    20,
		Offset:   0,
	}
	page, err := service.ListDeliveries(ctx, filter)

	assert.NoError(t, err)
	assert.True(t, page.HasMore)
	assert.Equal(t, len(expectedDeliveries), len(page.Items))
	repo.AssertExpectations(t)
}

func TestWebhookService_CreateSubscription(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()
	webhookID := uuid.New().String()
	consumer := "default"
	eventName := "user.created"

	repo.On("LockWebhook", mock.Anything, mock.Anything, mock.Anything, mock.Anything, false).Return(&store.WebhookRegistration{}, nil)
	repo.On("CreateSubscription", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	id, createdAt, err := service.CreateSubscription(ctx, webhookID, eventName, consumer, nil, "POST", 30, false, "", nil, SubscriptionTemplateSettings{})

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
	res, err := service.TestSubscriptionTemplate(ctx, eventName, template, "default", true)
	assert.NoError(t, err)
	assert.Equal(t, `{"new_id": "123"}`, res)
	repo.AssertExpectations(t)
}

func TestWebhookService_PushEvent_AutoRegister(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil, WithAutoRegisterEvents(true))

	ctx := testContext()
	consumer := "default"
	eventName := "user.signup"
	payload := map[string]any{"user_id": "123"}

	// Event does not exist — should be auto-registered
	repo.On("GetEventByName", mock.Anything, mock.Anything, eventName).Return(nil, nil)
	repo.On("RegisterEvent", mock.Anything, mock.Anything, mock.MatchedBy(func(e *store.EventRegistration) bool {
		return e.Name == eventName && e.Active == true
	})).Return(nil)
	repo.On("StoreEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	inserter.On("Insert", mock.Anything, mock.Anything).Return(&rivertype.JobInsertResult{}, nil)

	eventID, _, _, _, err := service.PushEvent(ctx, consumer, eventName, payload, 0, nil, nil, nil)

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
	consumer := "default"
	eventName := "user.created"
	payload := map[string]any{"user_id": "456"}

	// Event already registered and active
	repo.On("GetEventByName", mock.Anything, mock.Anything, eventName).Return(&store.EventRegistration{
		Name:   eventName,
		Active: true,
	}, nil)
	repo.On("StoreEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	inserter.On("Insert", mock.Anything, mock.Anything).Return(&rivertype.JobInsertResult{}, nil)

	eventID, _, _, _, err := service.PushEvent(ctx, consumer, eventName, payload, 0, nil, nil, nil)

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

	_, _, _, _, err := service.PushEvent(ctx, "default", "user.deleted", nil, 0, nil, nil, nil)

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
	consumer := "default"

	repo.On("LockWebhook", mock.Anything, mock.Anything, mock.Anything, mock.Anything, false).Return(&store.WebhookRegistration{}, nil)
	repo.On("CreateSubscription", mock.Anything, mock.Anything, mock.MatchedBy(func(sub *store.EventSubscription) bool {
		return sub.EventName == store.CatchAllEventName
	})).Return(nil)

	id, createdAt, err := service.CreateSubscription(ctx, webhookID, store.CatchAllEventName, consumer, nil, "POST", 30, false, "", nil, SubscriptionTemplateSettings{})

	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.False(t, createdAt.IsZero())
	repo.AssertExpectations(t)
}

func TestWebhookService_UpdateWebhookConfig_MergesSecretHeaderChanges(t *testing.T) {
	repo := new(mockRepo)
	key := bytes.Repeat([]byte{1}, 32)
	cryptoSvc, err := cryptosvc.NewService(key)
	require.NoError(t, err)
	service := NewWebhookService(nil, repo, cryptoSvc)

	ctx := testContext()
	consumer := "default"
	webhookID := uuid.New()
	existingHeaders := map[string]string{
		"Authorization": "Bearer old",
		"X-Trace":       "keep-me",
		"X-Remove":      "drop-me",
	}
	encryptedHeaders, err := service.EncryptSecretHeaders(existingHeaders)
	require.NoError(t, err)

	repo.On("GetWebhookByID", mock.Anything, mock.Anything, webhookID, consumer).Return(&store.WebhookRegistration{
		ID:            webhookID,
		Consumer:      consumer,
		URL:           "https://example.com/webhook",
		Active:        true,
		SecretHeaders: encryptedHeaders,
	}, nil)
	repo.On("UpdateWebhook", mock.Anything, mock.Anything, mock.MatchedBy(func(webhook *store.WebhookRegistration) bool {
		decrypted, err := service.DecryptSecretHeaders(webhook.SecretHeaders)
		require.NoError(t, err)
		return assert.Equal(t, map[string]string{
			"Authorization": "Bearer new",
			"X-Trace":       "keep-me",
		}, decrypted)
	})).Return(nil)
	repo.On("DeleteRateLimitState", mock.Anything, webhookID).Return(nil)

	err = service.UpdateWebhookConfig(ctx, webhookID.String(), consumer, nil, "", nil, true, "", nil, map[string]string{
		"Authorization": "Bearer new",
		"X-Remove":      "",
	}, "", false, []string{"secret_headers"})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestWebhookService_UpdateWebhookConfig_RejectsOutOfBoundsValues(t *testing.T) {
	repo := new(mockRepo)
	service := NewWebhookService(nil, repo, nil)

	ctx := testContext()
	consumer := "default"
	webhookID := uuid.New()

	repo.On("GetWebhookByID", mock.Anything, mock.Anything, webhookID, consumer).Return(&store.WebhookRegistration{
		ID:                    webhookID,
		Consumer:              consumer,
		URL:                   "https://example.com/webhook",
		Active:                true,
		MaxRetries:            3,
		RetryBackoffSeconds:   60,
		RequestTimeoutSeconds: 30,
		ExpectedStatusCodes:   pq.Int64Array{200},
		ContentType:           "application/json",
	}, nil)

	// request_timeout_seconds above the 300s create-path bound must be
	// rejected on update too; UpdateWebhook must never be called.
	err := service.UpdateWebhookConfig(ctx, webhookID.String(), consumer, nil, "", nil, true, "", &HTTPConfigUpdate{
		RequestTimeoutSeconds: 86400,
	}, nil, "", false, []string{"http_config"})

	require.Error(t, err)
	var svcErr *svcerrors.ServiceError
	require.ErrorAs(t, err, &svcErr)
	assert.Equal(t, svcerrors.InvalidArgument, svcErr.Status)
	repo.AssertNotCalled(t, "UpdateWebhook", mock.Anything, mock.Anything, mock.Anything)
}

func TestWebhookService_RequiresTransform(t *testing.T) {
	ctx := testContext()
	const consumer = "default"
	webhookID := uuid.New()
	requiring := &store.WebhookRegistration{ID: webhookID, Consumer: consumer, RequiresTransform: true}

	t.Run("create webhook with events needs a template", func(t *testing.T) {
		service := NewWebhookService(nil, new(mockRepo), nil)
		_, err := service.CreateWebhook(ctx, WebhookRegistrationRequest{
			Consumer: consumer, URL: "https://example.com/hook", Events: []string{"order.created"}, RequiresTransform: true,
		})
		var svcErr *svcerrors.ServiceError
		require.ErrorAs(t, err, &svcErr)
		assert.Equal(t, svcerrors.InvalidArgument, svcErr.Status)
	})

	t.Run("subscription without a transform is refused", func(t *testing.T) {
		repo := new(mockRepo)
		service := NewWebhookService(nil, repo, nil)
		repo.On("LockWebhook", mock.Anything, mock.Anything, webhookID, consumer, false).Return(requiring, nil)

		_, _, err := service.CreateSubscription(ctx, webhookID.String(), "order.created", consumer, nil, "POST", 30, false, "", nil, SubscriptionTemplateSettings{})
		var svcErr *svcerrors.ServiceError
		require.ErrorAs(t, err, &svcErr)
		assert.Equal(t, svcerrors.InvalidArgument, svcErr.Status)
		_, _, err = service.CreateSubscription(ctx, webhookID.String(), "order.created", consumer, nil, "POST", 30, true, "   ", nil, SubscriptionTemplateSettings{})
		require.ErrorAs(t, err, &svcErr)
		repo.AssertNotCalled(t, "CreateSubscription", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("turning it on needs every subscription transformed", func(t *testing.T) {
		repo := new(mockRepo)
		service := NewWebhookService(nil, repo, nil)
		repo.On("GetWebhookByID", mock.Anything, mock.Anything, webhookID, consumer).Return(&store.WebhookRegistration{ID: webhookID, Consumer: consumer}, nil)
		repo.On("LockWebhook", mock.Anything, mock.Anything, webhookID, consumer, true).Return(&store.WebhookRegistration{ID: webhookID, Consumer: consumer}, nil)
		repo.On("ListSubscriptions", mock.Anything, mock.Anything, webhookID).Return([]*store.EventSubscription{
			{EventName: "order.created", TransformEnabled: true, TransformTemplate: "{}"},
			{EventName: "order.refunded"},
		}, nil)

		err := service.UpdateWebhookConfig(ctx, webhookID.String(), consumer, nil, "", nil, false, "", nil, nil, "", true, []string{"requires_transform"})
		var svcErr *svcerrors.ServiceError
		require.ErrorAs(t, err, &svcErr)
		assert.Equal(t, svcerrors.FailedPrecondition, svcErr.Status)
		assert.Contains(t, svcErr.Error(), "order.refunded")
		repo.AssertNotCalled(t, "UpdateWebhook", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("events cannot be replaced in bulk", func(t *testing.T) {
		repo := new(mockRepo)
		service := NewWebhookService(nil, repo, nil)
		repo.On("GetWebhookByID", mock.Anything, mock.Anything, webhookID, consumer).Return(requiring, nil)

		err := service.UpdateWebhookConfig(ctx, webhookID.String(), consumer, []string{"order.created"}, "", nil, false, "", nil, nil, "", false, []string{"events"})
		var svcErr *svcerrors.ServiceError
		require.ErrorAs(t, err, &svcErr)
		assert.Equal(t, svcerrors.FailedPrecondition, svcErr.Status)
	})
}

func TestWebhookService_RotateWebhookSecret(t *testing.T) {
	repo := new(mockRepo)
	cryptoSvc, err := cryptosvc.NewService(bytes.Repeat([]byte{1}, 32))
	require.NoError(t, err)
	service := NewWebhookService(nil, repo, cryptoSvc)

	ctx := testContext()
	consumer := "default"
	webhookID := uuid.New()
	oldSecret, err := service.EncryptWebhookSecret("whsec_old")
	require.NoError(t, err)

	repo.On("GetWebhookByID", mock.Anything, mock.Anything, webhookID, consumer).Return(&store.WebhookRegistration{
		ID:            webhookID,
		Consumer:      consumer,
		URL:           "https://example.com/webhook",
		Active:        true,
		WebhookSecret: oldSecret,
	}, nil)
	var stored []byte
	repo.On("UpdateWebhook", mock.Anything, mock.Anything, mock.MatchedBy(func(webhook *store.WebhookRegistration) bool {
		stored = webhook.WebhookSecret
		return webhook.ID == webhookID && webhook.URL == "https://example.com/webhook"
	})).Return(nil)

	secret, err := service.RotateWebhookSecret(ctx, webhookID.String(), consumer)

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(secret, "whsec_"), "secret %q is not in Standard Webhooks format", secret)
	assert.NotEqual(t, "whsec_old", secret)
	decrypted, err := service.DecryptWebhookSecret(stored)
	require.NoError(t, err)
	assert.Equal(t, secret, decrypted, "the returned secret must be the one stored")
	repo.AssertExpectations(t)
}
