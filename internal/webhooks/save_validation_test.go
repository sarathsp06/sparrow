package webhooks

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	svcerrors "github.com/sarathsp06/sparrow/internal/errors"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

const unparsableTemplate = `{"id": {{.payload.id`

func TestValidateTransformTemplate(t *testing.T) {
	for _, ok := range []string{"", "   ", `{"id": {{ json .payload.id }}}`, `{{ .payload.missing_field }}`} {
		assert.NoError(t, validateTransformTemplate(ok), ok)
	}
	for _, bad := range []string{unparsableTemplate, `{{ nosuchhelper .payload }}`, `{{ range 2000000000 }}{{ end }}`} {
		err := validateTransformTemplate(bad)
		assertStatus(t, err, svcerrors.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid transform_template", bad)
	}
}

// A template that cannot parse would fail every delivery, so saving one is
// refused before anything is stored.
func TestSubscriptionSavesRejectUnparsableTemplates(t *testing.T) {
	ctx := testContext()

	t.Run("create", func(t *testing.T) {
		repo := new(mockRepo)
		svc := NewWebhookService(nil, repo, nil)
		_, _, err := svc.CreateSubscription(ctx, uuid.NewString(), "order.created", "default", nil, "", 0, true, unparsableTemplate, nil, SubscriptionTemplateSettings{})
		assertStatus(t, err, svcerrors.InvalidArgument)
		assert.Contains(t, err.Error(), "unclosed action")
		repo.AssertNotCalled(t, "LockWebhook", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("update", func(t *testing.T) {
		repo := new(mockRepo)
		svc := NewWebhookService(nil, repo, nil)
		err := svc.UpdateSubscription(ctx, uuid.NewString(), "default", nil, "", 0, true, unparsableTemplate, nil, SubscriptionTemplateSettings{}, TemplateSaveMeta{})
		assertStatus(t, err, svcerrors.InvalidArgument)
		repo.AssertNotCalled(t, "GetSubscription", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("webhook registration with events", func(t *testing.T) {
		repo := new(mockRepo)
		svc := NewWebhookService(nil, repo, nil)
		_, _, err := svc.CreateWebhook(ctx, WebhookRegistrationRequest{
			Consumer:          "default",
			URL:               "https://example.com/hook",
			Events:            []string{"order.created"},
			TransformTemplate: unparsableTemplate,
		})
		assertStatus(t, err, svcerrors.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid transform_template")
		repo.AssertNotCalled(t, "RegisterWebhookWithSubscriptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestValidateNewEventTypeName(t *testing.T) {
	for _, ok := range []string{"order.created", "stripe.payment_intent.succeeded", "github.pull_request.opened", "user-signed-up", "invoice.sent.v2", "a"} {
		assert.NoError(t, validateNewEventTypeName(ok), ok)
	}
	for _, bad := range []string{"Bad Name!", "Order.Created", "order created", ".order", "order.", "order..created", "*", "orders/create", "commande.créée"} {
		err := validateNewEventTypeName(bad)
		assertStatus(t, err, svcerrors.InvalidArgument)
		assert.Contains(t, err.Error(), "order.created", bad)
	}
}

func TestPushEvent_AutoRegisterRejectsInvalidNames(t *testing.T) {
	repo := newFakeEventTypeRepo()
	svc := NewWebhookService(new(mockJobInserter), repo, nil, WithAutoRegisterEvents(true))

	_, _, _, _, err := svc.PushEvent(testContext(), "default", "Bad Name!", map[string]any{}, 0, nil, nil, nil)
	assertStatus(t, err, svcerrors.InvalidArgument)
	assert.Empty(t, repo.heads, "no event type is registered")
}

func TestSaveEventType_NameRuleOnlyAppliesToNewTypes(t *testing.T) {
	ctx := testContext()
	repo := newFakeEventTypeRepo()
	svc := NewWebhookService(nil, repo, nil)

	_, _, err := svc.RegisterEvent(ctx, "Bad Name!", "", nil, nil, true)
	assertStatus(t, err, svcerrors.InvalidArgument)
	assert.Empty(t, repo.heads)

	// A type saved before the rule can still be updated.
	repo.heads["LegacyName"] = &store.EventRegistration{Name: "LegacyName", Active: true, Version: 1}
	res, err := svc.saveEventType(ctx, EventTypeDefinition{Name: "LegacyName", Description: "still here", Active: true}, saveMustExist, saveEventTypeOptions{})
	require.NoError(t, err)
	assert.Equal(t, "still here", res.Event.Description)
}
