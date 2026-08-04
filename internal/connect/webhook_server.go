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

// NewWebhookConnectServer creates a Connect-RPC adapter over the existing gRPC
// server implementations. The generated Connect handler interfaces require one
// typed method per RPC, so this module keeps those methods as thin adapters and
// centralizes the forwarding behavior in forwardUnary.
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

type grpcUnary[Req, Resp any] func(context.Context, *Req) (*Resp, error)

func forwardUnary[Req, Resp any](ctx context.Context, req *connect.Request[Req], call grpcUnary[Req, Resp]) (*connect.Response[Resp], error) {
	res, err := call(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// WebhookService implementation.
func (s *WebhookConnectServer) RegisterWebhook(ctx context.Context, req *connect.Request[pb.RegisterWebhookRequest]) (*connect.Response[pb.RegisterWebhookResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.RegisterWebhook)
}

func (s *WebhookConnectServer) UnregisterWebhook(ctx context.Context, req *connect.Request[pb.UnregisterWebhookRequest]) (*connect.Response[pb.UnregisterWebhookResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.UnregisterWebhook)
}

func (s *WebhookConnectServer) ListWebhooks(ctx context.Context, req *connect.Request[pb.ListWebhooksRequest]) (*connect.Response[pb.ListWebhooksResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.ListWebhooks)
}

func (s *WebhookConnectServer) UpdateWebhookConfig(ctx context.Context, req *connect.Request[pb.UpdateWebhookConfigRequest]) (*connect.Response[pb.UpdateWebhookConfigResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.UpdateWebhookConfig)
}

func (s *WebhookConnectServer) PauseWebhook(ctx context.Context, req *connect.Request[pb.PauseWebhookRequest]) (*connect.Response[pb.PauseWebhookResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.PauseWebhook)
}

func (s *WebhookConnectServer) ResumeWebhook(ctx context.Context, req *connect.Request[pb.ResumeWebhookRequest]) (*connect.Response[pb.ResumeWebhookResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.ResumeWebhook)
}

func (s *WebhookConnectServer) GetNamespaceStats(ctx context.Context, req *connect.Request[pb.GetNamespaceStatsRequest]) (*connect.Response[pb.GetNamespaceStatsResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.GetNamespaceStats)
}

func (s *WebhookConnectServer) GetTemplateFunctions(ctx context.Context, req *connect.Request[pb.GetTemplateFunctionsRequest]) (*connect.Response[pb.GetTemplateFunctionsResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.GetTemplateFunctions)
}

// EventService implementation.
func (s *WebhookConnectServer) RegisterEvent(ctx context.Context, req *connect.Request[pb.RegisterEventRequest]) (*connect.Response[pb.RegisterEventResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.RegisterEvent)
}

func (s *WebhookConnectServer) ListEvents(ctx context.Context, req *connect.Request[pb.ListEventsRequest]) (*connect.Response[pb.ListEventsResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.ListEvents)
}

func (s *WebhookConnectServer) UpdateEvent(ctx context.Context, req *connect.Request[pb.UpdateEventRequest]) (*connect.Response[pb.UpdateEventResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.UpdateEvent)
}

func (s *WebhookConnectServer) DeleteEvent(ctx context.Context, req *connect.Request[pb.DeleteEventRequest]) (*connect.Response[pb.DeleteEventResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.DeleteEvent)
}

func (s *WebhookConnectServer) PushEvent(ctx context.Context, req *connect.Request[pb.PushEventRequest]) (*connect.Response[pb.PushEventResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.PushEvent)
}

func (s *WebhookConnectServer) GetEvent(ctx context.Context, req *connect.Request[pb.GetEventRequest]) (*connect.Response[pb.GetEventResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.GetEvent)
}

func (s *WebhookConnectServer) GetEventRecord(ctx context.Context, req *connect.Request[pb.GetEventRecordRequest]) (*connect.Response[pb.GetEventRecordResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.GetEventRecord)
}

func (s *WebhookConnectServer) ListEventReports(ctx context.Context, req *connect.Request[pb.ListEventReportsRequest]) (*connect.Response[pb.ListEventReportsResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.ListEventReports)
}

func (s *WebhookConnectServer) RePushEvent(ctx context.Context, req *connect.Request[pb.RePushEventRequest]) (*connect.Response[pb.RePushEventResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.RePushEvent)
}

func (s *WebhookConnectServer) RePushEvents(ctx context.Context, req *connect.Request[pb.RePushEventsRequest]) (*connect.Response[pb.RePushEventsResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.RePushEvents)
}

func (s *WebhookConnectServer) GetRepushStatus(ctx context.Context, req *connect.Request[pb.GetRepushStatusRequest]) (*connect.Response[pb.GetRepushStatusResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.GetRepushStatus)
}

func (s *WebhookConnectServer) CancelRepush(ctx context.Context, req *connect.Request[pb.CancelRepushRequest]) (*connect.Response[pb.CancelRepushResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.CancelRepush)
}

// SubscriptionService implementation.
func (s *WebhookConnectServer) CreateSubscription(ctx context.Context, req *connect.Request[pb.CreateSubscriptionRequest]) (*connect.Response[pb.CreateSubscriptionResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.CreateSubscription)
}

func (s *WebhookConnectServer) GetSubscription(ctx context.Context, req *connect.Request[pb.GetSubscriptionRequest]) (*connect.Response[pb.GetSubscriptionResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.GetSubscription)
}

func (s *WebhookConnectServer) ListSubscriptions(ctx context.Context, req *connect.Request[pb.ListSubscriptionsRequest]) (*connect.Response[pb.ListSubscriptionsResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.ListSubscriptions)
}

func (s *WebhookConnectServer) UpdateSubscription(ctx context.Context, req *connect.Request[pb.UpdateSubscriptionRequest]) (*connect.Response[pb.UpdateSubscriptionResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.UpdateSubscription)
}

func (s *WebhookConnectServer) DeleteSubscription(ctx context.Context, req *connect.Request[pb.DeleteSubscriptionRequest]) (*connect.Response[pb.DeleteSubscriptionResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.DeleteSubscription)
}

func (s *WebhookConnectServer) TestSubscriptionTemplate(ctx context.Context, req *connect.Request[pb.TestSubscriptionTemplateRequest]) (*connect.Response[pb.TestSubscriptionTemplateResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.TestSubscriptionTemplate)
}

// DeliveryService implementation.
func (s *WebhookConnectServer) GetDeliveryStatus(ctx context.Context, req *connect.Request[pb.GetDeliveryStatusRequest]) (*connect.Response[pb.GetDeliveryStatusResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.GetDeliveryStatus)
}

func (s *WebhookConnectServer) ListDeliveries(ctx context.Context, req *connect.Request[pb.ListDeliveriesRequest]) (*connect.Response[pb.ListDeliveriesResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.ListDeliveries)
}

func (s *WebhookConnectServer) RetryDelivery(ctx context.Context, req *connect.Request[pb.RetryDeliveryRequest]) (*connect.Response[pb.RetryDeliveryResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.RetryDelivery)
}

func (s *WebhookConnectServer) GetDeliveryAttempts(ctx context.Context, req *connect.Request[pb.GetDeliveryAttemptsRequest]) (*connect.Response[pb.GetDeliveryAttemptsResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.GetDeliveryAttempts)
}

func (s *WebhookConnectServer) RetryDeliveries(ctx context.Context, req *connect.Request[pb.RetryDeliveriesRequest]) (*connect.Response[pb.RetryDeliveriesResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.RetryDeliveries)
}

func (s *WebhookConnectServer) GetRetryStatus(ctx context.Context, req *connect.Request[pb.GetRetryStatusRequest]) (*connect.Response[pb.GetRetryStatusResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.GetRetryStatus)
}

func (s *WebhookConnectServer) CancelRetry(ctx context.Context, req *connect.Request[pb.CancelRetryRequest]) (*connect.Response[pb.CancelRetryResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.CancelRetry)
}

// HealthService implementation.
func (s *WebhookConnectServer) GetWebhookHealth(ctx context.Context, req *connect.Request[pb.GetWebhookHealthRequest]) (*connect.Response[pb.GetWebhookHealthResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.GetWebhookHealth)
}

func (s *WebhookConnectServer) ListWebhooksByHealth(ctx context.Context, req *connect.Request[pb.ListWebhooksByHealthRequest]) (*connect.Response[pb.ListWebhooksByHealthResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.ListWebhooksByHealth)
}

func (s *WebhookConnectServer) GetHealthSummary(ctx context.Context, req *connect.Request[pb.GetHealthSummaryRequest]) (*connect.Response[pb.GetHealthSummaryResponse], error) {
	return forwardUnary(ctx, req, s.grpcService.GetHealthSummary)
}

var _ pbconnect.WebhookServiceHandler = (*WebhookConnectServer)(nil)
var _ pbconnect.EventServiceHandler = (*WebhookConnectServer)(nil)
var _ pbconnect.SubscriptionServiceHandler = (*WebhookConnectServer)(nil)
var _ pbconnect.DeliveryServiceHandler = (*WebhookConnectServer)(nil)
var _ pbconnect.HealthServiceHandler = (*WebhookConnectServer)(nil)
