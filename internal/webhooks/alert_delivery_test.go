package webhooks

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

func (m *mockRepo) ListWebhooks(ctx context.Context, tenantID uuid.UUID, consumer string, event string, activeOnly bool) ([]*store.WebhookRegistration, error) {
	args := m.Called(ctx, tenantID, consumer, event, activeOnly)
	return args.Get(0).([]*store.WebhookRegistration), args.Error(1)
}

func (m *mockRepo) ListSubscriptionsByWebhookIDs(ctx context.Context, tenantID uuid.UUID, webhookIDs []uuid.UUID) ([]*store.EventSubscription, error) {
	args := m.Called(ctx, tenantID, webhookIDs)
	return args.Get(0).([]*store.EventSubscription), args.Error(1)
}

func TestAlertDeliveryConfigured(t *testing.T) {
	paused := time.Now()
	hook := &store.WebhookRegistration{ID: uuid.New(), Consumer: tenant.SystemConsumer, Active: true}
	cases := []struct {
		name  string
		hooks []*store.WebhookRegistration
		subs  []*store.EventSubscription
		want  bool
	}{
		{name: "no _sparrow webhook", want: false},
		{name: "webhook without subscriptions", hooks: []*store.WebhookRegistration{hook}, want: false},
		{name: "subscribed to an alert event", hooks: []*store.WebhookRegistration{hook},
			subs: []*store.EventSubscription{{WebhookID: hook.ID, EventName: "sparrow.webhook.delivery_failed"}}, want: true},
		{name: "catch-all subscription", hooks: []*store.WebhookRegistration{hook},
			subs: []*store.EventSubscription{{WebhookID: hook.ID, EventName: store.CatchAllEventName}}, want: true},
		{name: "only paused subscriptions", hooks: []*store.WebhookRegistration{hook},
			subs: []*store.EventSubscription{{WebhookID: hook.ID, EventName: "sparrow.webhook.disabled", PausedAt: &paused}}, want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := new(mockRepo)
			svc := NewWebhookService(new(mockJobInserter), repo, nil)
			// Only active webhooks of the _sparrow consumer are considered.
			repo.On("ListWebhooks", mock.Anything, mock.Anything, tenant.SystemConsumer, "", true).Return(c.hooks, nil)
			repo.On("ListSubscriptionsByWebhookIDs", mock.Anything, mock.Anything, mock.Anything).Return(c.subs, nil)

			got, err := svc.AlertDeliveryConfigured(testContext())
			assert.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}
