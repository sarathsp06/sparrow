package grpc

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	pb "github.com/sarathsp06/sparrow/proto"
)

// WebhookServer implements the WebhookService gRPC interface
type WebhookServer struct {
	pb.UnimplementedWebhookServiceServer
	service webhooks.WebhookServiceInterface
}

var _ pb.WebhookServiceServer = (*WebhookServer)(nil)

// NewWebhookServer creates a new WebhookServer instance
func NewWebhookServer(service webhooks.WebhookServiceInterface) *WebhookServer {
	return &WebhookServer{
		service: service,
	}
}

// RegisterWebhook registers a URL for specific events in a namespace
func (s *WebhookServer) RegisterWebhook(ctx context.Context, req *pb.RegisterWebhookRequest) (*pb.RegisterWebhookResponse, error) {
	// Use new service interface if it has CreateWebhook method (enhanced), fallback to legacy method
	if enhancedService, ok := s.service.(interface {
		CreateWebhook(ctx context.Context, req webhooks.WebhookRegistrationRequest) (*webhooks.WebhookRegistration, error)
	}); ok {
		// Use enhanced service with HTTP config support
		webhookReq := CreateWebhookRegistrationRequest(req)
		webhook, err := enhancedService.CreateWebhook(ctx, webhookReq)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to register webhook: %v", err)
		}
		return &pb.RegisterWebhookResponse{
			WebhookId: webhook.ID,
			Success:   true,
			Message:   "Webhook registered successfully",
			CreatedAt: webhook.CreatedAt.Unix(),
		}, nil
	}

	// Fallback to legacy method for backward compatibility
	timeout := int(req.Timeout)
	if timeout == 0 && req.HttpConfig != nil && req.HttpConfig.RequestTimeoutSeconds > 0 {
		timeout = int(req.HttpConfig.RequestTimeoutSeconds)
	}
	if timeout == 0 {
		timeout = 30 // Default timeout
	}

	webhookID, createdAt, err := s.service.RegisterWebhook(ctx, req.Namespace, req.Events, req.Url, req.Headers, timeout, req.Active, req.Description)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register webhook: %v", err)
	}
	return &pb.RegisterWebhookResponse{
		WebhookId: webhookID,
		Success:   true,
		Message:   "Webhook registered successfully",
		CreatedAt: createdAt,
	}, nil
}

// UnregisterWebhook removes a webhook registration
func (s *WebhookServer) UnregisterWebhook(ctx context.Context, req *pb.UnregisterWebhookRequest) (*pb.UnregisterWebhookResponse, error) {
	err := s.service.UnregisterWebhook(ctx, req.WebhookId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unregister webhook: %v", err)
	}
	return &pb.UnregisterWebhookResponse{
		Success: true,
		Message: "Webhook unregistered successfully",
	}, nil
}

// PushEvent pushes an event that triggers registered webhooks
func (s *WebhookServer) PushEvent(ctx context.Context, req *pb.PushEventRequest) (*pb.PushEventResponse, error) {
	eventID, err := s.service.PushEvent(ctx, req.Namespace, req.Event, req.Payload.AsMap(), req.TtlSeconds, req.Metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to push event: %v", err)
	}
	return &pb.PushEventResponse{
		EventId: eventID,
		Success: true,
		Message: "Event pushed successfully",
	}, nil
}

// GetWebhookStatus gets the status of webhook deliveries
func (s *WebhookServer) GetWebhookStatus(ctx context.Context, req *pb.GetWebhookStatusRequest) (*pb.GetWebhookStatusResponse, error) {
	if req.GetWebhookId() == "" {
		return nil, status.Error(codes.InvalidArgument, "webhook_id is required")
	}
	if req.GetNamespace() == "" {
		return nil, status.Error(codes.InvalidArgument, "namespace is required")
	}
	deliveries, totalDeliveries, err := s.service.GetWebhookStatus(ctx, req.GetNamespace(), req.GetWebhookId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get webhook status: %v", err)
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
			RequestBody:  d.RequestBody,
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
		TotalDeliveries: totalDeliveries,
		Success:         true,
		Message:         "Webhook status retrieved successfully",
	}, nil
}

// ListWebhooks lists all registered webhooks for a namespace
func (s *WebhookServer) ListWebhooks(ctx context.Context, req *pb.ListWebhooksRequest) (*pb.ListWebhooksResponse, error) {
	regs, err := s.service.ListWebhooks(ctx, req.Namespace, req.Event, req.ActiveOnly)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list webhooks: %v", err)
	}
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
			HttpConfig: &pb.WebhookHTTPConfig{
				MaxRetries:            int32(reg.MaxRetries),
				RetryBackoffSeconds:   int32(reg.RetryBackoffSeconds),
				CaptureResponseBody:   reg.CaptureResponseBody,
				FollowRedirects:       reg.FollowRedirects,
				VerifySsl:             reg.VerifySSL,
				RequestTimeoutSeconds: int32(reg.RequestTimeoutSeconds),
				ExpectedStatusCodes:   convertExpectedStatusCodes(reg.ExpectedStatusCodes),
				WebhookSecret:         reg.WebhookSecret,
				UserAgent:             reg.UserAgent,
				ContentType:           reg.ContentType,
			},
		}
	}
	return &pb.ListWebhooksResponse{
		Webhooks:   pbWebhooks,
		TotalCount: int32(len(pbWebhooks)),
		Success:    true,
		Message:    "Webhooks listed successfully",
	}, nil
}

// RegisterEvent registers a new event type
func (s *WebhookServer) RegisterEvent(ctx context.Context, req *pb.RegisterEventRequest) (*pb.RegisterEventResponse, error) {
	// Convert JSON schema string to map[string]any
	var schema map[string]any
	if req.Schema != "" {
		if err := json.Unmarshal([]byte(req.Schema), &schema); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid schema JSON: %v", err)
		}
	}

	eventID, createdAt, err := s.service.RegisterEvent(ctx, req.Name, req.Description, schema, req.Metadata, req.Active)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register event: %v", err)
	}
	return &pb.RegisterEventResponse{
		EventId:   eventID,
		Success:   true,
		Message:   "Event registered successfully",
		CreatedAt: createdAt,
	}, nil
}

// ListEvents lists all registered events
func (s *WebhookServer) ListEvents(ctx context.Context, req *pb.ListEventsRequest) (*pb.ListEventsResponse, error) {
	events, err := s.service.ListEvents(ctx, req.ActiveOnly)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list events: %v", err)
	}

	pbEvents := make([]*pb.RegisteredEvent, len(events))
	for i, event := range events {
		pbEvents[i] = &pb.RegisteredEvent{
			EventId:     event.ID,
			Name:        event.Name,
			Description: event.Description,
			Active:      event.Active,
			CreatedAt:   event.CreatedAt.Unix(),
			UpdatedAt:   event.UpdatedAt.Unix(),
		}

		// Convert schema map to JSON string for protobuf
		if event.Schema != nil {
			schemaJSON, err := json.Marshal(event.Schema)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to marshal event schema: %v", err)
			}
			pbEvents[i].Schema = string(schemaJSON)
		}

		// Convert metadata to protobuf format
		pbEvents[i].Metadata = event.Metadata
	}

	return &pb.ListEventsResponse{
		Events:     pbEvents,
		TotalCount: int32(len(pbEvents)),
		Success:    true,
	}, nil
}

// UpdateEvent updates an event registration
func (s *WebhookServer) UpdateEvent(ctx context.Context, req *pb.UpdateEventRequest) (*pb.UpdateEventResponse, error) {
	// Convert JSON schema string to map[string]any
	var schema map[string]any
	if req.Schema != "" {
		if err := json.Unmarshal([]byte(req.Schema), &schema); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid schema JSON: %v", err)
		}
	}

	err := s.service.UpdateEvent(ctx, req.Name, req.Description, schema, req.Metadata, req.Active)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to update event: %v", err)
	}
	return &pb.UpdateEventResponse{
		Success: true,
		Message: "Event updated successfully",
	}, nil
}

// DeleteEvent deletes an event registration
func (s *WebhookServer) DeleteEvent(ctx context.Context, req *pb.DeleteEventRequest) (*pb.DeleteEventResponse, error) {
	err := s.service.DeleteEvent(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to delete event: %v", err)
	}
	return &pb.DeleteEventResponse{
		Success: true,
		Message: "Event deleted successfully",
	}, nil
}

// GetWebhookHealth gets health metrics for a webhook
func (s *WebhookServer) GetWebhookHealth(ctx context.Context, req *pb.GetWebhookHealthRequest) (*pb.GetWebhookHealthResponse, error) {
	healthData, err := s.service.GetWebhookHealth(ctx, req.WebhookId, req.Namespace)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get webhook health: %v", err)
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
	return &pb.GetWebhookHealthResponse{
		Success:   true,
		Message:   "Webhook health retrieved successfully",
		WebhookId: req.WebhookId,
		Health:    convertWebhookHealth(healthData.Health),
		Metrics:   pbMetrics,
	}, nil
}

// GetHealthSummary gets a summary of webhook health
func (s *WebhookServer) GetHealthSummary(ctx context.Context, req *pb.GetHealthSummaryRequest) (*pb.GetHealthSummaryResponse, error) {
	summaryData, err := s.service.GetHealthSummary(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get health summary: %v", err)
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
	return &pb.GetHealthSummaryResponse{
		Success: true,
		Message: "Health summary retrieved successfully",
		Summary: pbSummary,
	}, nil
}

// ResubmitWebhook manually retries failed or pending webhook deliveries
func (s *WebhookServer) ResubmitWebhook(ctx context.Context, req *pb.ResubmitWebhookRequest) (*pb.ResubmitWebhookResponse, error) {
	if req.GetDeliveryId() == "" {
		return nil, status.Error(codes.InvalidArgument, "identifier cannot be empty")
	}

	newDeliveryID, err := s.service.ResendWebhook(ctx, req.GetDeliveryId(), req.Namespace, req.Force)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resubmit delivery: %v", err)
	}
	return &pb.ResubmitWebhookResponse{
		Success:          true,
		Message:          "Delivery resubmitted successfully",
		ResubmittedCount: 1,
		DeliveryIds:      []string{newDeliveryID},
	}, nil
}

// ListWebhooksByHealth lists webhooks filtered by health status
func (s *WebhookServer) ListWebhooksByHealth(ctx context.Context, req *pb.ListWebhooksByHealthRequest) (*pb.ListWebhooksByHealthResponse, error) {
	// Convert protobuf health enum to store health enum
	var storeHealth store.WebhookHealth
	switch req.Health {
	case pb.WebhookHealth_HEALTH_HEALTHY:
		storeHealth = store.HealthHealthy
	case pb.WebhookHealth_HEALTH_DEGRADED:
		storeHealth = store.HealthDegraded
	case pb.WebhookHealth_HEALTH_UNHEALTHY:
		storeHealth = store.HealthUnhealthy
	case pb.WebhookHealth_HEALTH_UNSPECIFIED:
		storeHealth = store.HealthUnknown
	default:
		storeHealth = store.HealthUnknown
	}

	webhooks, err := s.service.ListWebhooksByHealth(ctx, storeHealth)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list webhooks by health: %v", err)
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

	return &pb.ListWebhooksByHealthResponse{
		Webhooks:   pbWebhooks,
		TotalCount: int32(len(pbWebhooks)),
		Success:    true,
	}, nil
}

// GetRegisteredWebhooks retrieves registered webhooks by ID or namespace
func (s *WebhookServer) GetRegisteredWebhooks(ctx context.Context, req *pb.GetRegisteredWebhooksRequest) (*pb.GetRegisteredWebhooksResponse, error) {
	regs, err := s.service.GetRegisteredWebhooks(ctx, req.Namespace, req.WebhookId, req.ActiveOnly)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get registered webhooks: %v", err)
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
	return &pb.GetRegisteredWebhooksResponse{
		Webhooks:   webhooks,
		TotalCount: int32(len(webhooks)),
		Success:    true,
		Message:    "Webhooks retrieved successfully",
	}, nil
}

// ListRegisteredWebhooksByEvent retrieves webhooks registered for specific events
func (s *WebhookServer) ListRegisteredWebhooksByEvent(ctx context.Context, req *pb.ListRegisteredWebhooksByEventRequest) (*pb.ListRegisteredWebhooksByEventResponse, error) {
	regs, err := s.service.ListRegisteredWebhooksByEvent(ctx, req.Namespace, req.Event, req.ActiveOnly)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list webhooks by event: %v", err)
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
	return &pb.ListRegisteredWebhooksByEventResponse{
		Webhooks:   webhooks,
		Event:      req.Event,
		Namespace:  req.Namespace,
		TotalCount: int32(len(webhooks)),
		Success:    true,
		Message:    "Webhooks by event retrieved successfully",
	}, nil
}

// GetWebhookDeliveryStatus retrieves delivery status for specific delivery
func (s *WebhookServer) GetWebhookDeliveryStatus(ctx context.Context, req *pb.GetWebhookDeliveryStatusRequest) (*pb.GetWebhookDeliveryStatusResponse, error) {
	delivery, err := s.service.GetWebhookDeliveryStatus(ctx, req.DeliveryId, req.Namespace)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get delivery status: %v", err)
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
			RequestBody:  delivery.RequestBody,
		}
		if delivery.LastAttemptedAt != nil {
			pbDelivery.LastAttemptedAt = delivery.LastAttemptedAt.Unix()
		}
		if delivery.NextRetryAt != nil {
			pbDelivery.NextRetryAt = delivery.NextRetryAt.Unix()
		}
	}
	return &pb.GetWebhookDeliveryStatusResponse{
		Delivery: pbDelivery,
		Success:  true,
		Message:  "Delivery status retrieved successfully",
	}, nil
}

// ResendWebhook resends a failed webhook delivery
func (s *WebhookServer) ResendWebhook(ctx context.Context, req *pb.ResendWebhookRequest) (*pb.ResendWebhookResponse, error) {
	newDeliveryID, err := s.service.ResendWebhook(ctx, req.DeliveryId, req.Namespace, req.ForceResend)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resend webhook: %v", err)
	}
	return &pb.ResendWebhookResponse{
		NewDeliveryId: newDeliveryID,
		Success:       true,
		Message:       "Webhook resent successfully",
	}, nil
}

// GetWebhookDeliveryHistory retrieves delivery history for a webhook
func (s *WebhookServer) GetWebhookDeliveryHistory(ctx context.Context, req *pb.GetWebhookDeliveryHistoryRequest) (*pb.GetWebhookDeliveryHistoryResponse, error) {
	deliveries, totalCount, err := s.service.GetWebhookDeliveryHistory(ctx, req.WebhookId, req.Namespace, req.Limit, req.Offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get delivery history: %v", err)
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
			RequestBody:  delivery.RequestBody,
		}
		if delivery.LastAttemptedAt != nil {
			pbDelivery.LastAttemptedAt = delivery.LastAttemptedAt.Unix()
		}
		if delivery.NextRetryAt != nil {
			pbDelivery.NextRetryAt = delivery.NextRetryAt.Unix()
		}
		pbDeliveries = append(pbDeliveries, pbDelivery)
	}
	return &pb.GetWebhookDeliveryHistoryResponse{
		Deliveries: pbDeliveries,
		TotalCount: totalCount,
		Success:    true,
		Message:    "Delivery history retrieved successfully",
	}, nil
}

// GetNamespaceStats retrieves statistics for a namespace
func (s *WebhookServer) GetNamespaceStats(ctx context.Context, req *pb.GetNamespaceStatsRequest) (*pb.GetNamespaceStatsResponse, error) {
	stats, err := s.service.GetNamespaceStats(ctx, req.Namespace)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get namespace stats: %v", err)
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
	return &pb.GetNamespaceStatsResponse{
		Namespace: req.Namespace,
		Stats:     pbStats,
		Success:   true,
		Message:   "Namespace stats retrieved successfully",
	}, nil
}

// UpdateWebhookConfig updates webhook configuration
func (s *WebhookServer) UpdateWebhookConfig(ctx context.Context, req *pb.UpdateWebhookConfigRequest) (*pb.UpdateWebhookConfigResponse, error) {
	var events []string
	var url string
	var headers map[string]string
	var timeout int
	var active bool
	var description string
	if req.Updates != nil {
		events = req.Updates.Events
		url = req.Updates.Url
		headers = req.Updates.Headers
		timeout = int(req.Updates.Timeout)
		active = req.Updates.Active
		description = req.Updates.Description
	}
	err := s.service.UpdateWebhookConfig(ctx, req.WebhookId, req.Namespace, events, url, headers, timeout, active, description)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update webhook config: %v", err)
	}
	return &pb.UpdateWebhookConfigResponse{
		Success: true,
		Message: "Webhook config updated successfully",
	}, nil
}

// PauseWebhook temporarily disables a webhook
func (s *WebhookServer) PauseWebhook(ctx context.Context, req *pb.PauseWebhookRequest) (*pb.PauseWebhookResponse, error) {
	err := s.service.PauseWebhook(ctx, req.WebhookId, req.Namespace, req.Reason)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to pause webhook: %v", err)
	}
	return &pb.PauseWebhookResponse{
		Success: true,
		Message: "Webhook paused successfully",
	}, nil
}

// ResumeWebhook re-enables a paused webhook
func (s *WebhookServer) ResumeWebhook(ctx context.Context, req *pb.ResumeWebhookRequest) (*pb.ResumeWebhookResponse, error) {
	err := s.service.ResumeWebhook(ctx, req.WebhookId, req.Namespace)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resume webhook: %v", err)
	}
	return &pb.ResumeWebhookResponse{
		Success: true,
		Message: "Webhook resumed successfully",
	}, nil
}

// Helper function to convert delivery status
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
		return pb.WebhookDeliveryStatus_DELIVERY_UNSPECIFIED
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
		return pb.WebhookHealth_HEALTH_UNSPECIFIED
	default:
		return pb.WebhookHealth_HEALTH_UNSPECIFIED
	}
}

// ListEventReports lists all events in descending order for a given namespace
func (s *WebhookServer) ListEventReports(ctx context.Context, req *pb.ListEventReportsRequest) (*pb.ListEventReportsResponse, error) {
	// Validate request
	if req.Namespace == "" {
		return nil, status.Errorf(codes.InvalidArgument, "namespace is required")
	}

	// Set default values
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	} else if limit > 1000 {
		limit = 1000
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	// Convert optional event name
	var eventName *string
	if req.EventName != nil {
		eventName = req.EventName
	}

	// Call service method
	events, totalCount, err := s.service.ListEventReports(ctx, req.Namespace, eventName, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list event reports: %v", err)
	}

	// Convert events to protobuf format
	var pbEvents []*pb.EventReport
	for _, event := range events {
		// Convert payload to protobuf Struct
		payloadStruct, err := convertMapToStruct(event.Payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert payload: %v", err)
		}

		// Convert metadata to map[string]string
		metadata := make(map[string]string)
		for k, v := range event.Metadata {
			metadata[k] = v
		}

		// Use the delivery statistics from the database query
		pbEvent := &pb.EventReport{
			EventId:              event.ID,
			Namespace:            event.Namespace,
			EventName:            event.Event,
			Payload:              payloadStruct,
			Metadata:             metadata,
			CreatedAt:            event.CreatedAt.Unix(),
			TtlSeconds:           event.TTL,
			WebhookCount:         event.WebhookCount,
			SuccessfulDeliveries: event.SuccessfulDeliveries,
			FailedDeliveries:     event.FailedDeliveries,
			PendingDeliveries:    event.PendingDeliveries,
		}
		pbEvents = append(pbEvents, pbEvent)
	}

	return &pb.ListEventReportsResponse{
		Events:     pbEvents,
		TotalCount: totalCount,
		Success:    true,
		Message:    "Event reports retrieved successfully",
	}, nil
}

// Helper function to convert map[string]any to protobuf Struct
func convertMapToStruct(m map[string]any) (*structpb.Struct, error) {
	if m == nil {
		return nil, nil
	}

	return structpb.NewStruct(m)
}
