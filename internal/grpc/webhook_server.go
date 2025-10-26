package grpc

import (
	"context"

	"github.com/sarathsp06/sparrow/internal/queue"
	"github.com/sarathsp06/sparrow/internal/services"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	pb "github.com/sarathsp06/sparrow/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WebhookServer implements the WebhookService gRPC interface
type WebhookServer struct {
	pb.UnimplementedWebhookServiceServer
	service *services.WebhookService
}

// NewWebhookServer creates a new WebhookServer instance
func NewWebhookServer(queueManager *queue.Manager, webhookRepo *webhooks.Repository) *WebhookServer {
	return &WebhookServer{
		service: services.NewWebhookService(queueManager, webhookRepo),
	}
}

// RegisterWebhook registers a URL for specific events in a namespace
func (s *WebhookServer) RegisterWebhook(ctx context.Context, req *pb.RegisterWebhookRequest) (*pb.RegisterWebhookResponse, error) {
	serviceReq := &services.RegisterWebhookRequest{
		Namespace:   req.Namespace,
		Events:      req.Events,
		URL:         req.Url,
		Headers:     req.Headers,
		Timeout:     req.Timeout,
		Active:      req.Active,
		Description: req.Description,
	}

	resp, err := s.service.RegisterWebhook(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register webhook: %v", err)
	}

	return &pb.RegisterWebhookResponse{
		WebhookId: resp.WebhookID,
		Success:   resp.Success,
		Message:   resp.Message,
		CreatedAt: resp.CreatedAt,
	}, nil
}

// UnregisterWebhook removes a webhook registration
func (s *WebhookServer) UnregisterWebhook(ctx context.Context, req *pb.UnregisterWebhookRequest) (*pb.UnregisterWebhookResponse, error) {
	serviceReq := &services.UnregisterWebhookRequest{
		WebhookID: req.WebhookId,
	}

	resp, err := s.service.UnregisterWebhook(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unregister webhook: %v", err)
	}

	return &pb.UnregisterWebhookResponse{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}

// PushEvent pushes an event that triggers registered webhooks
func (s *WebhookServer) PushEvent(ctx context.Context, req *pb.PushEventRequest) (*pb.PushEventResponse, error) {
	serviceReq := &services.PushEventRequest{
		Namespace:  req.Namespace,
		Event:      req.Event,
		Payload:    req.Payload,
		TTLSeconds: req.TtlSeconds,
		Metadata:   req.Metadata,
	}

	resp, err := s.service.PushEvent(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to push event: %v", err)
	}

	return &pb.PushEventResponse{
		EventId:           resp.EventID,
		WebhooksTriggered: resp.WebhooksTriggered,
		WebhookIds:        resp.WebhookIDs,
		Success:           resp.Success,
		Message:           resp.Message,
	}, nil
}

// GetWebhookStatus gets the status of webhook deliveries
func (s *WebhookServer) GetWebhookStatus(ctx context.Context, req *pb.GetWebhookStatusRequest) (*pb.GetWebhookStatusResponse, error) {
	serviceReq := &services.GetWebhookStatusRequest{}

	switch id := req.Identifier.(type) {
	case *pb.GetWebhookStatusRequest_WebhookId:
		if id.WebhookId == "" {
			return nil, status.Error(codes.InvalidArgument, "webhook_id is required")
		}
		serviceReq.WebhookID = id.WebhookId
	case *pb.GetWebhookStatusRequest_EventId:
		if id.EventId == "" {
			return nil, status.Error(codes.InvalidArgument, "event_id is required")
		}
		serviceReq.EventID = id.EventId
	default:
		return nil, status.Error(codes.InvalidArgument, "either webhook_id or event_id is required")
	}

	resp, err := s.service.GetWebhookStatus(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get webhook status: %v", err)
	}

	// Convert to protobuf format
	pbDeliveries := make([]*pb.WebhookDelivery, len(resp.Deliveries))
	for i, d := range resp.Deliveries {
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

	return &pb.GetWebhookStatusResponse{
		Deliveries:      pbDeliveries,
		TotalDeliveries: resp.TotalDeliveries,
		Success:         resp.Success,
		Message:         resp.Message,
	}, nil
}

// ListWebhooks lists all registered webhooks for a namespace
func (s *WebhookServer) ListWebhooks(ctx context.Context, req *pb.ListWebhooksRequest) (*pb.ListWebhooksResponse, error) {
	serviceReq := &services.ListWebhooksRequest{
		Namespace:  req.Namespace,
		Event:      req.Event,
		ActiveOnly: req.ActiveOnly,
	}

	resp, err := s.service.ListWebhooks(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list webhooks: %v", err)
	}

	// Convert to protobuf format
	pbWebhooks := make([]*pb.RegisteredWebhook, len(resp.Webhooks))
	for i, reg := range resp.Webhooks {
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

	return &pb.ListWebhooksResponse{
		Webhooks:   pbWebhooks,
		TotalCount: resp.TotalCount,
		Success:    resp.Success,
		Message:    resp.Message,
	}, nil
}

// RegisterEvent registers a new event type
func (s *WebhookServer) RegisterEvent(ctx context.Context, req *pb.RegisterEventRequest) (*pb.RegisterEventResponse, error) {
	// Convert to service request
	serviceReq := &services.RegisterEventRequest{
		Name:        req.Name,
		Description: req.Description,
		Schema:      req.Schema,
		Metadata:    req.Metadata,
		Active:      req.Active,
	}

	// Call service
	serviceResp, err := s.service.RegisterEvent(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to register event: %v", err)
	}

	if !serviceResp.Success {
		return nil, status.Errorf(codes.InvalidArgument, "%s", serviceResp.Message)
	}

	// Convert to protobuf response
	return &pb.RegisterEventResponse{
		EventId:   serviceResp.EventID,
		Success:   serviceResp.Success,
		Message:   serviceResp.Message,
		CreatedAt: serviceResp.CreatedAt,
	}, nil
}

// ListEvents lists all registered event types
func (s *WebhookServer) ListEvents(ctx context.Context, req *pb.ListEventsRequest) (*pb.ListEventsResponse, error) {
	// Convert to service request
	serviceReq := &services.ListEventsRequest{
		ActiveOnly: req.ActiveOnly,
	}

	// Call service
	serviceResp, err := s.service.ListEvents(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to list events: %v", err)
	}

	if !serviceResp.Success {
		return nil, status.Errorf(codes.InvalidArgument, "%s", serviceResp.Message)
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

	return &pb.ListEventsResponse{
		Events:     pbEvents,
		TotalCount: serviceResp.TotalCount,
		Success:    serviceResp.Success,
		Message:    serviceResp.Message,
	}, nil
}

// UpdateEvent updates an event registration
func (s *WebhookServer) UpdateEvent(ctx context.Context, req *pb.UpdateEventRequest) (*pb.UpdateEventResponse, error) {
	// Convert to service request
	serviceReq := &services.UpdateEventRequest{
		Name:        req.Name,
		Description: req.Description,
		Schema:      req.Schema,
		Metadata:    req.Metadata,
		Active:      req.Active,
	}

	// Call service
	serviceResp, err := s.service.UpdateEvent(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to update event: %v", err)
	}

	if !serviceResp.Success {
		return nil, status.Errorf(codes.InvalidArgument, "%s", serviceResp.Message)
	}

	return &pb.UpdateEventResponse{
		Success: serviceResp.Success,
		Message: serviceResp.Message,
	}, nil
}

// DeleteEvent deletes an event registration
func (s *WebhookServer) DeleteEvent(ctx context.Context, req *pb.DeleteEventRequest) (*pb.DeleteEventResponse, error) {
	// Convert to service request
	serviceReq := &services.DeleteEventRequest{
		Name: req.Name,
	}

	// Call service
	serviceResp, err := s.service.DeleteEvent(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to delete event: %v", err)
	}

	if !serviceResp.Success {
		return nil, status.Errorf(codes.InvalidArgument, "%s", serviceResp.Message)
	}

	return &pb.DeleteEventResponse{
		Success: serviceResp.Success,
		Message: serviceResp.Message,
	}, nil
}

// GetWebhookHealth gets health metrics for a webhook
func (s *WebhookServer) GetWebhookHealth(ctx context.Context, req *pb.GetWebhookHealthRequest) (*pb.GetWebhookHealthResponse, error) {
	// Convert to service request
	serviceReq := &services.GetWebhookHealthRequest{
		WebhookID: req.WebhookId,
		Namespace: req.Namespace,
	}

	// Call service
	serviceResp, err := s.service.GetWebhookHealth(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get webhook health: %v", err)
	}

	if !serviceResp.Success {
		return nil, status.Errorf(codes.InvalidArgument, "%s", serviceResp.Message)
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

	return &pb.GetWebhookHealthResponse{
		Success:   serviceResp.Success,
		Message:   serviceResp.Message,
		WebhookId: serviceResp.WebhookID,
		Health:    convertWebhookHealth(serviceResp.Health),
		Metrics:   pbMetrics,
	}, nil
}

// ListWebhooksByHealth lists webhooks filtered by health status
func (s *WebhookServer) ListWebhooksByHealth(ctx context.Context, req *pb.ListWebhooksByHealthRequest) (*pb.ListWebhooksByHealthResponse, error) {
	// Convert to service request
	serviceReq := &services.ListWebhooksByHealthRequest{
		Health: convertPbHealthToInternal(req.Health),
	}

	// Call service
	serviceResp, err := s.service.ListWebhooksByHealth(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to list webhooks by health: %v", err)
	}

	if !serviceResp.Success {
		return nil, status.Errorf(codes.InvalidArgument, "%s", serviceResp.Message)
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

	return &pb.ListWebhooksByHealthResponse{
		Success:    serviceResp.Success,
		Message:    serviceResp.Message,
		Webhooks:   pbWebhooks,
		TotalCount: serviceResp.TotalCount,
	}, nil
}

// GetHealthSummary gets a summary of webhook health
func (s *WebhookServer) GetHealthSummary(ctx context.Context, req *pb.GetHealthSummaryRequest) (*pb.GetHealthSummaryResponse, error) {
	// Convert to service request
	serviceReq := &services.GetHealthSummaryRequest{}

	// Call service
	serviceResp, err := s.service.GetHealthSummary(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get health summary: %v", err)
	}

	if !serviceResp.Success {
		return nil, status.Errorf(codes.InvalidArgument, "%s", serviceResp.Message)
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

	return &pb.GetHealthSummaryResponse{
		Success: serviceResp.Success,
		Message: serviceResp.Message,
		Summary: pbSummary,
	}, nil
}

// ResubmitWebhook manually retries failed or pending webhook deliveries
func (s *WebhookServer) ResubmitWebhook(ctx context.Context, req *pb.ResubmitWebhookRequest) (*pb.ResubmitWebhookResponse, error) {
	// Convert to service request
	serviceReq := &services.ResubmitWebhookRequest{
		Namespace: req.Namespace,
		Force:     req.Force,
	}

	// Handle the identifier (either delivery_id or webhook_id)
	switch id := req.Identifier.(type) {
	case *pb.ResubmitWebhookRequest_DeliveryId:
		if id.DeliveryId == "" {
			return nil, status.Error(codes.InvalidArgument, "delivery_id cannot be empty")
		}
		serviceReq.DeliveryID = id.DeliveryId
	case *pb.ResubmitWebhookRequest_WebhookId:
		if id.WebhookId == "" {
			return nil, status.Error(codes.InvalidArgument, "webhook_id cannot be empty")
		}
		serviceReq.WebhookID = id.WebhookId
	default:
		return nil, status.Error(codes.InvalidArgument, "either delivery_id or webhook_id is required")
	}

	// Call service
	serviceResp, err := s.service.ResubmitWebhook(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to resubmit webhook: %v", err)
	}

	if !serviceResp.Success {
		return nil, status.Errorf(codes.InvalidArgument, "%s", serviceResp.Message)
	}

	return &pb.ResubmitWebhookResponse{
		Success:          serviceResp.Success,
		Message:          serviceResp.Message,
		ResubmittedCount: serviceResp.ResubmittedCount,
		DeliveryIds:      serviceResp.DeliveryIDs,
	}, nil
}

// GetRegisteredWebhooks retrieves registered webhooks by ID or namespace
func (s *WebhookServer) GetRegisteredWebhooks(ctx context.Context, req *pb.GetRegisteredWebhooksRequest) (*pb.GetRegisteredWebhooksResponse, error) {
	serviceReq := &services.GetRegisteredWebhooksRequest{
		WebhookID:  req.WebhookId,
		Namespace:  req.Namespace,
		ActiveOnly: req.ActiveOnly,
	}

	serviceResp, err := s.service.GetRegisteredWebhooks(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get registered webhooks: %v", err)
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

	return &pb.GetRegisteredWebhooksResponse{
		Webhooks:   webhooks,
		TotalCount: serviceResp.TotalCount,
		Success:    serviceResp.Success,
		Message:    serviceResp.Message,
	}, nil
}

// ListRegisteredWebhooksByEvent retrieves webhooks registered for specific events
func (s *WebhookServer) ListRegisteredWebhooksByEvent(ctx context.Context, req *pb.ListRegisteredWebhooksByEventRequest) (*pb.ListRegisteredWebhooksByEventResponse, error) {
	serviceReq := &services.ListRegisteredWebhooksByEventRequest{
		Namespace:  req.Namespace,
		Event:      req.Event,
		ActiveOnly: req.ActiveOnly,
	}

	serviceResp, err := s.service.ListRegisteredWebhooksByEvent(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list webhooks by event: %v", err)
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

	return &pb.ListRegisteredWebhooksByEventResponse{
		Webhooks:   webhooks,
		Event:      serviceResp.Event,
		Namespace:  serviceResp.Namespace,
		TotalCount: serviceResp.TotalCount,
		Success:    serviceResp.Success,
		Message:    serviceResp.Message,
	}, nil
}

// GetWebhookDeliveryStatus retrieves delivery status for specific delivery
func (s *WebhookServer) GetWebhookDeliveryStatus(ctx context.Context, req *pb.GetWebhookDeliveryStatusRequest) (*pb.GetWebhookDeliveryStatusResponse, error) {
	serviceReq := &services.GetWebhookDeliveryStatusRequest{
		DeliveryID: req.DeliveryId,
		Namespace:  req.Namespace,
	}

	serviceResp, err := s.service.GetWebhookDeliveryStatus(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get delivery status: %v", err)
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

	return &pb.GetWebhookDeliveryStatusResponse{
		Delivery: delivery,
		Success:  serviceResp.Success,
		Message:  serviceResp.Message,
	}, nil
}

// ResendWebhook resends a failed webhook delivery
func (s *WebhookServer) ResendWebhook(ctx context.Context, req *pb.ResendWebhookRequest) (*pb.ResendWebhookResponse, error) {
	serviceReq := &services.ResendWebhookRequest{
		DeliveryID:  req.DeliveryId,
		Namespace:   req.Namespace,
		ForceResend: req.ForceResend,
	}

	serviceResp, err := s.service.ResendWebhook(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resend webhook: %v", err)
	}

	return &pb.ResendWebhookResponse{
		NewDeliveryId: serviceResp.NewDeliveryID,
		Success:       serviceResp.Success,
		Message:       serviceResp.Message,
	}, nil
}

// GetWebhookDeliveryHistory retrieves delivery history for a webhook
func (s *WebhookServer) GetWebhookDeliveryHistory(ctx context.Context, req *pb.GetWebhookDeliveryHistoryRequest) (*pb.GetWebhookDeliveryHistoryResponse, error) {
	serviceReq := &services.GetWebhookDeliveryHistoryRequest{
		WebhookID: req.WebhookId,
		Namespace: req.Namespace,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}

	serviceResp, err := s.service.GetWebhookDeliveryHistory(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get delivery history: %v", err)
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

	return &pb.GetWebhookDeliveryHistoryResponse{
		Deliveries: deliveries,
		TotalCount: serviceResp.TotalCount,
		Success:    serviceResp.Success,
		Message:    serviceResp.Message,
	}, nil
}

// GetNamespaceStats retrieves statistics for a namespace
func (s *WebhookServer) GetNamespaceStats(ctx context.Context, req *pb.GetNamespaceStatsRequest) (*pb.GetNamespaceStatsResponse, error) {
	serviceReq := &services.GetNamespaceStatsRequest{
		Namespace: req.Namespace,
	}

	serviceResp, err := s.service.GetNamespaceStats(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get namespace stats: %v", err)
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

	return &pb.GetNamespaceStatsResponse{
		Namespace: serviceResp.Namespace,
		Stats:     stats,
		Success:   serviceResp.Success,
		Message:   serviceResp.Message,
	}, nil
}

// UpdateWebhookConfig updates webhook configuration
func (s *WebhookServer) UpdateWebhookConfig(ctx context.Context, req *pb.UpdateWebhookConfigRequest) (*pb.UpdateWebhookConfigResponse, error) {
	var updates *webhooks.WebhookUpdateFields
	if req.Updates != nil {
		updates = &webhooks.WebhookUpdateFields{
			Events:      req.Updates.Events,
			URL:         req.Updates.Url,
			Headers:     req.Updates.Headers,
			Timeout:     int(req.Updates.Timeout),
			Active:      req.Updates.Active,
			Description: req.Updates.Description,
		}
	}

	serviceReq := &services.UpdateWebhookConfigRequest{
		WebhookID: req.WebhookId,
		Namespace: req.Namespace,
		Updates:   updates,
	}

	serviceResp, err := s.service.UpdateWebhookConfig(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update webhook config: %v", err)
	}

	return &pb.UpdateWebhookConfigResponse{
		Success: serviceResp.Success,
		Message: serviceResp.Message,
	}, nil
}

// PauseWebhook temporarily disables a webhook
func (s *WebhookServer) PauseWebhook(ctx context.Context, req *pb.PauseWebhookRequest) (*pb.PauseWebhookResponse, error) {
	serviceReq := &services.PauseWebhookRequest{
		WebhookID: req.WebhookId,
		Namespace: req.Namespace,
		Reason:    req.Reason,
	}

	serviceResp, err := s.service.PauseWebhook(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to pause webhook: %v", err)
	}

	return &pb.PauseWebhookResponse{
		Success: serviceResp.Success,
		Message: serviceResp.Message,
	}, nil
}

// ResumeWebhook re-enables a paused webhook
func (s *WebhookServer) ResumeWebhook(ctx context.Context, req *pb.ResumeWebhookRequest) (*pb.ResumeWebhookResponse, error) {
	serviceReq := &services.ResumeWebhookRequest{
		WebhookID: req.WebhookId,
		Namespace: req.Namespace,
	}

	serviceResp, err := s.service.ResumeWebhook(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resume webhook: %v", err)
	}

	return &pb.ResumeWebhookResponse{
		Success: serviceResp.Success,
		Message: serviceResp.Message,
	}, nil
}

// Helper function to convert delivery status
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
