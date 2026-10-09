package webhooks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
	"github.com/sarathsp06/sparrow/pkg/storage"
	"github.com/sarathsp06/sparrow/pkg/template"
)

// getSubscriptionInConsumer loads a subscription by ID and verifies it belongs to the
// given consumer. Returns svcerrors.NotFoundError if the subscription is not in the consumer.
func (s *WebhookService) getSubscriptionInConsumer(ctx context.Context, subscriptionID string, consumer string) (*store.EventSubscription, error) {
	tenantID := tenant.DefaultTenantID

	if consumer == "" {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "consumer is required")
	}

	id, err := parseUUID(subscriptionID, "subscription ID")
	if err != nil {
		return nil, err
	}

	sub, err := s.webhookRepo.GetSubscription(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if sub.Consumer != consumer {
		return nil, svcerrors.Error(svcerrors.NotFound, "subscription not found in consumer")
	}
	return sub, nil
}

// ListSubscriptionsByWebhookIDs batch-fetches subscriptions for multiple
// webhooks in a single query, scoped to the default tenant.
func (s *WebhookService) ListSubscriptionsByWebhookIDs(ctx context.Context, webhookIDs []uuid.UUID) ([]*store.EventSubscription, error) {
	return s.webhookRepo.ListSubscriptionsByWebhookIDs(ctx, tenant.DefaultTenantID, webhookIDs)
}

// SubscriptionTemplateSettings controls how a subscription's transform
// template is rendered and what happens when it fails. Empty fields take the
// defaults: on_transform_error=fail, template_missing_key=error.
type SubscriptionTemplateSettings struct {
	OnTransformError   string
	TemplateMissingKey string
}

// validate rejects unknown values; empty means "use the default".
func (t SubscriptionTemplateSettings) validate() error {
	switch t.OnTransformError {
	case "", store.OnTransformErrorFail, store.OnTransformErrorFallback:
	default:
		return svcerrors.Errorf(svcerrors.InvalidArgument, "on_transform_error must be %q or %q", store.OnTransformErrorFail, store.OnTransformErrorFallback)
	}
	switch t.TemplateMissingKey {
	case "", store.TemplateMissingKeyError, store.TemplateMissingKeyZero:
	default:
		return svcerrors.Errorf(svcerrors.InvalidArgument, "template_missing_key must be %q or %q", store.TemplateMissingKeyError, store.TemplateMissingKeyZero)
	}
	return nil
}

// checkRequiredTransform enforces a webhook's requires_transform on the
// transform settings a subscription is about to be saved with.
func checkRequiredTransform(webhook *store.WebhookRegistration, eventName string, transformEnabled bool, transformTemplate string) error {
	if !webhook.RequiresTransform || (transformEnabled && strings.TrimSpace(transformTemplate) != "") {
		return nil
	}
	return svcerrors.Errorf(svcerrors.InvalidArgument,
		"this webhook requires a payload transform: the %q subscription needs transform_enabled=true and a transform_template", eventName)
}

// Subscription Management Implementation

func (s *WebhookService) CreateSubscription(ctx context.Context, webhookID, eventName, consumer string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string, labelFilters map[string]string, settings SubscriptionTemplateSettings) (string, time.Time, error) {
	s.logger.InfoContext(ctx, "Creating subscription", "webhook_id", webhookID, "event_name", eventName, "consumer", consumer)

	if consumer == "" {
		return "", time.Time{}, svcerrors.Error(svcerrors.InvalidArgument, "consumer is required")
	}
	if err := checkEventForConsumer(consumer, eventName); err != nil {
		return "", time.Time{}, err
	}
	if err := validateHeaders("headers", headers); err != nil {
		return "", time.Time{}, err
	}

	tenantID := tenant.DefaultTenantID

	id, err := parseUUID(webhookID, "webhook ID")
	if err != nil {
		return "", time.Time{}, err
	}

	if err := validateLabels(labelFilters, "label_filters"); err != nil {
		return "", time.Time{}, err
	}
	if err := settings.validate(); err != nil {
		return "", time.Time{}, err
	}

	sub := &store.EventSubscription{
		OnTransformError:   settings.OnTransformError,
		TemplateMissingKey: settings.TemplateMissingKey,
		WebhookID:          id,
		EventName:          eventName,
		Consumer:           consumer,
		Headers:            headers,
		Method:             method,
		Timeout:            timeout,
		TransformEnabled:   transformEnabled,
		TransformTemplate:  transformTemplate,
		LabelFilters:       labelFilters,
	}

	// The shared lock on the webhook keeps requires_transform from changing
	// between the check and the insert.
	err = s.webhookRepo.RunInTransaction(func(repo store.RepositoryInterface) error {
		webhook, err := repo.LockWebhook(ctx, tenantID, id, consumer, false)
		if err != nil {
			if storage.IsNotFound(err) {
				return svcerrors.Error(svcerrors.NotFound, "webhook not found in consumer")
			}
			return fmt.Errorf("failed to get webhook: %w", err)
		}
		if err := checkRequiredTransform(webhook, eventName, transformEnabled, transformTemplate); err != nil {
			return err
		}
		if err := repo.CreateSubscription(ctx, tenantID, sub); err != nil {
			return err
		}
		if strings.TrimSpace(transformTemplate) == "" {
			return nil
		}
		// First version of a template created with the subscription.
		return repo.InsertTemplateVersion(ctx, tenantID, &store.SubscriptionTemplateVersion{SubscriptionID: sub.ID, Template: transformTemplate, Source: "manual"})
	})
	if err != nil {
		return "", time.Time{}, err
	}
	return sub.ID.String(), sub.CreatedAt, nil
}

func (s *WebhookService) GetSubscription(ctx context.Context, subscriptionID string, consumer string) (*store.EventSubscription, error) {
	return s.getSubscriptionInConsumer(ctx, subscriptionID, consumer)
}

// ListSubscriptions lists subscriptions, optionally filtered by consumer,
// webhook, or event name. An empty consumer lists across all consumers.
func (s *WebhookService) ListSubscriptions(ctx context.Context, consumer string, webhookID string, eventName string, limit, offset int32) ([]*store.EventSubscription, int32, error) {

	tenantID := tenant.DefaultTenantID

	l, o := normalizePagination(int(limit), int(offset))
	limit, offset = int32(l), int32(o)

	var subs []*store.EventSubscription
	var totalCount int
	var err error

	if webhookID != "" {
		var id uuid.UUID
		id, err = parseUUID(webhookID, "webhook ID")
		if err != nil {
			return nil, 0, err
		}
		// Verify the webhook belongs to the requested consumer before listing.
		// With no consumer given, the id alone (globally unique) is the scope.
		if consumer != "" {
			if _, err = s.webhookRepo.GetWebhookByID(ctx, tenantID, id, consumer); err != nil {
				if storage.IsNotFound(err) {
					return nil, 0, svcerrors.Error(svcerrors.NotFound, "webhook not found in consumer")
				}
				return nil, 0, fmt.Errorf("failed to get webhook: %w", err)
			}
		}
		subs, err = s.webhookRepo.ListSubscriptions(ctx, tenantID, id)
		totalCount = len(subs)
		// ponytail: repo lists all rows, paginate in memory; push LIMIT/OFFSET into SQL if per-webhook subscription counts grow large
		subs = paginateSubscriptions(subs, offset, limit)
	} else if eventName != "" {
		subs, err = s.webhookRepo.ListSubscriptionsByEvent(ctx, tenantID, consumer, eventName)
		totalCount = len(subs)
		subs = paginateSubscriptions(subs, offset, limit)
	} else {
		// List all subscriptions, optionally scoped to one consumer
		subs, totalCount, err = s.webhookRepo.ListSubscriptionsByConsumer(ctx, tenantID, consumer, int(limit), int(offset))
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list subscriptions: %w", err)
		}
	}

	return subs, int32(totalCount), err
}

// paginateSubscriptions applies offset/limit to an in-memory slice, clamping to bounds.
func paginateSubscriptions(subs []*store.EventSubscription, offset, limit int32) []*store.EventSubscription {
	start := int(offset)
	if start >= len(subs) {
		return []*store.EventSubscription{}
	}
	end := start + int(limit)
	if end > len(subs) {
		end = len(subs)
	}
	return subs[start:end]
}

// TemplateSaveMeta describes how a saved template came to be, for the
// subscription's template history.
type TemplateSaveMeta struct {
	Source  string // "manual" (default) or "ai_draft"
	Notes   string // the drafter's notes, when Source is ai_draft
	SavedBy string // credential name when auth is on
}

func (s *WebhookService) UpdateSubscription(ctx context.Context, subscriptionID string, consumer string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string, labelFilters map[string]string, settings SubscriptionTemplateSettings, meta TemplateSaveMeta) error {
	if err := validateLabels(labelFilters, "label_filters"); err != nil {
		return err
	}
	if err := settings.validate(); err != nil {
		return err
	}
	if err := validateHeaders("headers", headers); err != nil {
		return err
	}

	sub, err := s.getSubscriptionInConsumer(ctx, subscriptionID, consumer)
	if err != nil {
		return err
	}
	templateChanged := strings.TrimSpace(transformTemplate) != "" && sub.TransformTemplate != transformTemplate

	sub.Headers = headers
	sub.Method = method
	sub.Timeout = timeout
	sub.TransformEnabled = transformEnabled
	sub.TransformTemplate = transformTemplate
	sub.LabelFilters = labelFilters
	sub.OnTransformError = settings.OnTransformError
	sub.TemplateMissingKey = settings.TemplateMissingKey

	// The shared lock on the webhook keeps requires_transform from changing
	// between the check and the update.
	return s.webhookRepo.RunInTransaction(func(repo store.RepositoryInterface) error {
		webhook, err := repo.LockWebhook(ctx, tenant.DefaultTenantID, sub.WebhookID, consumer, false)
		if err != nil {
			return fmt.Errorf("failed to get webhook: %w", err)
		}
		if err := checkRequiredTransform(webhook, sub.EventName, transformEnabled, transformTemplate); err != nil {
			return err
		}
		if err := repo.UpdateSubscription(ctx, tenant.DefaultTenantID, sub); err != nil {
			return err
		}
		if !templateChanged {
			return nil
		}
		// A changed template is saved together with its history row.
		return repo.InsertTemplateVersion(ctx, tenant.DefaultTenantID, &store.SubscriptionTemplateVersion{
			SubscriptionID: sub.ID,
			Template:       transformTemplate,
			Source:         meta.Source,
			Notes:          meta.Notes,
			SavedBy:        meta.SavedBy,
		})
	})
}

// ListSubscriptionTemplateVersions returns the saved template history,
// newest first. The subscription must belong to consumer.
func (s *WebhookService) ListSubscriptionTemplateVersions(ctx context.Context, subscriptionID string, consumer string, limit int) ([]*store.SubscriptionTemplateVersion, error) {
	sub, err := s.getSubscriptionInConsumer(ctx, subscriptionID, consumer)
	if err != nil {
		return nil, err
	}
	return s.webhookRepo.ListTemplateVersions(ctx, tenant.DefaultTenantID, sub.ID, limit)
}

// RenderTemplatePreview renders a transform template against payload as a
// dry-run delivery of eventName (attempt 1, a placeholder event id, now as
// the timestamp). strict makes a reference to a key the payload lacks an
// error naming that key instead of rendering "<no value>". AI draft
// verification uses it with strict=true.
func RenderTemplatePreview(eventName, transformTemplate string, payload map[string]any, strict bool) (string, error) {
	engine := template.NewTemplateEngine()
	data := template.NewWebhookTemplateContext(
		"dry-run-event-id",
		eventName,
		time.Now().UTC().Format(time.RFC3339),
		1,
		payload,
	)
	result, err := engine.TransformPayloadWith(transformTemplate, data, template.ExecOptions{StrictMissingKeys: strict})
	if err != nil {
		return "", svcerrors.Wrapf(err, svcerrors.InvalidArgument, "template transformation failed: %v", err)
	}
	return string(result), nil
}

// maxPauseReasonLength bounds the free-text reason stored with a pause.
const maxPauseReasonLength = 500

// PauseSubscription stops a subscription's deliveries from being attempted.
// Events keep fanning out to it, but each delivery is recorded with status
// paused and nothing is queued, so nothing is lost and nothing counts
// against the webhook's health. Pausing an already paused subscription
// updates the reason and keeps the original pause time.
func (s *WebhookService) PauseSubscription(ctx context.Context, subscriptionID, consumer, reason string) (*store.EventSubscription, error) {
	if len([]rune(reason)) > maxPauseReasonLength {
		return nil, svcerrors.Errorf(svcerrors.InvalidArgument, "reason must be at most %d characters", maxPauseReasonLength)
	}
	sub, err := s.getSubscriptionInConsumer(ctx, subscriptionID, consumer)
	if err != nil {
		return nil, err
	}
	if _, err := s.webhookRepo.SetSubscriptionPaused(ctx, tenant.DefaultTenantID, sub.ID, true, reason); err != nil {
		return nil, fmt.Errorf("failed to pause subscription: %w", err)
	}
	s.logger.InfoContext(ctx, "Subscription paused", "subscription_id", sub.ID, "reason", reason)
	return s.webhookRepo.GetSubscription(ctx, tenant.DefaultTenantID, sub.ID)
}

// ResumeResult describes a resumed subscription.
type ResumeResult struct {
	Subscription *store.EventSubscription
	// PausedSince is when the pause began; nil if it was not paused.
	PausedSince *time.Time
	// PausedDeliveries counts the deliveries recorded while paused. They stay
	// paused until retried, so the caller can decide what to replay.
	PausedDeliveries int
}

// ResumeSubscription lets a subscription's new deliveries be attempted again.
// Deliveries recorded while it was paused are not sent automatically; retry
// them (all, or since a time) to deliver them.
func (s *WebhookService) ResumeSubscription(ctx context.Context, subscriptionID, consumer string) (*ResumeResult, error) {
	sub, err := s.getSubscriptionInConsumer(ctx, subscriptionID, consumer)
	if err != nil {
		return nil, err
	}
	pausedSince, err := s.webhookRepo.SetSubscriptionPaused(ctx, tenant.DefaultTenantID, sub.ID, false, "")
	if err != nil {
		return nil, fmt.Errorf("failed to resume subscription: %w", err)
	}
	status := string(store.StatusPaused)
	count, err := s.webhookRepo.CountDeliveries(ctx, tenant.DefaultTenantID, store.DeliveryFilter{
		Consumer:       consumer,
		SubscriptionID: &sub.ID,
		Status:         &status,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count paused deliveries: %w", err)
	}
	updated, err := s.webhookRepo.GetSubscription(ctx, tenant.DefaultTenantID, sub.ID)
	if err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "Subscription resumed", "subscription_id", sub.ID, "paused_deliveries", count)
	return &ResumeResult{Subscription: updated, PausedSince: pausedSince, PausedDeliveries: count}, nil
}

func (s *WebhookService) DeleteSubscription(ctx context.Context, subscriptionID string, consumer string) error {
	sub, err := s.getSubscriptionInConsumer(ctx, subscriptionID, consumer)
	if err != nil {
		return err
	}
	return s.webhookRepo.DeleteSubscription(ctx, tenant.DefaultTenantID, sub.ID)
}

// TestSubscriptionTemplate renders transformTemplate against the event type's
// sample payload. strict renders missing keys as errors, as a subscription
// with template_missing_key=error does at delivery time.
func (s *WebhookService) TestSubscriptionTemplate(ctx context.Context, eventName, transformTemplate, consumer string, strict bool) (string, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.TestSubscriptionTemplate")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing test subscription template request", "event_name", eventName, "consumer", consumer)

	if eventName == "" {
		return "", svcerrors.Error(svcerrors.InvalidArgument, "event name is required")
	}

	tenantID := tenant.DefaultTenantID

	event, err := s.webhookRepo.GetEventByName(ctx, tenantID, eventName)
	if err != nil {
		return "", fmt.Errorf("failed to get event: %w", err)
	}
	if event == nil {
		return "", svcerrors.Error(svcerrors.NotFound, "event not found")
	}

	engine := template.NewTemplateEngine()

	// Create context for template
	data := template.NewWebhookTemplateContext(
		"dry-run-event-id",
		eventName,
		time.Now().UTC().Format(time.RFC3339),
		1,
		event.SamplePayload,
	)

	result, err := engine.TransformPayloadWith(transformTemplate, data, template.ExecOptions{StrictMissingKeys: strict})
	if err != nil {
		return "", svcerrors.Wrapf(err, svcerrors.InvalidArgument, "template transformation failed: %v", err)
	}

	return string(result), nil
}

func (s *WebhookService) GetTemplateFunctions() []TemplateFunctionInfo {
	functions := template.GetTemplateFunctions()
	res := make([]TemplateFunctionInfo, len(functions))
	for i, f := range functions {
		res[i] = TemplateFunctionInfo{
			Name:        f.Name,
			Description: f.Description,
		}
	}
	return res
}
