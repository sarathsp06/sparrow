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
