package connect

import (
	"context"

	"connectrpc.com/connect"

	pb "github.com/sarathsp06/sparrow/proto"
	pbconnect "github.com/sarathsp06/sparrow/proto/protoconnect"
)

type WebhookConnectServer struct {
	grpcService *grpcServerWrapper
}

type grpcServerWrapper struct {
	pb.WebhookServiceServer
	pb.EventServiceServer
	pb.SubscriptionServiceServer
	pb.DeliveryServiceServer
	pb.HealthServiceServer
}

// NewWebhookConnectServer creates a new Connect-RPC server
func NewWebhookConnectServer(
	webhookService pb.WebhookServiceServer,
	eventService pb.EventServiceServer,
	subscriptionService pb.SubscriptionServiceServer,
	deliveryService pb.DeliveryServiceServer,
	healthService pb.HealthServiceServer,
) *WebhookConnectServer {
	return &WebhookConnectServer{
		grpcService: &grpcServerWrapper{
			WebhookServiceServer:      webhookService,
			EventServiceServer:        eventService,
			SubscriptionServiceServer: subscriptionService,
			DeliveryServiceServer:     deliveryService,
			HealthServiceServer:       healthService,
		},
	}
}

// WebhookService Implementation
func (s *WebhookConnectServer) RegisterWebhook(ctx context.Context, req *connect.Request[pb.RegisterWebhookRequest]) (*connect.Response[pb.RegisterWebhookResponse], error) {
	res, err := s.grpcService.RegisterWebhook(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) UnregisterWebhook(ctx context.Context, req *connect.Request[pb.UnregisterWebhookRequest]) (*connect.Response[pb.UnregisterWebhookResponse], error) {
	res, err := s.grpcService.UnregisterWebhook(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) ListWebhooks(ctx context.Context, req *connect.Request[pb.ListWebhooksRequest]) (*connect.Response[pb.ListWebhooksResponse], error) {
	res, err := s.grpcService.ListWebhooks(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) UpdateWebhookConfig(ctx context.Context, req *connect.Request[pb.UpdateWebhookConfigRequest]) (*connect.Response[pb.UpdateWebhookConfigResponse], error) {
	res, err := s.grpcService.UpdateWebhookConfig(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) PauseWebhook(ctx context.Context, req *connect.Request[pb.PauseWebhookRequest]) (*connect.Response[pb.PauseWebhookResponse], error) {
	res, err := s.grpcService.PauseWebhook(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) ResumeWebhook(ctx context.Context, req *connect.Request[pb.ResumeWebhookRequest]) (*connect.Response[pb.ResumeWebhookResponse], error) {
	res, err := s.grpcService.ResumeWebhook(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) GetNamespaceStats(ctx context.Context, req *connect.Request[pb.GetNamespaceStatsRequest]) (*connect.Response[pb.GetNamespaceStatsResponse], error) {
	res, err := s.grpcService.GetNamespaceStats(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) GetTemplateFunctions(ctx context.Context, req *connect.Request[pb.GetTemplateFunctionsRequest]) (*connect.Response[pb.GetTemplateFunctionsResponse], error) {
	res, err := s.grpcService.GetTemplateFunctions(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// EventService Implementation
func (s *WebhookConnectServer) RegisterEvent(ctx context.Context, req *connect.Request[pb.RegisterEventRequest]) (*connect.Response[pb.RegisterEventResponse], error) {
	res, err := s.grpcService.RegisterEvent(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) ListEvents(ctx context.Context, req *connect.Request[pb.ListEventsRequest]) (*connect.Response[pb.ListEventsResponse], error) {
	res, err := s.grpcService.ListEvents(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) UpdateEvent(ctx context.Context, req *connect.Request[pb.UpdateEventRequest]) (*connect.Response[pb.UpdateEventResponse], error) {
	res, err := s.grpcService.UpdateEvent(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) DeleteEvent(ctx context.Context, req *connect.Request[pb.DeleteEventRequest]) (*connect.Response[pb.DeleteEventResponse], error) {
	res, err := s.grpcService.DeleteEvent(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) PushEvent(ctx context.Context, req *connect.Request[pb.PushEventRequest]) (*connect.Response[pb.PushEventResponse], error) {
	res, err := s.grpcService.PushEvent(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) GetEvent(ctx context.Context, req *connect.Request[pb.GetEventRequest]) (*connect.Response[pb.GetEventResponse], error) {
	res, err := s.grpcService.GetEvent(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) ListEventReports(ctx context.Context, req *connect.Request[pb.ListEventReportsRequest]) (*connect.Response[pb.ListEventReportsResponse], error) {
	res, err := s.grpcService.ListEventReports(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// SubscriptionService Implementation
func (s *WebhookConnectServer) CreateSubscription(ctx context.Context, req *connect.Request[pb.CreateSubscriptionRequest]) (*connect.Response[pb.CreateSubscriptionResponse], error) {
	res, err := s.grpcService.CreateSubscription(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) GetSubscription(ctx context.Context, req *connect.Request[pb.GetSubscriptionRequest]) (*connect.Response[pb.GetSubscriptionResponse], error) {
	res, err := s.grpcService.GetSubscription(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) ListSubscriptions(ctx context.Context, req *connect.Request[pb.ListSubscriptionsRequest]) (*connect.Response[pb.ListSubscriptionsResponse], error) {
	res, err := s.grpcService.ListSubscriptions(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) UpdateSubscription(ctx context.Context, req *connect.Request[pb.UpdateSubscriptionRequest]) (*connect.Response[pb.UpdateSubscriptionResponse], error) {
	res, err := s.grpcService.UpdateSubscription(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) DeleteSubscription(ctx context.Context, req *connect.Request[pb.DeleteSubscriptionRequest]) (*connect.Response[pb.DeleteSubscriptionResponse], error) {
	res, err := s.grpcService.DeleteSubscription(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) TestSubscriptionTemplate(ctx context.Context, req *connect.Request[pb.TestSubscriptionTemplateRequest]) (*connect.Response[pb.TestSubscriptionTemplateResponse], error) {
	res, err := s.grpcService.TestSubscriptionTemplate(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// DeliveryService Implementation
func (s *WebhookConnectServer) GetDeliveryStatus(ctx context.Context, req *connect.Request[pb.GetDeliveryStatusRequest]) (*connect.Response[pb.GetDeliveryStatusResponse], error) {
	res, err := s.grpcService.GetDeliveryStatus(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) ListDeliveries(ctx context.Context, req *connect.Request[pb.ListDeliveriesRequest]) (*connect.Response[pb.ListDeliveriesResponse], error) {
	res, err := s.grpcService.ListDeliveries(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) RetryDelivery(ctx context.Context, req *connect.Request[pb.RetryDeliveryRequest]) (*connect.Response[pb.RetryDeliveryResponse], error) {
	res, err := s.grpcService.RetryDelivery(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) GetDeliveryAttempts(ctx context.Context, req *connect.Request[pb.GetDeliveryAttemptsRequest]) (*connect.Response[pb.GetDeliveryAttemptsResponse], error) {
	res, err := s.grpcService.GetDeliveryAttempts(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// HealthService Implementation
func (s *WebhookConnectServer) GetWebhookHealth(ctx context.Context, req *connect.Request[pb.GetWebhookHealthRequest]) (*connect.Response[pb.GetWebhookHealthResponse], error) {
	res, err := s.grpcService.GetWebhookHealth(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) ListWebhooksByHealth(ctx context.Context, req *connect.Request[pb.ListWebhooksByHealthRequest]) (*connect.Response[pb.ListWebhooksByHealthResponse], error) {
	res, err := s.grpcService.ListWebhooksByHealth(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *WebhookConnectServer) GetHealthSummary(ctx context.Context, req *connect.Request[pb.GetHealthSummaryRequest]) (*connect.Response[pb.GetHealthSummaryResponse], error) {
	res, err := s.grpcService.GetHealthSummary(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

var _ pbconnect.WebhookServiceHandler = (*WebhookConnectServer)(nil)
var _ pbconnect.EventServiceHandler = (*WebhookConnectServer)(nil)
var _ pbconnect.SubscriptionServiceHandler = (*WebhookConnectServer)(nil)
var _ pbconnect.DeliveryServiceHandler = (*WebhookConnectServer)(nil)
var _ pbconnect.HealthServiceHandler = (*WebhookConnectServer)(nil)
