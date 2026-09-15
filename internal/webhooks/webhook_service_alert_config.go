package webhooks

import (
	"context"
	"net/mail"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
	"github.com/sarathsp06/sparrow/pkg/storage"
)

// systemEventTypes are the Sparrow-generated event types alert configs may
// subscribe to. Kept in sync with the emission call sites in
// internal/webhooks/queue/webhook_worker.go.
var systemEventTypes = map[string]bool{
	"sparrow.webhook.health_changed":  true,
	"sparrow.webhook.delivery_failed": true,
}

// CreateAlertConfig registers an email recipient for one or more Sparrow
// system event types, either for one webhook (webhookID set) or every
// webhook in the consumer (webhookID empty).
func (s *WebhookService) CreateAlertConfig(ctx context.Context, consumer, webhookID, email string, eventTypes []string) (*store.AlertConfig, error) {
	if consumer == "" {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "consumer is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, svcerrors.Errorf(svcerrors.InvalidArgument, "invalid email: %v", err)
	}
	if len(eventTypes) == 0 {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "event_types must not be empty")
	}
	for _, et := range eventTypes {
		if !systemEventTypes[et] {
			return nil, svcerrors.Errorf(svcerrors.InvalidArgument, "unsupported event_type %q", et)
		}
	}

	tenantID := tenant.DefaultTenantID
	cfg := &store.AlertConfig{
		TenantID:   tenantID,
		Consumer:   consumer,
		Email:      email,
		EventTypes: eventTypes,
	}
	if webhookID != "" {
		id, err := parseUUID(webhookID, "webhook ID")
		if err != nil {
			return nil, err
		}
		if _, err := s.webhookRepo.GetWebhookByID(ctx, tenantID, id, consumer); err != nil {
			if storage.IsNotFound(err) {
				return nil, svcerrors.Error(svcerrors.NotFound, "webhook not found in consumer")
			}
			return nil, err
		}
		cfg.WebhookID = &id
	}

	if err := s.webhookRepo.CreateAlertConfig(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ListAlertConfigs lists alert configs in a consumer, optionally scoped to
// one webhook's own configs (webhookID != ""). Listing without a webhookID
// returns every config in the consumer, including consumer-wide ones.
func (s *WebhookService) ListAlertConfigs(ctx context.Context, consumer string, webhookID string) ([]*store.AlertConfig, error) {
	if consumer == "" {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "consumer is required")
	}
	var whPtr *uuid.UUID
	if webhookID != "" {
		id, err := parseUUID(webhookID, "webhook ID")
		if err != nil {
			return nil, err
		}
		whPtr = &id
	}
	return s.webhookRepo.ListAlertConfigs(ctx, tenant.DefaultTenantID, consumer, whPtr)
}

// DeleteAlertConfig deletes an alert config by id, verifying it belongs to
// the given consumer.
func (s *WebhookService) DeleteAlertConfig(ctx context.Context, consumer, id string) error {
	if consumer == "" {
		return svcerrors.Error(svcerrors.InvalidArgument, "consumer is required")
	}
	parsedID, err := parseUUID(id, "alert config ID")
	if err != nil {
		return err
	}
	tenantID := tenant.DefaultTenantID
	cfg, err := s.webhookRepo.GetAlertConfig(ctx, tenantID, parsedID)
	if err != nil {
		if storage.IsNotFound(err) {
			return svcerrors.Error(svcerrors.NotFound, "alert config not found")
		}
		return err
	}
	if cfg.Consumer != consumer {
		return svcerrors.Error(svcerrors.NotFound, "alert config not found in consumer")
	}
	return s.webhookRepo.DeleteAlertConfig(ctx, tenantID, parsedID)
}
