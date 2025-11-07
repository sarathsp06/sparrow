package connect

import (
	"context"

	"connectrpc.com/connect"

	pb "github.com/sarathsp06/sparrow/proto"
	pbconnect "github.com/sarathsp06/sparrow/proto/protoconnect"
)

type WebhookConnectServer struct {
	grpcService pb.WebhookServiceServer
}

var _ pbconnect.WebhookServiceHandler = (*WebhookConnectServer)(nil)

// NewWebhookConnectServer creates a new Connect-RPC server
func NewWebhookConnectServer(grpcService pb.WebhookServiceServer) *WebhookConnectServer {
	return &WebhookConnectServer{
		grpcService: grpcService,
	}
}

// ListEventReports
func (s *WebhookConnectServer) ListEventReports(ctx context.Context, req *connect.Request[pb.ListEventReportsRequest]) (*connect.Response[pb.ListEventReportsResponse], error) {
	res, err := s.grpcService.ListEventReports(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// RegisterWebhook registers a URL for specific events in a namespace
func (s *WebhookConnectServer) RegisterWebhook(ctx context.Context, req *connect.Request[pb.RegisterWebhookRequest]) (*connect.Response[pb.RegisterWebhookResponse], error) {
	res, err := s.grpcService.RegisterWebhook(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// UnregisterWebhook removes a webhook registration
func (s *WebhookConnectServer) UnregisterWebhook(ctx context.Context, req *connect.Request[pb.UnregisterWebhookRequest]) (*connect.Response[pb.UnregisterWebhookResponse], error) {
	res, err := s.grpcService.UnregisterWebhook(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// PushEvent pushes an event that triggers registered webhooks
func (s *WebhookConnectServer) PushEvent(ctx context.Context, req *connect.Request[pb.PushEventRequest]) (*connect.Response[pb.PushEventResponse], error) {
	res, err := s.grpcService.PushEvent(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// GetWebhookStatus gets the status of webhook deliveries
func (s *WebhookConnectServer) GetWebhookStatus(ctx context.Context, req *connect.Request[pb.GetWebhookStatusRequest]) (*connect.Response[pb.GetWebhookStatusResponse], error) {
	res, err := s.grpcService.GetWebhookStatus(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// ListWebhooks lists all registered webhooks for a namespace
func (s *WebhookConnectServer) ListWebhooks(ctx context.Context, req *connect.Request[pb.ListWebhooksRequest]) (*connect.Response[pb.ListWebhooksResponse], error) {
	res, err := s.grpcService.ListWebhooks(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// RegisterEvent registers a new event type (no namespace required)
func (s *WebhookConnectServer) RegisterEvent(ctx context.Context, req *connect.Request[pb.RegisterEventRequest]) (*connect.Response[pb.RegisterEventResponse], error) {
	res, err := s.grpcService.RegisterEvent(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// ListEvents lists all registered event types
func (s *WebhookConnectServer) ListEvents(ctx context.Context, req *connect.Request[pb.ListEventsRequest]) (*connect.Response[pb.ListEventsResponse], error) {
	res, err := s.grpcService.ListEvents(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// UpdateEvent updates an event registration
func (s *WebhookConnectServer) UpdateEvent(ctx context.Context, req *connect.Request[pb.UpdateEventRequest]) (*connect.Response[pb.UpdateEventResponse], error) {
	res, err := s.grpcService.UpdateEvent(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// DeleteEvent deletes an event registration
func (s *WebhookConnectServer) DeleteEvent(ctx context.Context, req *connect.Request[pb.DeleteEventRequest]) (*connect.Response[pb.DeleteEventResponse], error) {
	res, err := s.grpcService.DeleteEvent(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// GetWebhookHealth gets health metrics for a specific webhook
func (s *WebhookConnectServer) GetWebhookHealth(ctx context.Context, req *connect.Request[pb.GetWebhookHealthRequest]) (*connect.Response[pb.GetWebhookHealthResponse], error) {
	res, err := s.grpcService.GetWebhookHealth(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// ListWebhooksByHealth lists webhooks filtered by health status
func (s *WebhookConnectServer) ListWebhooksByHealth(ctx context.Context, req *connect.Request[pb.ListWebhooksByHealthRequest]) (*connect.Response[pb.ListWebhooksByHealthResponse], error) {
	res, err := s.grpcService.ListWebhooksByHealth(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// GetHealthSummary gets a summary of webhook health across all namespaces
func (s *WebhookConnectServer) GetHealthSummary(ctx context.Context, req *connect.Request[pb.GetHealthSummaryRequest]) (*connect.Response[pb.GetHealthSummaryResponse], error) {
	res, err := s.grpcService.GetHealthSummary(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// ResubmitWebhook manually retries failed or pending webhook deliveries
func (s *WebhookConnectServer) ResubmitWebhook(ctx context.Context, req *connect.Request[pb.ResubmitWebhookRequest]) (*connect.Response[pb.ResubmitWebhookResponse], error) {
	res, err := s.grpcService.ResubmitWebhook(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// GetRegisteredWebhooks retrieves registered webhooks by ID or namespace
func (s *WebhookConnectServer) GetRegisteredWebhooks(ctx context.Context, req *connect.Request[pb.GetRegisteredWebhooksRequest]) (*connect.Response[pb.GetRegisteredWebhooksResponse], error) {
	res, err := s.grpcService.GetRegisteredWebhooks(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// ListRegisteredWebhooksByEvent retrieves webhooks registered for specific events
func (s *WebhookConnectServer) ListRegisteredWebhooksByEvent(ctx context.Context, req *connect.Request[pb.ListRegisteredWebhooksByEventRequest]) (*connect.Response[pb.ListRegisteredWebhooksByEventResponse], error) {
	res, err := s.grpcService.ListRegisteredWebhooksByEvent(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// GetWebhookDeliveryStatus retrieves delivery status for specific delivery
func (s *WebhookConnectServer) GetWebhookDeliveryStatus(ctx context.Context, req *connect.Request[pb.GetWebhookDeliveryStatusRequest]) (*connect.Response[pb.GetWebhookDeliveryStatusResponse], error) {
	res, err := s.grpcService.GetWebhookDeliveryStatus(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// ResendWebhook resends a failed webhook delivery
func (s *WebhookConnectServer) ResendWebhook(ctx context.Context, req *connect.Request[pb.ResendWebhookRequest]) (*connect.Response[pb.ResendWebhookResponse], error) {
	res, err := s.grpcService.ResendWebhook(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// GetWebhookDeliveryHistory retrieves delivery history for a webhook
func (s *WebhookConnectServer) GetWebhookDeliveryHistory(ctx context.Context, req *connect.Request[pb.GetWebhookDeliveryHistoryRequest]) (*connect.Response[pb.GetWebhookDeliveryHistoryResponse], error) {
	res, err := s.grpcService.GetWebhookDeliveryHistory(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// GetNamespaceStats retrieves statistics for a namespace
func (s *WebhookConnectServer) GetNamespaceStats(ctx context.Context, req *connect.Request[pb.GetNamespaceStatsRequest]) (*connect.Response[pb.GetNamespaceStatsResponse], error) {
	res, err := s.grpcService.GetNamespaceStats(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// UpdateWebhookConfig updates webhook configuration
func (s *WebhookConnectServer) UpdateWebhookConfig(ctx context.Context, req *connect.Request[pb.UpdateWebhookConfigRequest]) (*connect.Response[pb.UpdateWebhookConfigResponse], error) {
	res, err := s.grpcService.UpdateWebhookConfig(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// PauseWebhook temporarily disables a webhook
func (s *WebhookConnectServer) PauseWebhook(ctx context.Context, req *connect.Request[pb.PauseWebhookRequest]) (*connect.Response[pb.PauseWebhookResponse], error) {
	res, err := s.grpcService.PauseWebhook(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// ResumeWebhook re-enables a paused webhook
func (s *WebhookConnectServer) ResumeWebhook(ctx context.Context, req *connect.Request[pb.ResumeWebhookRequest]) (*connect.Response[pb.ResumeWebhookResponse], error) {
	res, err := s.grpcService.ResumeWebhook(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}
