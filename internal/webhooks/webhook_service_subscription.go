package webhooks

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"google.golang.org/grpc/codes"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

// getSubscriptionInNamespace loads a subscription by ID and verifies it belongs to the
// given namespace. Returns svcerrors.NotFoundError if the subscription is not in the namespace.
func (s *WebhookService) getSubscriptionInNamespace(ctx context.Context, subscriptionID string, namespace string) (*store.EventSubscription, error) {
	tenantID := tenant.DefaultTenantID

	if namespace == "" {
		return nil, svcerrors.Error(codes.InvalidArgument, "namespace is required")
	}

	id, err := parseUUID(subscriptionID, "subscription ID")
	if err != nil {
		return nil, err
	}

	sub, err := s.webhookRepo.GetSubscription(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if sub.Namespace != namespace {
		return nil, svcerrors.Error(codes.NotFound, "subscription not found in namespace")
	}
	return sub, nil
}

// Subscription Management Implementation

func (s *WebhookService) CreateSubscription(ctx context.Context, webhookID, eventName, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string, labelFilters map[string]string) (string, time.Time, error) {
	s.logger.InfoContext(ctx, "Creating subscription", "webhook_id", webhookID, "event_name", eventName, "namespace", namespace)

	if namespace == "" {
		return "", time.Time{}, svcerrors.Error(codes.InvalidArgument, "namespace is required")
	}

	tenantID := tenant.DefaultTenantID

	id, err := parseUUID(webhookID, "webhook ID")
	if err != nil {
		return "", time.Time{}, err
	}

	if err := validateLabels(labelFilters, "label_filters"); err != nil {
		return "", time.Time{}, err
	}

	sub := &store.EventSubscription{
		WebhookID:         id,
		EventName:         eventName,
		Namespace:         namespace,
		Headers:           headers,
		Method:            method,
		Timeout:           timeout,
		TransformEnabled:  transformEnabled,
		TransformTemplate: transformTemplate,
		LabelFilters:      labelFilters,
	}

	if err := s.webhookRepo.CreateSubscription(ctx, tenantID, sub); err != nil {
		return "", time.Time{}, err
	}

	return sub.ID.String(), sub.CreatedAt, nil
}

func (s *WebhookService) GetSubscription(ctx context.Context, subscriptionID string, namespace string) (*store.EventSubscription, error) {
	return s.getSubscriptionInNamespace(ctx, subscriptionID, namespace)
}

func (s *WebhookService) ListSubscriptions(ctx context.Context, namespace string, webhookID string, eventName string, limit, offset int32) ([]*store.EventSubscription, int32, error) {
	if namespace == "" {
		return nil, 0, svcerrors.Error(codes.InvalidArgument, "namespace is required")
	}

	tenantID := tenant.DefaultTenantID

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var subs []*store.EventSubscription
	var totalCount int
	var err error

	if webhookID != "" {
		var id uuid.UUID
		id, err = parseUUID(webhookID, "webhook ID")
		if err != nil {
			return nil, 0, err
		}
		subs, err = s.webhookRepo.ListSubscriptions(ctx, tenantID, id)
		totalCount = len(subs)
		// Apply pagination manually for now if repo doesn't support it for ListSubscriptions
		if int(offset) < len(subs) {
			end := int(offset + limit)
			if end > len(subs) {
				end = len(subs)
			}
			subs = subs[int(offset):end]
		} else {
			subs = []*store.EventSubscription{}
		}
	} else if eventName != "" {
		subs, err = s.webhookRepo.GetSubscriptionsByEvent(ctx, tenantID, namespace, eventName, nil)
		totalCount = len(subs)
		if int(offset) < len(subs) {
			end := int(offset + limit)
			if end > len(subs) {
				end = len(subs)
			}
			subs = subs[int(offset):end]
		} else {
			subs = []*store.EventSubscription{}
		}
	} else {
		// List all subscriptions in namespace
		subs, totalCount, err = s.webhookRepo.ListSubscriptionsByNamespace(ctx, tenantID, namespace, int(limit), int(offset))
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list subscriptions: %w", err)
		}
	}

	return subs, int32(totalCount), err
}

func (s *WebhookService) UpdateSubscription(ctx context.Context, subscriptionID string, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string, labelFilters map[string]string) error {
	if err := validateLabels(labelFilters, "label_filters"); err != nil {
		return err
	}

	sub, err := s.getSubscriptionInNamespace(ctx, subscriptionID, namespace)
	if err != nil {
		return err
	}

	sub.Headers = headers
	sub.Method = method
	sub.Timeout = timeout
	sub.TransformEnabled = transformEnabled
	sub.TransformTemplate = transformTemplate
	sub.LabelFilters = labelFilters

	return s.webhookRepo.UpdateSubscription(ctx, tenant.DefaultTenantID, sub)
}

func (s *WebhookService) DeleteSubscription(ctx context.Context, subscriptionID string, namespace string) error {
	sub, err := s.getSubscriptionInNamespace(ctx, subscriptionID, namespace)
	if err != nil {
		return err
	}
	return s.webhookRepo.DeleteSubscription(ctx, tenant.DefaultTenantID, sub.ID)
}

func (s *WebhookService) TestSubscriptionTemplate(ctx context.Context, eventName, transformTemplate, namespace string) (string, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.TestSubscriptionTemplate")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing test subscription template request", "event_name", eventName, "namespace", namespace)

	if eventName == "" {
		return "", svcerrors.Error(codes.InvalidArgument, "event name is required")
	}

	tenantID := tenant.DefaultTenantID

	event, err := s.webhookRepo.GetEventByName(ctx, tenantID, eventName)
	if err != nil {
		return "", fmt.Errorf("failed to get event: %w", err)
	}
	if event == nil {
		return "", svcerrors.Error(codes.NotFound, "event not found")
	}

	engine := client.NewTemplateEngine()

	// Create context for template
	data := client.NewWebhookTemplateContext(
		"dry-run-event-id",
		eventName,
		time.Now().UTC().Format(time.RFC3339),
		1,
		event.SamplePayload,
	)

	result, err := engine.TransformPayload(transformTemplate, data)
	if err != nil {
		return "", svcerrors.Wrapf(err, codes.InvalidArgument, "template transformation failed: %v", err)
	}

	return string(result), nil
}

func (s *WebhookService) GetTemplateFunctions() []TemplateFunctionInfo {
	functions := client.GetTemplateFunctions()
	res := make([]TemplateFunctionInfo, len(functions))
	for i, f := range functions {
		res[i] = TemplateFunctionInfo{
			Name:        f.Name,
			Description: f.Description,
		}
	}
	return res
}
