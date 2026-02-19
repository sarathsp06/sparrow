package grpc

import (
	"context"

	"github.com/sarathsp06/sparrow/internal/webhooks"
	pb "github.com/sarathsp06/sparrow/proto"
)

// WebhookServer implements all gRPC services
type WebhookServer struct {
	pb.UnimplementedWebhookServiceServer
	pb.UnimplementedEventServiceServer
	pb.UnimplementedSubscriptionServiceServer
	pb.UnimplementedDeliveryServiceServer
	pb.UnimplementedHealthServiceServer
	service webhooks.WebhookServiceInterface
}

var _ pb.WebhookServiceServer = (*WebhookServer)(nil)
var _ pb.EventServiceServer = (*WebhookServer)(nil)
var _ pb.SubscriptionServiceServer = (*WebhookServer)(nil)
var _ pb.DeliveryServiceServer = (*WebhookServer)(nil)
var _ pb.HealthServiceServer = (*WebhookServer)(nil)

// NewWebhookServer creates a new WebhookServer instance
func NewWebhookServer(service webhooks.WebhookServiceInterface) *WebhookServer {
	return &WebhookServer{
		service: service,
	}
}

// Helper function to get events for a webhook from subscriptions
// Used by webhook handlers in other files
func (s *WebhookServer) getWebhookEvents(ctx context.Context, webhookID string, namespace string) []string {
	subs, _, err := s.service.ListSubscriptions(ctx, namespace, webhookID, "", 1000, 0)
	if err != nil {
		return []string{}
	}
	events := make([]string, 0, len(subs))
	for _, sub := range subs {
		events = append(events, sub.EventName)
	}
	return events
}
