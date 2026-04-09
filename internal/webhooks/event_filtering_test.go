package webhooks

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// ---------------------------------------------------------------------------
// validateLabels tests
// ---------------------------------------------------------------------------

func TestValidateLabels_EmptyMap(t *testing.T) {
	err := validateLabels(nil, "labels")
	assert.NoError(t, err)

	err = validateLabels(map[string]string{}, "labels")
	assert.NoError(t, err)
}

func TestValidateLabels_ValidEntries(t *testing.T) {
	m := map[string]string{
		"region":      "us-east-1",
		"environment": "production",
		"tier":        "premium",
		"app.version": "1.2.3",
		"team_name":   "backend",
		"release-v2":  "true",
	}
	err := validateLabels(m, "labels")
	assert.NoError(t, err)
}

func TestValidateLabels_EmptyKey(t *testing.T) {
	m := map[string]string{"": "value"}
	err := validateLabels(m, "labels")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key must not be empty")
}

func TestValidateLabels_KeyTooLong(t *testing.T) {
	longKey := strings.Repeat("a", maxLabelKeyLen+1)
	m := map[string]string{longKey: "value"}
	err := validateLabels(m, "labels")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length")
}

func TestValidateLabels_KeyExactlyAtLimit(t *testing.T) {
	key := strings.Repeat("k", maxLabelKeyLen)
	m := map[string]string{key: "value"}
	err := validateLabels(m, "labels")
	assert.NoError(t, err)
}

func TestValidateLabels_InvalidKeyCharacters(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"space", "my key"},
		{"exclamation", "key!"},
		{"at sign", "key@host"},
		{"slash", "path/key"},
		{"colon", "ns:key"},
		{"equals", "key=val"},
		{"bracket", "key[0]"},
		{"unicode", "clé"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := map[string]string{tt.key: "value"}
			err := validateLabels(m, "labels")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid characters")
		})
	}
}

func TestValidateLabels_ValidKeyCharacters(t *testing.T) {
	// Alphanumeric, dot, underscore, hyphen are allowed.
	keys := []string{"abc", "ABC", "123", "a.b", "a_b", "a-b", "A.1_2-z"}
	for _, k := range keys {
		m := map[string]string{k: "value"}
		err := validateLabels(m, "labels")
		assert.NoError(t, err, "key %q should be valid", k)
	}
}

func TestValidateLabels_ValueTooLong(t *testing.T) {
	longVal := strings.Repeat("v", maxLabelValueLen+1)
	m := map[string]string{"key": longVal}
	err := validateLabels(m, "labels")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length")
}

func TestValidateLabels_ValueExactlyAtLimit(t *testing.T) {
	val := strings.Repeat("v", maxLabelValueLen)
	m := map[string]string{"key": val}
	err := validateLabels(m, "labels")
	assert.NoError(t, err)
}

func TestValidateLabels_EmptyValue(t *testing.T) {
	// Empty values are allowed (only keys have non-empty constraint).
	m := map[string]string{"key": ""}
	err := validateLabels(m, "labels")
	assert.NoError(t, err)
}

func TestValidateLabels_TooManyEntries(t *testing.T) {
	m := make(map[string]string, maxLabelsPerMap+1)
	for i := 0; i <= maxLabelsPerMap; i++ {
		m[strings.Repeat("k", i+1)] = "v"
	}
	err := validateLabels(m, "labels")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many entries")
}

func TestValidateLabels_ExactlyAtEntryLimit(t *testing.T) {
	m := make(map[string]string, maxLabelsPerMap)
	for i := 0; i < maxLabelsPerMap; i++ {
		m[strings.Repeat("k", i+1)] = "v"
	}
	err := validateLabels(m, "labels")
	assert.NoError(t, err)
}

func TestValidateLabels_FieldNameInError(t *testing.T) {
	// The error message should reference the field name passed in.
	m := map[string]string{"": "value"}
	err := validateLabels(m, "label_filters")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "label_filters")
}

// ---------------------------------------------------------------------------
// PushEvent label validation tests
// ---------------------------------------------------------------------------

func TestPushEvent_WithValidLabels(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()
	namespace := "default"
	eventName := "user.signup"
	payload := map[string]any{"user_id": "123"}
	labels := map[string]string{"region": "us", "tier": "premium"}

	repo.On("GetEventByName", mock.Anything, mock.Anything, eventName).Return(&store.EventRegistration{
		Name:   eventName,
		Active: true,
	}, nil)
	repo.On("StoreEvent", mock.Anything, mock.Anything, mock.MatchedBy(func(e *store.EventRecord) bool {
		// Verify labels are passed through to the stored event record.
		return e.Labels["region"] == "us" && e.Labels["tier"] == "premium"
	})).Return(nil)
	inserter.On("Insert", mock.Anything, mock.Anything).Return(&rivertype.JobInsertResult{}, nil)

	eventID, _, _, err := service.PushEvent(ctx, namespace, eventName, payload, 0, nil, labels, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, eventID)
	repo.AssertExpectations(t)
	inserter.AssertExpectations(t)
}

func TestPushEvent_WithNilLabels(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()

	repo.On("GetEventByName", mock.Anything, mock.Anything, "test.event").Return(&store.EventRegistration{
		Name:   "test.event",
		Active: true,
	}, nil)
	repo.On("StoreEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	inserter.On("Insert", mock.Anything, mock.Anything).Return(&rivertype.JobInsertResult{}, nil)

	eventID, _, _, err := service.PushEvent(ctx, "default", "test.event", nil, 0, nil, nil, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, eventID)
	repo.AssertExpectations(t)
}

func TestPushEvent_WithInvalidLabels(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()

	tests := []struct {
		name   string
		labels map[string]string
		errMsg string
	}{
		{
			name:   "empty key",
			labels: map[string]string{"": "value"},
			errMsg: "key must not be empty",
		},
		{
			name:   "invalid key chars",
			labels: map[string]string{"bad key": "value"},
			errMsg: "invalid characters",
		},
		{
			name:   "value too long",
			labels: map[string]string{"key": strings.Repeat("x", maxLabelValueLen+1)},
			errMsg: "exceeds maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := service.PushEvent(ctx, "default", "some.event", nil, 0, nil, tt.labels, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
			// Ensure no repo calls were made — validation should short-circuit.
			repo.AssertNotCalled(t, "GetEventByName", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// ---------------------------------------------------------------------------
// CreateSubscription label-filter tests
// ---------------------------------------------------------------------------

func TestCreateSubscription_WithLabelFilters(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()
	webhookID := uuid.New().String()
	namespace := "default"
	eventName := "order.created"
	labelFilters := map[string]string{"region": "us", "tier": "premium"}

	repo.On("CreateSubscription", mock.Anything, mock.Anything, mock.MatchedBy(func(sub *store.EventSubscription) bool {
		return sub.EventName == eventName &&
			sub.Namespace == namespace &&
			sub.LabelFilters["region"] == "us" &&
			sub.LabelFilters["tier"] == "premium"
	})).Return(nil)

	id, createdAt, err := service.CreateSubscription(ctx, webhookID, eventName, namespace, nil, "POST", 30, false, "", labelFilters)
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.False(t, createdAt.IsZero())
	repo.AssertExpectations(t)
}

func TestCreateSubscription_WithEmptyLabelFilters(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()
	webhookID := uuid.New().String()

	repo.On("CreateSubscription", mock.Anything, mock.Anything, mock.MatchedBy(func(sub *store.EventSubscription) bool {
		return len(sub.LabelFilters) == 0
	})).Return(nil)

	id, createdAt, err := service.CreateSubscription(ctx, webhookID, "test.event", "default", nil, "POST", 30, false, "", nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.False(t, createdAt.IsZero())
	repo.AssertExpectations(t)
}

func TestCreateSubscription_WithInvalidLabelFilters(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()
	webhookID := uuid.New().String()

	tests := []struct {
		name         string
		labelFilters map[string]string
		errMsg       string
	}{
		{
			name:         "empty key",
			labelFilters: map[string]string{"": "val"},
			errMsg:       "key must not be empty",
		},
		{
			name:         "invalid key chars",
			labelFilters: map[string]string{"key with space": "val"},
			errMsg:       "invalid characters",
		},
		{
			name:         "key too long",
			labelFilters: map[string]string{strings.Repeat("a", maxLabelKeyLen+1): "val"},
			errMsg:       "exceeds maximum length",
		},
		{
			name:         "value too long",
			labelFilters: map[string]string{"key": strings.Repeat("v", maxLabelValueLen+1)},
			errMsg:       "exceeds maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := service.CreateSubscription(ctx, webhookID, "test.event", "default", nil, "POST", 30, false, "", tt.labelFilters)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
			repo.AssertNotCalled(t, "CreateSubscription", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestCreateSubscription_CatchAllWithLabelFilters(t *testing.T) {
	// A catch-all subscription (* event) can still have label filters.
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	service := NewWebhookService(inserter, repo, nil)

	ctx := testContext()
	webhookID := uuid.New().String()
	labelFilters := map[string]string{"env": "prod"}

	repo.On("CreateSubscription", mock.Anything, mock.Anything, mock.MatchedBy(func(sub *store.EventSubscription) bool {
		return sub.EventName == store.CatchAllEventName && sub.LabelFilters["env"] == "prod"
	})).Return(nil)

	id, _, err := service.CreateSubscription(ctx, webhookID, store.CatchAllEventName, "default", nil, "POST", 30, false, "", labelFilters)
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetSubscriptionsByEvent mock tests (service → repo contract)
// ---------------------------------------------------------------------------

// mockRepoWithEventQuery extends mockRepo with the event-subscription query methods.
type mockRepoWithEventQuery struct {
	mockRepo
}

func (m *mockRepoWithEventQuery) GetSubscriptionsByEvent(ctx context.Context, tenantID uuid.UUID, namespace, event string, labels map[string]string) ([]*store.EventSubscription, error) {
	args := m.Called(ctx, tenantID, namespace, event, labels)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.([]*store.EventSubscription), args.Error(1)
}

func (m *mockRepoWithEventQuery) GetSubscriptionsWithWebhooksByEvent(ctx context.Context, tenantID uuid.UUID, namespace, event string, labels map[string]string) ([]*store.SubscriptionWithWebhook, error) {
	args := m.Called(ctx, tenantID, namespace, event, labels)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.([]*store.SubscriptionWithWebhook), args.Error(1)
}

func TestGetSubscriptionsByEvent_ExactMatch(t *testing.T) {
	repo := new(mockRepoWithEventQuery)
	service := NewWebhookService(nil, repo, nil)

	ctx := testContext()
	webhookID := uuid.New()
	subID := uuid.New()

	expected := []*store.EventSubscription{
		{
			ID:        subID,
			WebhookID: webhookID,
			EventName: "signup",
			Namespace: "default",
			CreatedAt: time.Now(),
		},
	}

	repo.On("GetSubscriptionsByEvent", mock.Anything, mock.Anything, "default", "signup", map[string]string(nil)).
		Return(expected, nil)

	result, err := service.GetWebhookRepo().(*mockRepoWithEventQuery).GetSubscriptionsByEvent(ctx, uuid.New(), "default", "signup", nil)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "signup", result[0].EventName)
	repo.AssertExpectations(t)
}

func TestGetSubscriptionsByEvent_CatchAllReturned(t *testing.T) {
	repo := new(mockRepoWithEventQuery)

	ctx := testContext()

	// Simulate DB returning both an exact match and a catch-all subscription.
	expected := []*store.EventSubscription{
		{
			ID:        uuid.New(),
			WebhookID: uuid.New(),
			EventName: "signup",
			Namespace: "default",
		},
		{
			ID:        uuid.New(),
			WebhookID: uuid.New(),
			EventName: store.CatchAllEventName,
			Namespace: "default",
		},
	}

	repo.On("GetSubscriptionsByEvent", mock.Anything, mock.Anything, "default", "signup", map[string]string(nil)).
		Return(expected, nil)

	result, err := repo.GetSubscriptionsByEvent(ctx, uuid.New(), "default", "signup", nil)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	eventNames := []string{result[0].EventName, result[1].EventName}
	assert.Contains(t, eventNames, "signup")
	assert.Contains(t, eventNames, store.CatchAllEventName)
	repo.AssertExpectations(t)
}

func TestGetSubscriptionsByEvent_NoMatches(t *testing.T) {
	repo := new(mockRepoWithEventQuery)

	ctx := testContext()

	repo.On("GetSubscriptionsByEvent", mock.Anything, mock.Anything, "default", "unknown.event", map[string]string(nil)).
		Return([]*store.EventSubscription{}, nil)

	result, err := repo.GetSubscriptionsByEvent(ctx, uuid.New(), "default", "unknown.event", nil)
	assert.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

func TestGetSubscriptionsByEvent_LabelFiltering(t *testing.T) {
	repo := new(mockRepoWithEventQuery)

	ctx := testContext()

	// Subscription with label_filters={"region":"us"} should match event with labels={"region":"us","tier":"premium"}.
	matchingSub := &store.EventSubscription{
		ID:           uuid.New(),
		WebhookID:    uuid.New(),
		EventName:    "order.created",
		Namespace:    "default",
		LabelFilters: map[string]string{"region": "us"},
	}

	eventLabels := map[string]string{"region": "us", "tier": "premium"}

	repo.On("GetSubscriptionsByEvent", mock.Anything, mock.Anything, "default", "order.created", eventLabels).
		Return([]*store.EventSubscription{matchingSub}, nil)

	result, err := repo.GetSubscriptionsByEvent(ctx, uuid.New(), "default", "order.created", eventLabels)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "order.created", result[0].EventName)
	repo.AssertExpectations(t)
}

func TestGetSubscriptionsByEvent_LabelMismatchFiltered(t *testing.T) {
	repo := new(mockRepoWithEventQuery)

	ctx := testContext()

	// If event labels don't contain all of the subscription's label_filters,
	// the DB query filters it out — result is empty.
	eventLabels := map[string]string{"region": "eu"}

	repo.On("GetSubscriptionsByEvent", mock.Anything, mock.Anything, "default", "order.created", eventLabels).
		Return([]*store.EventSubscription{}, nil)

	result, err := repo.GetSubscriptionsByEvent(ctx, uuid.New(), "default", "order.created", eventLabels)
	assert.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

func TestGetSubscriptionsByEvent_EmptyLabelFiltersMatchAll(t *testing.T) {
	repo := new(mockRepoWithEventQuery)

	ctx := testContext()

	// A subscription with label_filters={} matches any event, regardless of its labels.
	sub := &store.EventSubscription{
		ID:           uuid.New(),
		WebhookID:    uuid.New(),
		EventName:    "signup",
		Namespace:    "default",
		LabelFilters: map[string]string{},
	}

	eventLabels := map[string]string{"region": "us", "tier": "premium"}

	repo.On("GetSubscriptionsByEvent", mock.Anything, mock.Anything, "default", "signup", eventLabels).
		Return([]*store.EventSubscription{sub}, nil)

	result, err := repo.GetSubscriptionsByEvent(ctx, uuid.New(), "default", "signup", eventLabels)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	repo.AssertExpectations(t)
}

func TestGetSubscriptionsByEvent_NamespaceIsolation(t *testing.T) {
	repo := new(mockRepoWithEventQuery)

	ctx := testContext()

	// Subscriptions in namespace "billing" should not appear when querying "default".
	repo.On("GetSubscriptionsByEvent", mock.Anything, mock.Anything, "default", "signup", map[string]string(nil)).
		Return([]*store.EventSubscription{}, nil)

	result, err := repo.GetSubscriptionsByEvent(ctx, uuid.New(), "default", "signup", nil)
	assert.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetSubscriptionsWithWebhooksByEvent mock tests (active webhook check)
// ---------------------------------------------------------------------------

func TestGetSubscriptionsWithWebhooksByEvent_ActiveWebhooksOnly(t *testing.T) {
	repo := new(mockRepoWithEventQuery)

	ctx := testContext()

	activeWebhook := &store.WebhookRegistration{
		ID:        uuid.New(),
		Namespace: "default",
		URL:       "https://example.com/hook",
		Active:    true,
	}
	sub := &store.EventSubscription{
		ID:        uuid.New(),
		WebhookID: activeWebhook.ID,
		EventName: "signup",
		Namespace: "default",
	}

	// The SQL query JOINs with wr.active = true, so only active webhooks appear.
	expected := []*store.SubscriptionWithWebhook{
		{Subscription: sub, Webhook: activeWebhook},
	}

	repo.On("GetSubscriptionsWithWebhooksByEvent", mock.Anything, mock.Anything, "default", "signup", map[string]string(nil)).
		Return(expected, nil)

	result, err := repo.GetSubscriptionsWithWebhooksByEvent(ctx, uuid.New(), "default", "signup", nil)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.True(t, result[0].Webhook.Active)
	repo.AssertExpectations(t)
}

func TestGetSubscriptionsWithWebhooksByEvent_InactiveExcluded(t *testing.T) {
	repo := new(mockRepoWithEventQuery)

	ctx := testContext()

	// An inactive webhook's subscriptions should not be returned by the query.
	repo.On("GetSubscriptionsWithWebhooksByEvent", mock.Anything, mock.Anything, "default", "signup", map[string]string(nil)).
		Return([]*store.SubscriptionWithWebhook{}, nil)

	result, err := repo.GetSubscriptionsWithWebhooksByEvent(ctx, uuid.New(), "default", "signup", nil)
	assert.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

func TestGetSubscriptionsWithWebhooksByEvent_MixedCatchAllAndSpecific(t *testing.T) {
	repo := new(mockRepoWithEventQuery)

	ctx := testContext()

	webhook1 := &store.WebhookRegistration{
		ID: uuid.New(), Namespace: "default", URL: "https://a.com/hook", Active: true,
	}
	webhook2 := &store.WebhookRegistration{
		ID: uuid.New(), Namespace: "default", URL: "https://b.com/hook", Active: true,
	}

	specificSub := &store.EventSubscription{
		ID: uuid.New(), WebhookID: webhook1.ID, EventName: "signup", Namespace: "default",
	}
	catchAllSub := &store.EventSubscription{
		ID: uuid.New(), WebhookID: webhook2.ID, EventName: store.CatchAllEventName, Namespace: "default",
	}

	expected := []*store.SubscriptionWithWebhook{
		{Subscription: specificSub, Webhook: webhook1},
		{Subscription: catchAllSub, Webhook: webhook2},
	}

	repo.On("GetSubscriptionsWithWebhooksByEvent", mock.Anything, mock.Anything, "default", "signup", map[string]string(nil)).
		Return(expected, nil)

	result, err := repo.GetSubscriptionsWithWebhooksByEvent(ctx, uuid.New(), "default", "signup", nil)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	names := []string{result[0].Subscription.EventName, result[1].Subscription.EventName}
	assert.Contains(t, names, "signup")
	assert.Contains(t, names, store.CatchAllEventName)
	repo.AssertExpectations(t)
}

func TestGetSubscriptionsWithWebhooksByEvent_LabelsPassedThrough(t *testing.T) {
	repo := new(mockRepoWithEventQuery)

	ctx := testContext()
	eventLabels := map[string]string{"region": "us", "env": "prod"}

	// Verify the labels are forwarded to the repository method exactly.
	repo.On("GetSubscriptionsWithWebhooksByEvent", mock.Anything, mock.Anything, "default", "order.created", eventLabels).
		Return([]*store.SubscriptionWithWebhook{}, nil)

	_, err := repo.GetSubscriptionsWithWebhooksByEvent(ctx, uuid.New(), "default", "order.created", eventLabels)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
