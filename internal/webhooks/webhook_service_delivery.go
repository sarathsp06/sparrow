package webhooks

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

// GetDeliveryStatus gets the status of a webhook delivery.
// When namespace is empty, looks up by delivery ID alone.
func (s *WebhookService) GetDeliveryStatus(ctx context.Context, deliveryID string, namespace string) (*store.WebhookDelivery, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetDeliveryStatus")
	defer span.End()

	s.logger.InfoContext(ctx, "Getting webhook delivery status",
		"delivery_id", deliveryID,
		"namespace", namespace)

	if deliveryID == "" {
		return nil, svcerrors.Error(codes.InvalidArgument, "delivery ID is required")
	}

	tenantID := tenant.DefaultTenantID

	id, err := parseUUID(deliveryID, "delivery ID")
	if err != nil {
		return nil, err
	}

	delivery, err := s.webhookRepo.GetDeliveryByID(ctx, tenantID, id, namespace)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get delivery by ID", "error", err)
		return nil, fmt.Errorf("failed to retrieve delivery status: %w", err)
	}
	if delivery == nil {
		return nil, svcerrors.Error(codes.NotFound, "delivery not found")
	}
	return delivery, nil
}

// GetDeliveryAttempts retrieves individual attempt history for a delivery.
// Returns all recorded health events for the delivery, ordered by timestamp ascending.
func (s *WebhookService) GetDeliveryAttempts(ctx context.Context, deliveryID string) ([]*store.WebhookHealthEvent, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetDeliveryAttempts")
	defer span.End()

	tenantID := tenant.DefaultTenantID

	s.logger.InfoContext(ctx, "Getting delivery attempts", "delivery_id", deliveryID, "tenant_id", tenantID.String())

	if deliveryID == "" {
		return nil, svcerrors.Error(codes.InvalidArgument, "delivery ID is required")
	}

	id, err := parseUUID(deliveryID, "delivery ID")
	if err != nil {
		return nil, err
	}

	attempts, err := s.webhookRepo.GetDeliveryAttempts(ctx, tenantID, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get delivery attempts", "error", err)
		return nil, fmt.Errorf("failed to retrieve delivery attempts: %w", err)
	}

	return attempts, nil
}

// ListDeliveries retrieves delivery history with filters.
// Supports filtering by namespace, webhook, event, status, error_category,
// subscription, and time range via the DeliveryFilter struct.
// When PrepareRetry is true, snapshots all matching delivery IDs into a batch job and returns the batch ID.
func (s *WebhookService) ListDeliveries(ctx context.Context, filter store.DeliveryFilter) ([]*store.WebhookDelivery, int32, string, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListDeliveries")
	defer span.End()

	s.logger.InfoContext(ctx, "Listing deliveries",
		"namespace", filter.Namespace,
		"webhook_id", filter.WebhookID,
		"event_id", filter.EventID,
		"status", filter.Status,
		"error_category", filter.ErrorCategory,
		"prepare_retry", filter.PrepareRetry,
		"limit", filter.Limit,
		"offset", filter.Offset)

	tenantID := tenant.DefaultTenantID

	filter.Limit, filter.Offset = normalizePagination(filter.Limit, filter.Offset)

	deliveries, totalCount, err := s.webhookRepo.ListDeliveriesFiltered(ctx, tenantID, filter)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list deliveries", "error", err)
		return nil, 0, "", fmt.Errorf("failed to retrieve deliveries: %w", err)
	}

	// Snapshot matching IDs into a batch job if requested
	var retryID string
	if filter.PrepareRetry {
		ids, err := s.webhookRepo.SnapshotDeliveryIDs(ctx, tenantID, filter)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to snapshot delivery IDs for retry", "error", err)
			return nil, 0, "", fmt.Errorf("failed to prepare retry: %w", err)
		}
		if len(ids) > 0 {
			filterMap := map[string]any{
				"namespace": filter.Namespace,
			}
			if filter.WebhookID != nil {
				filterMap["webhook_id"] = filter.WebhookID.String()
			}
			if filter.EventID != nil {
				filterMap["event_id"] = filter.EventID.String()
			}
			if filter.Status != nil {
				filterMap["status"] = *filter.Status
			}
			if filter.ErrorCategory != nil {
				filterMap["error_category"] = *filter.ErrorCategory
			}
			batchData := &store.BatchJobData{
				ItemIDs: ids,
				Filter:  filterMap,
			}
			batchJob, err := s.webhookRepo.CreateBatchJob(ctx, tenantID, filter.Namespace, store.BatchTypeDeliveryRetry, batchData)
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to create batch job for retry", "error", err)
				return nil, 0, "", fmt.Errorf("failed to create retry batch: %w", err)
			}
			retryID = batchJob.ID.String()
			s.logger.InfoContext(ctx, "Created retry batch job",
				"retry_id", retryID,
				"delivery_count", len(ids))
		}
	}

	return deliveries, int32(totalCount), retryID, nil
}

// RetryDelivery manually retries failed or pending webhook deliveries
func (s *WebhookService) RetryDelivery(ctx context.Context, namespace string, deliveryID string, webhookID string, force bool) ([]string, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.RetryDelivery")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing retry delivery request",
		"delivery_id", deliveryID,
		"webhook_id", webhookID,
		"namespace", namespace,
		"force", force)

	tenantID := tenant.DefaultTenantID

	// Validate required fields
	if deliveryID == "" && webhookID == "" {
		return nil, 0, svcerrors.Error(codes.InvalidArgument, "either delivery_id or webhook_id is required")
	}

	// Namespace is required for webhook-level retry (multiple deliveries),
	// but optional for single-delivery retry (delivery_id is globally unique within a tenant).
	if namespace == "" && webhookID != "" {
		return nil, 0, svcerrors.Error(codes.InvalidArgument, "namespace is required for webhook-level retry")
	}

	if deliveryID != "" && webhookID != "" {
		return nil, 0, svcerrors.Error(codes.InvalidArgument, "only one of delivery_id or webhook_id can be specified")
	}

	var deliveriesToResubmit []*store.WebhookDelivery

	if deliveryID != "" {
		id, err := parseUUID(deliveryID, "delivery ID")
		if err != nil {
			return nil, 0, err
		}

		// Resubmit specific delivery
		delivery, err := s.webhookRepo.GetDeliveryByID(ctx, tenantID, id, namespace)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get delivery", "error", err)
			return nil, 0, fmt.Errorf("failed to retrieve delivery: %w", err)
		}

		if delivery == nil {
			return nil, 0, svcerrors.Error(codes.NotFound, "delivery not found")
		}

		// Check if delivery can be resubmitted
		if !force && delivery.Status == store.StatusSuccess {
			return nil, 0, svcerrors.Error(codes.FailedPrecondition, "delivery already succeeded. Use force to resubmit anyway")
		}

		deliveriesToResubmit = []*store.WebhookDelivery{delivery}
	} else {
		id, err := parseUUID(webhookID, "webhook ID")
		if err != nil {
			return nil, 0, err
		}

		// Resubmit all failed/pending deliveries for webhook
		_, err = s.webhookRepo.GetWebhookByID(ctx, tenantID, id, namespace)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get webhook", "error", err)
			return nil, 0, fmt.Errorf("failed to retrieve webhook: %w", err)
		}

		// Get retriable deliveries
		deliveriesToResubmit, err = s.webhookRepo.GetRetriableDeliveries(ctx, tenantID, id, namespace, force)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get retriable deliveries", "error", err)
			return nil, 0, fmt.Errorf("failed to retrieve deliveries: %w", err)
		}

		if len(deliveriesToResubmit) == 0 {
			message := "No failed or pending deliveries found"
			if !force {
				message += ". Use force to resubmit all deliveries"
			}
			s.logger.InfoContext(ctx, message)
			return []string{}, 0, nil
		}
	}

	// Process each delivery for resubmission
	var resubmittedIDs []string
	var resubmittedCount int32

	for _, delivery := range deliveriesToResubmit {
		// Reset delivery status to pending
		err := s.webhookRepo.ResetDeliveryForRetry(ctx, delivery.ID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to reset delivery for retry",
				"delivery_id", delivery.ID,
				"error", err)
			continue
		}

		// Get webhook info for queuing
		webhook, err := s.webhookRepo.GetWebhookByID(ctx, tenantID, delivery.WebhookID, namespace)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get webhook for delivery",
				"webhook_id", delivery.WebhookID,
				"delivery_id", delivery.ID,
				"error", err)
			continue
		}

		// Queue the webhook for delivery.
		// Manual retries never expire -- use far-future sentinel so TTL doesn't apply.
		args := &queue.WebhookArgs{
			DeliveryID:  delivery.ID.String(),
			WebhookID:   delivery.WebhookID.String(),
			EventID:     delivery.EventID.String(),
			ExpiresAt:   store.NoExpiryTime,
			Namespace:   webhook.Namespace,
			TenantID:    tenantID.String(),
			MaxAttempts: delivery.MaxAttempts,
		}
		if delivery.SubscriptionID != nil {
			args.SubscriptionID = delivery.SubscriptionID.String()
		}

		_, err = s.jobInserter.Insert(ctx, args)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to queue webhook for resubmission",
				"delivery_id", delivery.ID,
				"webhook_id", delivery.WebhookID,
				"error", err)
			continue
		}

		resubmittedIDs = append(resubmittedIDs, delivery.ID.String())
		resubmittedCount++
	}

	if resubmittedCount == 0 {
		return nil, 0, svcerrors.Error(codes.FailedPrecondition, "failed to resubmit any deliveries")
	}

	s.logger.InfoContext(ctx, "Webhook deliveries resubmitted successfully",
		"resubmitted_count", resubmittedCount,
		"total_requested", len(deliveriesToResubmit))

	return resubmittedIDs, resubmittedCount, nil
}
