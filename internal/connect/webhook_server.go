package connect

import (
	"context"
	"errors"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	pb "github.com/sarathsp06/sparrow/proto"
	"github.com/sarathsp06/sparrow/proto/protoconnect"
)

// WebhookConnectServer implements the WebhookService Connect-RPC interface
type WebhookConnectServer struct {
	service webhooks.WebhookServiceInterface
}

// NewWebhookConnectServer creates a new Connect-RPC server
func NewWebhookConnectServer(queueManager *queue.Manager, webhookRepo *store.Repository) *WebhookConnectServer {
	return &WebhookConnectServer{
		service: webhooks.NewWebhookService(queueManager, webhookRepo),
	}
}

// RegisterWebhook registers a URL for specific events in a namespace
func (s *WebhookConnectServer) RegisterWebhook(
	ctx context.Context,
	req *connect.Request[pb.RegisterWebhookRequest],
) (*connect.Response[pb.RegisterWebhookResponse], error) {
	webhookID, createdAt, err := s.service.RegisterWebhook(ctx, req.Msg.Namespace, req.Msg.Events, req.Msg.Url, req.Msg.Headers, int(req.Msg.Timeout), req.Msg.Active, req.Msg.Description)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.RegisterWebhookResponse{
		WebhookId: webhookID,
		Success:   true,
		Message:   "Webhook registered successfully",
		CreatedAt: createdAt,
	}

	return connect.NewResponse(result), nil
}

// UnregisterWebhook removes a webhook registration
func (s *WebhookConnectServer) UnregisterWebhook(
	ctx context.Context,
	req *connect.Request[pb.UnregisterWebhookRequest],
) (*connect.Response[pb.UnregisterWebhookResponse], error) {
	err := s.service.UnregisterWebhook(ctx, req.Msg.WebhookId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.UnregisterWebhookResponse{
		Success: true,
		Message: "Webhook unregistered successfully",
	}

	return connect.NewResponse(result), nil
}

// PushEvent pushes an event that triggers registered webhooks
func (s *WebhookConnectServer) PushEvent(
	ctx context.Context,
	req *connect.Request[pb.PushEventRequest],
) (*connect.Response[pb.PushEventResponse], error) {
	eventID, webhooksTriggered, webhookIDs, err := s.service.PushEvent(ctx, req.Msg.Namespace, req.Msg.Event, req.Msg.Payload, req.Msg.TtlSeconds, req.Msg.Metadata)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.PushEventResponse{
		EventId:           eventID,
		WebhooksTriggered: webhooksTriggered,
		WebhookIds:        webhookIDs,
		Success:           true,
		Message:           "Event pushed successfully",
	}

	return connect.NewResponse(result), nil
}

// GetWebhookStatus gets the status of webhook deliveries
func (s *WebhookConnectServer) GetWebhookStatus(
	ctx context.Context,
	req *connect.Request[pb.GetWebhookStatusRequest],
) (*connect.Response[pb.GetWebhookStatusResponse], error) {
	var webhookID, eventID string
	switch id := req.Msg.Identifier.(type) {
	case *pb.GetWebhookStatusRequest_WebhookId:
		if id.WebhookId == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("webhook_id is required"))
		}
		webhookID = id.WebhookId
	case *pb.GetWebhookStatusRequest_EventId:
		if id.EventId == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("event_id is required"))
		}
		eventID = id.EventId
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("either webhook_id or event_id is required"))
	}
	deliveries, totalDeliveries, err := s.service.GetWebhookStatus(ctx, webhookID, eventID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pbDeliveries := make([]*pb.WebhookDelivery, len(deliveries))
	for i, d := range deliveries {
		pbDeliveries[i] = &pb.WebhookDelivery{
			DeliveryId:   d.ID,
			WebhookId:    d.WebhookID,
			EventId:      d.EventID,
			Status:       convertDeliveryStatus(d.Status),
			AttemptCount: int32(d.AttemptCount),
			MaxAttempts:  int32(d.MaxAttempts),
			CreatedAt:    d.CreatedAt.Unix(),
			ExpiresAt:    d.ExpiresAt.Unix(),
			ResponseCode: int32(d.ResponseCode),
			ResponseBody: d.ResponseBody,
			ErrorMessage: d.ErrorMessage,
		}
		if d.LastAttemptedAt != nil {
			pbDeliveries[i].LastAttemptedAt = d.LastAttemptedAt.Unix()
		}
		if d.NextRetryAt != nil {
			pbDeliveries[i].NextRetryAt = d.NextRetryAt.Unix()
		}
	}
	return connect.NewResponse(&pb.GetWebhookStatusResponse{
		Deliveries:      pbDeliveries,
		TotalDeliveries: totalDeliveries,
		Success:         true,
		Message:         "Webhook status retrieved successfully",
	}), nil
}

// ListWebhooks lists all registered webhooks for a namespace
func (s *WebhookConnectServer) ListWebhooks(
	ctx context.Context,
	req *connect.Request[pb.ListWebhooksRequest],
) (*connect.Response[pb.ListWebhooksResponse], error) {
	regs, err := s.service.ListWebhooks(ctx, req.Msg.Namespace, req.Msg.Event, req.Msg.ActiveOnly)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert to protobuf format
	pbWebhooks := make([]*pb.RegisteredWebhook, len(regs))
	for i, reg := range regs {
		pbWebhooks[i] = &pb.RegisteredWebhook{
			WebhookId:   reg.ID,
			Namespace:   reg.Namespace,
			Events:      reg.Events,
			Url:         reg.URL,
			Headers:     reg.Headers,
			Timeout:     int32(reg.Timeout),
			Active:      reg.Active,
			Description: reg.Description,
			Health:      convertWebhookHealth(reg.Health),
			CreatedAt:   reg.CreatedAt.Unix(),
			UpdatedAt:   reg.UpdatedAt.Unix(),
		}
	}

	result := &pb.ListWebhooksResponse{
		Webhooks:   pbWebhooks,
		TotalCount: int32(len(pbWebhooks)),
		Success:    true,
		Message:    "Webhooks listed successfully",
	}

	return connect.NewResponse(result), nil
}

// RegisterEvent registers a new event type
func (s *WebhookConnectServer) RegisterEvent(
	ctx context.Context,
	req *connect.Request[pb.RegisterEventRequest],
) (*connect.Response[pb.RegisterEventResponse], error) {
	eventID, createdAt, err := s.service.RegisterEvent(ctx, req.Msg.Name, req.Msg.Description, req.Msg.Schema, req.Msg.Metadata, req.Msg.Active)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.RegisterEventResponse{
		EventId:   eventID,
		Success:   true,
		Message:   "Event registered successfully",
		CreatedAt: createdAt,
	}

	return connect.NewResponse(result), nil
}


// ListEvents lists all registered event types
func (s *WebhookConnectServer) ListEvents(
	ctx context.Context,
	req *connect.Request[pb.ListEventsRequest],
) (*connect.Response[pb.ListEventsResponse], error) {
	events, err := s.service.ListEvents(ctx, req.Msg.ActiveOnly)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pbEvents := make([]*pb.RegisteredEvent, len(events))
	for i, event := range events {
		pbEvents[i] = &pb.RegisteredEvent{
			EventId:     event.ID,
			Name:        event.Name,
			Description: event.Description,
			Schema:      event.Schema,
			Metadata:    event.Metadata,
			Active:      event.Active,
			CreatedAt:   event.CreatedAt.Unix(),
			UpdatedAt:   event.UpdatedAt.Unix(),
		}
	}

	result := &pb.ListEventsResponse{
		Events:     pbEvents,
		TotalCount: int32(len(pbEvents)),
		Success:    true,
		Message:    "Events listed successfully",
	}

	return connect.NewResponse(result), nil
}

// UpdateEvent updates an event registration
func (s *WebhookConnectServer) UpdateEvent(
	ctx context.Context,
	req *connect.Request[pb.UpdateEventRequest],
) (*connect.Response[pb.UpdateEventResponse], error) {
	err := s.service.UpdateEvent(ctx, req.Msg.Name, req.Msg.Description, req.Msg.Schema, req.Msg.Metadata, req.Msg.Active)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.UpdateEventResponse{
		Success: true,
		Message: "Event updated successfully",
	}

	return connect.NewResponse(result), nil
}

// DeleteEvent deletes an event registration
func (s *WebhookConnectServer) DeleteEvent(
	ctx context.Context,
	req *connect.Request[pb.DeleteEventRequest],
) (*connect.Response[pb.DeleteEventResponse], error) {
	err := s.service.DeleteEvent(ctx, req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.DeleteEventResponse{
		Success: true,
		Message: "Event deleted successfully",
	}

	return connect.NewResponse(result), nil
}

// GetWebhookHealth gets health metrics for a webhook
func (s *WebhookConnectServer) GetWebhookHealth(
	ctx context.Context,
	req *connect.Request[pb.GetWebhookHealthRequest],
) (*connect.Response[pb.GetWebhookHealthResponse], error) {
	healthData, err := s.service.GetWebhookHealth(ctx, req.Msg.WebhookId, req.Msg.Namespace)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var pbMetrics *pb.WebhookHealthMetrics
	if healthData != nil {
		pbMetrics = &pb.WebhookHealthMetrics{
			WebhookId:            healthData.WebhookID,
			TotalDeliveries:      int32(healthData.TotalDeliveries),
			SuccessfulDeliveries: int32(healthData.SuccessfulDeliveries),
			FailedDeliveries:     int32(healthData.FailedDeliveries),
			ConsecutiveFailures:  int32(healthData.ConsecutiveFailures),
			SuccessRate:          healthData.SuccessRate,
			AvgResponseTime:      int32(healthData.AvgResponseTime),
			CreatedAt:            healthData.CreatedAt.Unix(),
			UpdatedAt:            healthData.UpdatedAt.Unix(),
		}

		if healthData.LastSuccessAt != nil {
			pbMetrics.LastSuccessAt = healthData.LastSuccessAt.Unix()
		}

		if healthData.LastFailureAt != nil {
			pbMetrics.LastFailureAt = healthData.LastFailureAt.Unix()
		}
	}

	result := &pb.GetWebhookHealthResponse{
		Success:   true,
		Message:   "Webhook health retrieved successfully",
		WebhookId: req.Msg.WebhookId,
		Health:    convertWebhookHealth(healthData.Health),
		Metrics:   pbMetrics,
	}

	return connect.NewResponse(result), nil
}


// ListWebhooksByHealth lists webhooks filtered by health status
func (s *WebhookConnectServer) ListWebhooksByHealth(
	ctx context.Context,
	req *connect.Request[pb.ListWebhooksByHealthRequest],
) (*connect.Response[pb.ListWebhooksByHealthResponse], error) {
	health := convertPbHealthToInternal(req.Msg.Health)
	webhooks, err := s.service.ListWebhooksByHealth(ctx, health)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pbWebhooks := make([]*pb.RegisteredWebhook, len(webhooks))
	for i, webhook := range webhooks {
		pbWebhooks[i] = &pb.RegisteredWebhook{
			WebhookId:   webhook.ID,
			Namespace:   webhook.Namespace,
			Events:      webhook.Events,
			Url:         webhook.URL,
			Headers:     webhook.Headers,
			Timeout:     int32(webhook.Timeout),
			Active:      webhook.Active,
			Description: webhook.Description,
			Health:      convertWebhookHealth(webhook.Health),
			CreatedAt:   webhook.CreatedAt.Unix(),
			UpdatedAt:   webhook.UpdatedAt.Unix(),
		}
	}

	result := &pb.ListWebhooksByHealthResponse{
		Success:    true,
		Message:    "Webhooks by health listed successfully",
		Webhooks:   pbWebhooks,
		TotalCount: int32(len(pbWebhooks)),
	}

	return connect.NewResponse(result), nil
}

// GetHealthSummary gets a summary of webhook health
func (s *WebhookConnectServer) GetHealthSummary(
	ctx context.Context,
	req *connect.Request[pb.GetHealthSummaryRequest],
) (*connect.Response[pb.GetHealthSummaryResponse], error) {
	summaryData, err := s.service.GetHealthSummary(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var pbSummary *pb.HealthSummary
	if summaryData != nil {
		pbSummary = &pb.HealthSummary{
			HealthyCount:   int32(summaryData.HealthyCount),
			DegradedCount:  int32(summaryData.DegradedCount),
			UnhealthyCount: int32(summaryData.UnhealthyCount),
			UnknownCount:   int32(summaryData.UnknownCount),
			TotalCount:     int32(summaryData.TotalCount),
		}
	}

	result := &pb.GetHealthSummaryResponse{
		Success: true,
		Message: "Health summary retrieved successfully",
		Summary: pbSummary,
	}

	return connect.NewResponse(result), nil
}


// ResubmitWebhook manually retries failed or pending webhook deliveries
func (s *WebhookConnectServer) ResubmitWebhook(
	ctx context.Context,
	req *connect.Request[pb.ResubmitWebhookRequest],
) (*connect.Response[pb.ResubmitWebhookResponse], error) {
	var deliveryID, webhookID string
	switch id := req.Msg.Identifier.(type) {
	case *pb.ResubmitWebhookRequest_DeliveryId:
		if id.DeliveryId == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("delivery_id cannot be empty"))
		}
		deliveryID = id.DeliveryId
	case *pb.ResubmitWebhookRequest_WebhookId:
		if id.WebhookId == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("webhook_id cannot be empty"))
		}
		webhookID = id.WebhookId
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("either delivery_id or webhook_id is required"))
	}
	deliveryIDs, resubmittedCount, err := s.service.ResubmitWebhook(ctx, deliveryID, webhookID, req.Msg.Namespace, req.Msg.Force)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.ResubmitWebhookResponse{
		Success:          true,
		Message:          "Webhook resubmitted successfully",
		DeliveryIds:      deliveryIDs,
		ResubmittedCount: resubmittedCount,
	}

	return connect.NewResponse(result), nil
}

// GetRegisteredWebhooks retrieves registered webhooks by ID or namespace
func (s *WebhookConnectServer) GetRegisteredWebhooks(
	ctx context.Context,
	req *connect.Request[pb.GetRegisteredWebhooksRequest],
) (*connect.Response[pb.GetRegisteredWebhooksResponse], error) {
	regs, err := s.service.GetRegisteredWebhooks(ctx, req.Msg.Namespace, req.Msg.WebhookId, req.Msg.ActiveOnly)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var webhooks []*pb.RegisteredWebhook
	for _, webhook := range regs {
		webhooks = append(webhooks, &pb.RegisteredWebhook{
			WebhookId:   webhook.ID,
			Namespace:   webhook.Namespace,
			Events:      webhook.Events,
			Url:         webhook.URL,
			Headers:     webhook.Headers,
			Timeout:     int32(webhook.Timeout),
			Active:      webhook.Active,
			Description: webhook.Description,
			Health:      convertWebhookHealth(webhook.Health),
			CreatedAt:   webhook.CreatedAt.Unix(),
			UpdatedAt:   webhook.UpdatedAt.Unix(),
		})
	}

	result := &pb.GetRegisteredWebhooksResponse{
		Webhooks:   webhooks,
		TotalCount: int32(len(webhooks)),
		Success:    true,
		Message:    "Webhooks retrieved successfully",
	}

	return connect.NewResponse(result), nil
}

// ListRegisteredWebhooksByEvent retrieves webhooks registered for specific events
func (s *WebhookConnectServer) ListRegisteredWebhooksByEvent(
	ctx context.Context,
	req *connect.Request[pb.ListRegisteredWebhooksByEventRequest],
) (*connect.Response[pb.ListRegisteredWebhooksByEventResponse], error) {
	regs, err := s.service.ListRegisteredWebhooksByEvent(ctx, req.Msg.Namespace, req.Msg.Event, req.Msg.ActiveOnly)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var webhooks []*pb.RegisteredWebhook
	for _, webhook := range regs {
		webhooks = append(webhooks, &pb.RegisteredWebhook{
			WebhookId:   webhook.ID,
			Namespace:   webhook.Namespace,
			Events:      webhook.Events,
			Url:         webhook.URL,
			Headers:     webhook.Headers,
			Timeout:     int32(webhook.Timeout),
			Active:      webhook.Active,
			Description: webhook.Description,
			Health:      convertWebhookHealth(webhook.Health),
			CreatedAt:   webhook.CreatedAt.Unix(),
			UpdatedAt:   webhook.UpdatedAt.Unix(),
		})
	}

	result := &pb.ListRegisteredWebhooksByEventResponse{
		Webhooks:   webhooks,
		Event:      req.Msg.Event,
		Namespace:  req.Msg.Namespace,
		TotalCount: int32(len(webhooks)),
		Success:    true,
		Message:    "Webhooks by event retrieved successfully",
	}

	return connect.NewResponse(result), nil
}

// GetWebhookDeliveryStatus retrieves delivery status for specific delivery
func (s *WebhookConnectServer) GetWebhookDeliveryStatus(
	ctx context.Context,
	req *connect.Request[pb.GetWebhookDeliveryStatusRequest],
) (*connect.Response[pb.GetWebhookDeliveryStatusResponse], error) {
	delivery, err := s.service.GetWebhookDeliveryStatus(ctx, req.Msg.DeliveryId, req.Msg.Namespace)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var pbDelivery *pb.WebhookDelivery
	if delivery != nil {
		pbDelivery = &pb.WebhookDelivery{
			DeliveryId:   delivery.ID,
			WebhookId:    delivery.WebhookID,
			EventId:      delivery.EventID,
			Status:       convertDeliveryStatus(delivery.Status),
			AttemptCount: int32(delivery.AttemptCount),
			MaxAttempts:  int32(delivery.MaxAttempts),
			CreatedAt:    delivery.CreatedAt.Unix(),
			ExpiresAt:    delivery.ExpiresAt.Unix(),
			ResponseCode: int32(delivery.ResponseCode),
			ResponseBody: delivery.ResponseBody,
			ErrorMessage: delivery.ErrorMessage,
		}

		if delivery.LastAttemptedAt != nil {
			pbDelivery.LastAttemptedAt = delivery.LastAttemptedAt.Unix()
		}
		if delivery.NextRetryAt != nil {
			pbDelivery.NextRetryAt = delivery.NextRetryAt.Unix()
		}
	}

	result := &pb.GetWebhookDeliveryStatusResponse{
		Delivery: pbDelivery,
		Success:  true,
		Message:  "Delivery status retrieved successfully",
	}

	return connect.NewResponse(result), nil
}

// ResendWebhook resends a failed webhook delivery
func (s *WebhookConnectServer) ResendWebhook(
	ctx context.Context,
	req *connect.Request[pb.ResendWebhookRequest],
) (*connect.Response[pb.ResendWebhookResponse], error) {
	newDeliveryID, err := s.service.ResendWebhook(ctx, req.Msg.DeliveryId, req.Msg.Namespace, req.Msg.ForceResend)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.ResendWebhookResponse{
		NewDeliveryId: newDeliveryID,
		Success:       true,
		Message:       "Webhook resent successfully",
	}

	return connect.NewResponse(result), nil
}

// GetWebhookDeliveryHistory retrieves delivery history for a webhook
func (s *WebhookConnectServer) GetWebhookDeliveryHistory(
	ctx context.Context,
	req *connect.Request[pb.GetWebhookDeliveryHistoryRequest],
) (*connect.Response[pb.GetWebhookDeliveryHistoryResponse], error) {
	deliveries, totalCount, err := s.service.GetWebhookDeliveryHistory(ctx, req.Msg.WebhookId, req.Msg.Namespace, req.Msg.Limit, req.Msg.Offset)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var pbDeliveries []*pb.WebhookDelivery
	for _, delivery := range deliveries {
		pbDelivery := &pb.WebhookDelivery{
			DeliveryId:   delivery.ID,
			WebhookId:    delivery.WebhookID,
			EventId:      delivery.EventID,
			Status:       convertDeliveryStatus(delivery.Status),
			AttemptCount: int32(delivery.AttemptCount),
			MaxAttempts:  int32(delivery.MaxAttempts),
			CreatedAt:    delivery.CreatedAt.Unix(),
			ExpiresAt:    delivery.ExpiresAt.Unix(),
			ResponseCode: int32(delivery.ResponseCode),
			ResponseBody: delivery.ResponseBody,
			ErrorMessage: delivery.ErrorMessage,
		}

		if delivery.LastAttemptedAt != nil {
			pbDelivery.LastAttemptedAt = delivery.LastAttemptedAt.Unix()
		}
		if delivery.NextRetryAt != nil {
			pbDelivery.NextRetryAt = delivery.NextRetryAt.Unix()
		}

		pbDeliveries = append(pbDeliveries, pbDelivery)
	}

	result := &pb.GetWebhookDeliveryHistoryResponse{
		Deliveries: pbDeliveries,
		TotalCount: totalCount,
		Success:    true,
		Message:    "Delivery history retrieved successfully",
	}

	return connect.NewResponse(result), nil
}

// GetNamespaceStats retrieves statistics for a namespace
func (s *WebhookConnectServer) GetNamespaceStats(
	ctx context.Context,
	req *connect.Request[pb.GetNamespaceStatsRequest],
) (*connect.Response[pb.GetNamespaceStatsResponse], error) {
	stats, err := s.service.GetNamespaceStats(ctx, req.Msg.Namespace)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var pbStats *pb.NamespaceStats
	if stats != nil {
		pbStats = &pb.NamespaceStats{
			TotalWebhooks:        int32(stats.TotalWebhooks),
			ActiveWebhooks:       int32(stats.ActiveWebhooks),
			TotalDeliveries:      int32(stats.TotalDeliveries),
			SuccessfulDeliveries: int32(stats.SuccessfulDeliveries),
			FailedDeliveries:     int32(stats.FailedDeliveries),
			PendingDeliveries:    int32(stats.PendingDeliveries),
			SuccessRate:          stats.SuccessRate,
		}
	}

	result := &pb.GetNamespaceStatsResponse{
		Namespace: req.Msg.Namespace,
		Stats:     pbStats,
		Success:   true,
		Message:   "Namespace stats retrieved successfully",
	}

	return connect.NewResponse(result), nil
}

// UpdateWebhookConfig updates webhook configuration
func (s *WebhookConnectServer) UpdateWebhookConfig(
	ctx context.Context,
	req *connect.Request[pb.UpdateWebhookConfigRequest],
) (*connect.Response[pb.UpdateWebhookConfigResponse], error) {
	var events []string
	var url string
	var headers map[string]string
	var timeout int
	var active bool
	var description string
	if req.Msg.Updates != nil {
		events = req.Msg.Updates.Events
		url = req.Msg.Updates.Url
		headers = req.Msg.Updates.Headers
		timeout = int(req.Msg.Updates.Timeout)
		active = req.Msg.Updates.Active
		description = req.Msg.Updates.Description
	}
	err := s.service.UpdateWebhookConfig(ctx, req.Msg.WebhookId, req.Msg.Namespace, events, url, headers, timeout, active, description)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.UpdateWebhookConfigResponse{
		Success: true,
		Message: "Webhook config updated successfully",
	}

	return connect.NewResponse(result), nil
}

// PauseWebhook temporarily disables a webhook
func (s *WebhookConnectServer) PauseWebhook(
	ctx context.Context,
	req *connect.Request[pb.PauseWebhookRequest],
) (*connect.Response[pb.PauseWebhookResponse], error) {
	err := s.service.PauseWebhook(ctx, req.Msg.WebhookId, req.Msg.Namespace, req.Msg.Reason)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.PauseWebhookResponse{
		Success: true,
		Message: "Webhook paused successfully",
	}

	return connect.NewResponse(result), nil
}

// ResumeWebhook re-enables a paused webhook
func (s *WebhookConnectServer) ResumeWebhook(
	ctx context.Context,
	req *connect.Request[pb.ResumeWebhookRequest],
) (*connect.Response[pb.ResumeWebhookResponse], error) {
	err := s.service.ResumeWebhook(ctx, req.Msg.WebhookId, req.Msg.Namespace)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.ResumeWebhookResponse{
		Success: true,
		Message: "Webhook resumed successfully",
	}

	return connect.NewResponse(result), nil
}

// convertDeliveryStatus converts internal status to protobuf status
func convertDeliveryStatus(status store.WebhookDeliveryStatus) pb.WebhookDeliveryStatus {
	switch status {
	case store.StatusPending:
		return pb.WebhookDeliveryStatus_DELIVERY_PENDING
	case store.StatusSending:
		return pb.WebhookDeliveryStatus_DELIVERY_SENDING
	case store.StatusSuccess:
		return pb.WebhookDeliveryStatus_DELIVERY_SUCCESS
	case store.StatusFailed:
		return pb.WebhookDeliveryStatus_DELIVERY_FAILED
	case store.StatusRetrying:
		return pb.WebhookDeliveryStatus_DELIVERY_RETRYING
	case store.StatusExpired:
		return pb.WebhookDeliveryStatus_DELIVERY_EXPIRED
	default:
		return pb.WebhookDeliveryStatus_DELIVERY_UNKNOWN
	}
}

// Helper function to convert webhook health to protobuf
func convertWebhookHealth(health store.WebhookHealth) pb.WebhookHealth {
	switch health {
	case store.HealthHealthy:
		return pb.WebhookHealth_HEALTH_HEALTHY
	case store.HealthDegraded:
		return pb.WebhookHealth_HEALTH_DEGRADED
	case store.HealthUnhealthy:
		return pb.WebhookHealth_HEALTH_UNHEALTHY
	case store.HealthUnknown:
		return pb.WebhookHealth_HEALTH_UNKNOWN
	default:
		return pb.WebhookHealth_HEALTH_UNKNOWN
	}
}

// Helper function to convert protobuf health to internal
func convertPbHealthToInternal(health pb.WebhookHealth) store.WebhookHealth {
	switch health {
	case pb.WebhookHealth_HEALTH_HEALTHY:
		return store.HealthHealthy
	case pb.WebhookHealth_HEALTH_DEGRADED:
		return store.HealthDegraded
	case pb.WebhookHealth_HEALTH_UNHEALTHY:
		return store.HealthUnhealthy
	case pb.WebhookHealth_HEALTH_UNKNOWN:
		return store.HealthUnknown
	default:
		return store.HealthUnknown
	}
}

// Handler returns the Connect-RPC handler
func (s *WebhookConnectServer) Handler() (string, http.Handler) {
	// Create simple handler
	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		log.Fatal(err)
	}
	path, handler := protoconnect.NewWebhookServiceHandler(s, connect.WithInterceptors(otelInterceptor))
	return path, handler
}
