package webhooks

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
	"github.com/sarathsp06/sparrow/pkg/storage"
)

func (m *mockRepo) CreateAlertConfig(ctx context.Context, cfg *store.AlertConfig) error {
	args := m.Called(ctx, cfg)
	return args.Error(0)
}

func newAlertConfigService() (*WebhookService, *mockRepo) {
	repo := &mockRepo{}
	svc := NewWebhookService(&mockJobInserter{}, repo, nil)
	return svc, repo
}

// requireStatus fails the test unless err is a *svcerrors.ServiceError with
// the given Status.
func requireStatus(t *testing.T, err error, want svcerrors.Status) {
	t.Helper()
	var svcErr *svcerrors.ServiceError
	if !errors.As(err, &svcErr) {
		t.Fatalf("expected a *svcerrors.ServiceError, got %v", err)
	}
	if svcErr.Status != want {
		t.Fatalf("expected status %v, got %v (%v)", want, svcErr.Status, svcErr)
	}
}

func TestCreateAlertConfig_RequiresConsumer(t *testing.T) {
	svc, _ := newAlertConfigService()
	_, err := svc.CreateAlertConfig(testContext(), "", "", "a@example.com", []string{"sparrow.webhook.health_changed"})
	requireStatus(t, err, svcerrors.InvalidArgument)
}

func TestCreateAlertConfig_RequiresValidEmail(t *testing.T) {
	svc, _ := newAlertConfigService()
	_, err := svc.CreateAlertConfig(testContext(), "acme", "", "not-an-email", []string{"sparrow.webhook.health_changed"})
	requireStatus(t, err, svcerrors.InvalidArgument)
}

func TestCreateAlertConfig_RequiresNonEmptyEventTypes(t *testing.T) {
	svc, _ := newAlertConfigService()
	_, err := svc.CreateAlertConfig(testContext(), "acme", "", "a@example.com", nil)
	requireStatus(t, err, svcerrors.InvalidArgument)
}

func TestCreateAlertConfig_RejectsUnsupportedEventType(t *testing.T) {
	svc, _ := newAlertConfigService()
	_, err := svc.CreateAlertConfig(testContext(), "acme", "", "a@example.com", []string{"user.created"})
	requireStatus(t, err, svcerrors.InvalidArgument)
}

func TestCreateAlertConfig_ConsumerWide(t *testing.T) {
	svc, repo := newAlertConfigService()
	repo.On("CreateAlertConfig", testContext(), mock.AnythingOfType("*store.AlertConfig")).Return(nil)

	cfg, err := svc.CreateAlertConfig(testContext(), "acme", "", "a@example.com", []string{"sparrow.webhook.health_changed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.WebhookID != nil {
		t.Errorf("expected a nil WebhookID for a consumer-wide config, got %v", cfg.WebhookID)
	}
	repo.AssertExpectations(t)
}

func TestCreateAlertConfig_WebhookNotFound(t *testing.T) {
	svc, repo := newAlertConfigService()
	webhookID := uuid.New()
	repo.On("GetWebhookByID", testContext(), tenant.DefaultTenantID, webhookID, "acme").
		Return(nil, storage.ErrNotFound)

	_, err := svc.CreateAlertConfig(testContext(), "acme", webhookID.String(), "a@example.com", []string{"sparrow.webhook.health_changed"})
	requireStatus(t, err, svcerrors.NotFound)
}
