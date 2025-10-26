package connect

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"

	"github.com/sarathsp06/sparrow/internal/queue"
	"github.com/sarathsp06/sparrow/internal/services"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	pb "github.com/sarathsp06/sparrow/proto"
	"github.com/sarathsp06/sparrow/proto/protoconnect"
)

// WebhookConnectServer implements the WebhookService Connect-RPC interface
type WebhookConnectServer struct {
	service *services.WebhookService
}

// NewWebhookConnectServer creates a new Connect-RPC server instance
func NewWebhookConnectServer(queueManager *queue.Manager, webhookRepo *webhooks.Repository) *WebhookConnectServer {
	return &WebhookConnectServer{
		service: services.NewWebhookService(queueManager, webhookRepo),
	}
}

// RegisterWebhook registers a URL for specific events in a namespace
func (s *WebhookConnectServer) RegisterWebhook(
	ctx context.Context,
	req *connect.Request[pb.RegisterWebhookRequest],
) (*connect.Response[pb.RegisterWebhookResponse], error) {
	// Convert to service request
	serviceReq := &services.RegisterWebhookRequest{
		Namespace:   req.Msg.Namespace,
		Events:      req.Msg.Events,
		URL:         req.Msg.Url,
		Headers:     req.Msg.Headers,
		Timeout:     req.Msg.Timeout,
		Active:      req.Msg.Active,
		Description: req.Msg.Description,
	}

	// Call service
	serviceResp, err := s.service.RegisterWebhook(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !serviceResp.Success {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(serviceResp.Message))
	}

	// Convert to protobuf response
	result := &pb.RegisterWebhookResponse{
		WebhookId: serviceResp.WebhookID,
		Success:   serviceResp.Success,
		Message:   serviceResp.Message,
		CreatedAt: serviceResp.CreatedAt,
	}

	return connect.NewResponse(result), nil
}

// UnregisterWebhook removes a webhook registration
func (s *WebhookConnectServer) UnregisterWebhook(
	ctx context.Context,
	req *connect.Request[pb.UnregisterWebhookRequest],
) (*connect.Response[pb.UnregisterWebhookResponse], error) {
	// Convert to service request
	serviceReq := &services.UnregisterWebhookRequest{
		WebhookID: req.Msg.WebhookId,
	}

	// Call service
	serviceResp, err := s.service.UnregisterWebhook(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !serviceResp.Success {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(serviceResp.Message))
	}

	// Convert to protobuf response
	result := &pb.UnregisterWebhookResponse{
		Success: serviceResp.Success,
		Message: serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// PushEvent pushes an event that triggers registered webhooks
func (s *WebhookConnectServer) PushEvent(
	ctx context.Context,
	req *connect.Request[pb.PushEventRequest],
) (*connect.Response[pb.PushEventResponse], error) {
	// Convert to service request
	serviceReq := &services.PushEventRequest{
		Namespace:  req.Msg.Namespace,
		Event:      req.Msg.Event,
		Payload:    req.Msg.Payload,
		TTLSeconds: req.Msg.TtlSeconds,
		Metadata:   req.Msg.Metadata,
	}

	// Call service
	serviceResp, err := s.service.PushEvent(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !serviceResp.Success {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(serviceResp.Message))
	}

	// Convert to protobuf response
	result := &pb.PushEventResponse{
		EventId:           serviceResp.EventID,
		WebhooksTriggered: serviceResp.WebhooksTriggered,
		WebhookIds:        serviceResp.WebhookIDs,
		Success:           serviceResp.Success,
		Message:           serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// GetWebhookStatus gets the status of webhook deliveries
func (s *WebhookConnectServer) GetWebhookStatus(
	ctx context.Context,
	req *connect.Request[pb.GetWebhookStatusRequest],
) (*connect.Response[pb.GetWebhookStatusResponse], error) {
	// This method needs new service methods to be implemented
	// For now, return not implemented
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("GetWebhookStatus not yet implemented in service layer"))
}

// ListWebhooks lists all registered webhooks for a namespace
func (s *WebhookConnectServer) ListWebhooks(
	ctx context.Context,
	req *connect.Request[pb.ListWebhooksRequest],
) (*connect.Response[pb.ListWebhooksResponse], error) {
	// Convert to service request
	serviceReq := &services.ListWebhooksRequest{
		Namespace:  req.Msg.Namespace,
		Event:      req.Msg.Event,
		ActiveOnly: req.Msg.ActiveOnly,
	}

	// Call service
	serviceResp, err := s.service.ListWebhooks(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !serviceResp.Success {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(serviceResp.Message))
	}

	// Convert to protobuf format
	pbWebhooks := make([]*pb.RegisteredWebhook, len(serviceResp.Webhooks))
	for i, reg := range serviceResp.Webhooks {
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

	// Convert to protobuf response
	result := &pb.ListWebhooksResponse{
		Webhooks:   pbWebhooks,
		TotalCount: serviceResp.TotalCount,
		Success:    serviceResp.Success,
		Message:    serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// RegisterEvent registers a new event type
func (s *WebhookConnectServer) RegisterEvent(
	ctx context.Context,
	req *connect.Request[pb.RegisterEventRequest],
) (*connect.Response[pb.RegisterEventResponse], error) {
	// Convert to service request
	serviceReq := &services.RegisterEventRequest{
		Name:        req.Msg.Name,
		Description: req.Msg.Description,
		Schema:      req.Msg.Schema,
		Metadata:    req.Msg.Metadata,
		Active:      req.Msg.Active,
	}

	// Call service
	serviceResp, err := s.service.RegisterEvent(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !serviceResp.Success {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(serviceResp.Message))
	}

	// Convert to protobuf response
	result := &pb.RegisterEventResponse{
		EventId:   serviceResp.EventID,
		Success:   serviceResp.Success,
		Message:   serviceResp.Message,
		CreatedAt: serviceResp.CreatedAt,
	}

	return connect.NewResponse(result), nil
}

// ListEvents lists all registered event types
func (s *WebhookConnectServer) ListEvents(
	ctx context.Context,
	req *connect.Request[pb.ListEventsRequest],
) (*connect.Response[pb.ListEventsResponse], error) {
	// Convert to service request
	serviceReq := &services.ListEventsRequest{
		ActiveOnly: req.Msg.ActiveOnly,
	}

	// Call service
	serviceResp, err := s.service.ListEvents(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !serviceResp.Success {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(serviceResp.Message))
	}

	// Convert to protobuf format
	pbEvents := make([]*pb.RegisteredEvent, len(serviceResp.Events))
	for i, event := range serviceResp.Events {
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

	// Convert to protobuf response
	result := &pb.ListEventsResponse{
		Events:     pbEvents,
		TotalCount: serviceResp.TotalCount,
		Success:    serviceResp.Success,
		Message:    serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// UpdateEvent updates an event registration
func (s *WebhookConnectServer) UpdateEvent(
	ctx context.Context,
	req *connect.Request[pb.UpdateEventRequest],
) (*connect.Response[pb.UpdateEventResponse], error) {
	// Convert to service request
	serviceReq := &services.UpdateEventRequest{
		Name:        req.Msg.Name,
		Description: req.Msg.Description,
		Schema:      req.Msg.Schema,
		Metadata:    req.Msg.Metadata,
		Active:      req.Msg.Active,
	}

	// Call service
	serviceResp, err := s.service.UpdateEvent(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !serviceResp.Success {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(serviceResp.Message))
	}

	// Convert to protobuf response
	result := &pb.UpdateEventResponse{
		Success: serviceResp.Success,
		Message: serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// DeleteEvent deletes an event registration
func (s *WebhookConnectServer) DeleteEvent(
	ctx context.Context,
	req *connect.Request[pb.DeleteEventRequest],
) (*connect.Response[pb.DeleteEventResponse], error) {
	// Convert to service request
	serviceReq := &services.DeleteEventRequest{
		Name: req.Msg.Name,
	}

	// Call service
	serviceResp, err := s.service.DeleteEvent(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !serviceResp.Success {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(serviceResp.Message))
	}

	// Convert to protobuf response
	result := &pb.DeleteEventResponse{
		Success: serviceResp.Success,
		Message: serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// GetWebhookHealth gets health metrics for a webhook
func (s *WebhookConnectServer) GetWebhookHealth(
	ctx context.Context,
	req *connect.Request[pb.GetWebhookHealthRequest],
) (*connect.Response[pb.GetWebhookHealthResponse], error) {
	// Convert to service request
	serviceReq := &services.GetWebhookHealthRequest{
		WebhookID: req.Msg.WebhookId,
		Namespace: req.Msg.Namespace,
	}

	// Call service
	serviceResp, err := s.service.GetWebhookHealth(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !serviceResp.Success {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(serviceResp.Message))
	}

	// Convert metrics to protobuf format
	var pbMetrics *pb.WebhookHealthMetrics
	if serviceResp.Metrics != nil {
		pbMetrics = &pb.WebhookHealthMetrics{
			WebhookId:            serviceResp.Metrics.WebhookID,
			TotalDeliveries:      int32(serviceResp.Metrics.TotalDeliveries),
			SuccessfulDeliveries: int32(serviceResp.Metrics.SuccessfulDeliveries),
			FailedDeliveries:     int32(serviceResp.Metrics.FailedDeliveries),
			ConsecutiveFailures:  int32(serviceResp.Metrics.ConsecutiveFailures),
			SuccessRate:          serviceResp.Metrics.SuccessRate,
			AvgResponseTime:      int32(serviceResp.Metrics.AvgResponseTime),
			CreatedAt:            serviceResp.Metrics.CreatedAt.Unix(),
			UpdatedAt:            serviceResp.Metrics.UpdatedAt.Unix(),
		}

		if serviceResp.Metrics.LastSuccessAt != nil {
			pbMetrics.LastSuccessAt = serviceResp.Metrics.LastSuccessAt.Unix()
		}

		if serviceResp.Metrics.LastFailureAt != nil {
			pbMetrics.LastFailureAt = serviceResp.Metrics.LastFailureAt.Unix()
		}
	}

	// Convert to protobuf response
	result := &pb.GetWebhookHealthResponse{
		Success:   serviceResp.Success,
		Message:   serviceResp.Message,
		WebhookId: serviceResp.WebhookID,
		Health:    convertWebhookHealth(serviceResp.Health),
		Metrics:   pbMetrics,
	}

	return connect.NewResponse(result), nil
}

// ListWebhooksByHealth lists webhooks filtered by health status
func (s *WebhookConnectServer) ListWebhooksByHealth(
	ctx context.Context,
	req *connect.Request[pb.ListWebhooksByHealthRequest],
) (*connect.Response[pb.ListWebhooksByHealthResponse], error) {
	// Convert to service request
	serviceReq := &services.ListWebhooksByHealthRequest{
		Health: convertPbHealthToInternal(req.Msg.Health),
	}

	// Call service
	serviceResp, err := s.service.ListWebhooksByHealth(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !serviceResp.Success {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(serviceResp.Message))
	}

	// Convert to protobuf format
	pbWebhooks := make([]*pb.RegisteredWebhook, len(serviceResp.Webhooks))
	for i, webhook := range serviceResp.Webhooks {
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

	// Convert to protobuf response
	result := &pb.ListWebhooksByHealthResponse{
		Success:    serviceResp.Success,
		Message:    serviceResp.Message,
		Webhooks:   pbWebhooks,
		TotalCount: serviceResp.TotalCount,
	}

	return connect.NewResponse(result), nil
}

// GetHealthSummary gets a summary of webhook health
func (s *WebhookConnectServer) GetHealthSummary(
	ctx context.Context,
	req *connect.Request[pb.GetHealthSummaryRequest],
) (*connect.Response[pb.GetHealthSummaryResponse], error) {
	// Convert to service request
	serviceReq := &services.GetHealthSummaryRequest{}

	// Call service
	serviceResp, err := s.service.GetHealthSummary(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !serviceResp.Success {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(serviceResp.Message))
	}

	// Convert to protobuf format
	var pbSummary *pb.HealthSummary
	if serviceResp.Summary != nil {
		pbSummary = &pb.HealthSummary{
			HealthyCount:   int32(serviceResp.Summary.HealthyCount),
			DegradedCount:  int32(serviceResp.Summary.DegradedCount),
			UnhealthyCount: int32(serviceResp.Summary.UnhealthyCount),
			UnknownCount:   int32(serviceResp.Summary.UnknownCount),
			TotalCount:     int32(serviceResp.Summary.TotalCount),
		}
	}

	// Convert to protobuf response
	result := &pb.GetHealthSummaryResponse{
		Success: serviceResp.Success,
		Message: serviceResp.Message,
		Summary: pbSummary,
	}

	return connect.NewResponse(result), nil
}

// ResubmitWebhook manually retries failed or pending webhook deliveries
func (s *WebhookConnectServer) ResubmitWebhook(
	ctx context.Context,
	req *connect.Request[pb.ResubmitWebhookRequest],
) (*connect.Response[pb.ResubmitWebhookResponse], error) {
	// Convert to service request
	serviceReq := &services.ResubmitWebhookRequest{
		Namespace: req.Msg.Namespace,
		Force:     req.Msg.Force,
	}

	// Handle the identifier (either delivery_id or webhook_id)
	switch id := req.Msg.Identifier.(type) {
	case *pb.ResubmitWebhookRequest_DeliveryId:
		if id.DeliveryId == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("delivery_id cannot be empty"))
		}
		serviceReq.DeliveryID = id.DeliveryId
	case *pb.ResubmitWebhookRequest_WebhookId:
		if id.WebhookId == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("webhook_id cannot be empty"))
		}
		serviceReq.WebhookID = id.WebhookId
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("either delivery_id or webhook_id is required"))
	}

	// Call service
	serviceResp, err := s.service.ResubmitWebhook(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !serviceResp.Success {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(serviceResp.Message))
	}

	// Convert to protobuf response
	result := &pb.ResubmitWebhookResponse{
		Success:          serviceResp.Success,
		Message:          serviceResp.Message,
		ResubmittedCount: serviceResp.ResubmittedCount,
		DeliveryIds:      serviceResp.DeliveryIDs,
	}

	return connect.NewResponse(result), nil
}

// GetRegisteredWebhooks retrieves registered webhooks by ID or namespace
func (s *WebhookConnectServer) GetRegisteredWebhooks(
	ctx context.Context,
	req *connect.Request[pb.GetRegisteredWebhooksRequest],
) (*connect.Response[pb.GetRegisteredWebhooksResponse], error) {
	serviceReq := &services.GetRegisteredWebhooksRequest{
		WebhookID:  req.Msg.WebhookId,
		Namespace:  req.Msg.Namespace,
		ActiveOnly: req.Msg.ActiveOnly,
	}

	serviceResp, err := s.service.GetRegisteredWebhooks(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var webhooks []*pb.RegisteredWebhook
	for _, webhook := range serviceResp.Webhooks {
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
		TotalCount: serviceResp.TotalCount,
		Success:    serviceResp.Success,
		Message:    serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// ListRegisteredWebhooksByEvent retrieves webhooks registered for specific events
func (s *WebhookConnectServer) ListRegisteredWebhooksByEvent(
	ctx context.Context,
	req *connect.Request[pb.ListRegisteredWebhooksByEventRequest],
) (*connect.Response[pb.ListRegisteredWebhooksByEventResponse], error) {
	serviceReq := &services.ListRegisteredWebhooksByEventRequest{
		Namespace:  req.Msg.Namespace,
		Event:      req.Msg.Event,
		ActiveOnly: req.Msg.ActiveOnly,
	}

	serviceResp, err := s.service.ListRegisteredWebhooksByEvent(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var webhooks []*pb.RegisteredWebhook
	for _, webhook := range serviceResp.Webhooks {
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
		Event:      serviceResp.Event,
		Namespace:  serviceResp.Namespace,
		TotalCount: serviceResp.TotalCount,
		Success:    serviceResp.Success,
		Message:    serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// GetWebhookDeliveryStatus retrieves delivery status for specific delivery
func (s *WebhookConnectServer) GetWebhookDeliveryStatus(
	ctx context.Context,
	req *connect.Request[pb.GetWebhookDeliveryStatusRequest],
) (*connect.Response[pb.GetWebhookDeliveryStatusResponse], error) {
	serviceReq := &services.GetWebhookDeliveryStatusRequest{
		DeliveryID: req.Msg.DeliveryId,
		Namespace:  req.Msg.Namespace,
	}

	serviceResp, err := s.service.GetWebhookDeliveryStatus(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var delivery *pb.WebhookDelivery
	if serviceResp.Delivery != nil {
		delivery = &pb.WebhookDelivery{
			DeliveryId:   serviceResp.Delivery.ID,
			WebhookId:    serviceResp.Delivery.WebhookID,
			EventId:      serviceResp.Delivery.EventID,
			Status:       convertDeliveryStatus(serviceResp.Delivery.Status),
			AttemptCount: int32(serviceResp.Delivery.AttemptCount),
			MaxAttempts:  int32(serviceResp.Delivery.MaxAttempts),
			CreatedAt:    serviceResp.Delivery.CreatedAt.Unix(),
			ExpiresAt:    serviceResp.Delivery.ExpiresAt.Unix(),
			ResponseCode: int32(serviceResp.Delivery.ResponseCode),
			ResponseBody: serviceResp.Delivery.ResponseBody,
			ErrorMessage: serviceResp.Delivery.ErrorMessage,
		}

		if serviceResp.Delivery.LastAttemptedAt != nil {
			delivery.LastAttemptedAt = serviceResp.Delivery.LastAttemptedAt.Unix()
		}
		if serviceResp.Delivery.NextRetryAt != nil {
			delivery.NextRetryAt = serviceResp.Delivery.NextRetryAt.Unix()
		}
	}

	result := &pb.GetWebhookDeliveryStatusResponse{
		Delivery: delivery,
		Success:  serviceResp.Success,
		Message:  serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// ResendWebhook resends a failed webhook delivery
func (s *WebhookConnectServer) ResendWebhook(
	ctx context.Context,
	req *connect.Request[pb.ResendWebhookRequest],
) (*connect.Response[pb.ResendWebhookResponse], error) {
	serviceReq := &services.ResendWebhookRequest{
		DeliveryID:  req.Msg.DeliveryId,
		Namespace:   req.Msg.Namespace,
		ForceResend: req.Msg.ForceResend,
	}

	serviceResp, err := s.service.ResendWebhook(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.ResendWebhookResponse{
		NewDeliveryId: serviceResp.NewDeliveryID,
		Success:       serviceResp.Success,
		Message:       serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// GetWebhookDeliveryHistory retrieves delivery history for a webhook
func (s *WebhookConnectServer) GetWebhookDeliveryHistory(
	ctx context.Context,
	req *connect.Request[pb.GetWebhookDeliveryHistoryRequest],
) (*connect.Response[pb.GetWebhookDeliveryHistoryResponse], error) {
	serviceReq := &services.GetWebhookDeliveryHistoryRequest{
		WebhookID: req.Msg.WebhookId,
		Namespace: req.Msg.Namespace,
		Limit:     req.Msg.Limit,
		Offset:    req.Msg.Offset,
	}

	serviceResp, err := s.service.GetWebhookDeliveryHistory(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var deliveries []*pb.WebhookDelivery
	for _, delivery := range serviceResp.Deliveries {
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

		deliveries = append(deliveries, pbDelivery)
	}

	result := &pb.GetWebhookDeliveryHistoryResponse{
		Deliveries: deliveries,
		TotalCount: serviceResp.TotalCount,
		Success:    serviceResp.Success,
		Message:    serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// GetNamespaceStats retrieves statistics for a namespace
func (s *WebhookConnectServer) GetNamespaceStats(
	ctx context.Context,
	req *connect.Request[pb.GetNamespaceStatsRequest],
) (*connect.Response[pb.GetNamespaceStatsResponse], error) {
	serviceReq := &services.GetNamespaceStatsRequest{
		Namespace: req.Msg.Namespace,
	}

	serviceResp, err := s.service.GetNamespaceStats(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var stats *pb.NamespaceStats
	if serviceResp.Stats != nil {
		stats = &pb.NamespaceStats{
			TotalWebhooks:        int32(serviceResp.Stats.TotalWebhooks),
			ActiveWebhooks:       int32(serviceResp.Stats.ActiveWebhooks),
			TotalDeliveries:      int32(serviceResp.Stats.TotalDeliveries),
			SuccessfulDeliveries: int32(serviceResp.Stats.SuccessfulDeliveries),
			FailedDeliveries:     int32(serviceResp.Stats.FailedDeliveries),
			PendingDeliveries:    int32(serviceResp.Stats.PendingDeliveries),
			SuccessRate:          serviceResp.Stats.SuccessRate,
		}
	}

	result := &pb.GetNamespaceStatsResponse{
		Namespace: serviceResp.Namespace,
		Stats:     stats,
		Success:   serviceResp.Success,
		Message:   serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// UpdateWebhookConfig updates webhook configuration
func (s *WebhookConnectServer) UpdateWebhookConfig(
	ctx context.Context,
	req *connect.Request[pb.UpdateWebhookConfigRequest],
) (*connect.Response[pb.UpdateWebhookConfigResponse], error) {
	var updates *webhooks.WebhookUpdateFields
	if req.Msg.Updates != nil {
		updates = &webhooks.WebhookUpdateFields{
			Events:      req.Msg.Updates.Events,
			URL:         req.Msg.Updates.Url,
			Headers:     req.Msg.Updates.Headers,
			Timeout:     int(req.Msg.Updates.Timeout),
			Active:      req.Msg.Updates.Active,
			Description: req.Msg.Updates.Description,
		}
	}

	serviceReq := &services.UpdateWebhookConfigRequest{
		WebhookID: req.Msg.WebhookId,
		Namespace: req.Msg.Namespace,
		Updates:   updates,
	}

	serviceResp, err := s.service.UpdateWebhookConfig(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.UpdateWebhookConfigResponse{
		Success: serviceResp.Success,
		Message: serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// PauseWebhook temporarily disables a webhook
func (s *WebhookConnectServer) PauseWebhook(
	ctx context.Context,
	req *connect.Request[pb.PauseWebhookRequest],
) (*connect.Response[pb.PauseWebhookResponse], error) {
	serviceReq := &services.PauseWebhookRequest{
		WebhookID: req.Msg.WebhookId,
		Namespace: req.Msg.Namespace,
		Reason:    req.Msg.Reason,
	}

	serviceResp, err := s.service.PauseWebhook(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.PauseWebhookResponse{
		Success: serviceResp.Success,
		Message: serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// ResumeWebhook re-enables a paused webhook
func (s *WebhookConnectServer) ResumeWebhook(
	ctx context.Context,
	req *connect.Request[pb.ResumeWebhookRequest],
) (*connect.Response[pb.ResumeWebhookResponse], error) {
	serviceReq := &services.ResumeWebhookRequest{
		WebhookID: req.Msg.WebhookId,
		Namespace: req.Msg.Namespace,
	}

	serviceResp, err := s.service.ResumeWebhook(ctx, serviceReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &pb.ResumeWebhookResponse{
		Success: serviceResp.Success,
		Message: serviceResp.Message,
	}

	return connect.NewResponse(result), nil
}

// convertDeliveryStatus converts internal status to protobuf status
func convertDeliveryStatus(status webhooks.WebhookDeliveryStatus) pb.WebhookDeliveryStatus {
	switch status {
	case webhooks.StatusPending:
		return pb.WebhookDeliveryStatus_DELIVERY_PENDING
	case webhooks.StatusSending:
		return pb.WebhookDeliveryStatus_DELIVERY_SENDING
	case webhooks.StatusSuccess:
		return pb.WebhookDeliveryStatus_DELIVERY_SUCCESS
	case webhooks.StatusFailed:
		return pb.WebhookDeliveryStatus_DELIVERY_FAILED
	case webhooks.StatusRetrying:
		return pb.WebhookDeliveryStatus_DELIVERY_RETRYING
	case webhooks.StatusExpired:
		return pb.WebhookDeliveryStatus_DELIVERY_EXPIRED
	default:
		return pb.WebhookDeliveryStatus_DELIVERY_UNKNOWN
	}
}

// Helper function to convert webhook health to protobuf
func convertWebhookHealth(health webhooks.WebhookHealth) pb.WebhookHealth {
	switch health {
	case webhooks.HealthHealthy:
		return pb.WebhookHealth_HEALTH_HEALTHY
	case webhooks.HealthDegraded:
		return pb.WebhookHealth_HEALTH_DEGRADED
	case webhooks.HealthUnhealthy:
		return pb.WebhookHealth_HEALTH_UNHEALTHY
	case webhooks.HealthUnknown:
		return pb.WebhookHealth_HEALTH_UNKNOWN
	default:
		return pb.WebhookHealth_HEALTH_UNKNOWN
	}
}

// Helper function to convert protobuf health to internal
func convertPbHealthToInternal(health pb.WebhookHealth) webhooks.WebhookHealth {
	switch health {
	case pb.WebhookHealth_HEALTH_HEALTHY:
		return webhooks.HealthHealthy
	case pb.WebhookHealth_HEALTH_DEGRADED:
		return webhooks.HealthDegraded
	case pb.WebhookHealth_HEALTH_UNHEALTHY:
		return webhooks.HealthUnhealthy
	case pb.WebhookHealth_HEALTH_UNKNOWN:
		return webhooks.HealthUnknown
	default:
		return webhooks.HealthUnknown
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
