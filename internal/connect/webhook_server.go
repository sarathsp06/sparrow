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
