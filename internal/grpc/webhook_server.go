package grpc

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
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

// getWebhookEventsMap batch-fetches event names for multiple webhooks in a single query,
// returning a map from webhook ID string to a slice of subscribed event names.
func (s *WebhookServer) getWebhookEventsMap(ctx context.Context, webhookRegs []*store.WebhookRegistration) map[string][]string {
	if len(webhookRegs) == 0 {
		return map[string][]string{}
	}

	webhookIDs := make([]uuid.UUID, len(webhookRegs))
	for i, reg := range webhookRegs {
		webhookIDs[i] = reg.ID
	}

	tenantID := tenant.DefaultTenantID
	subs, err := s.service.GetWebhookRepo().ListSubscriptionsByWebhookIDs(ctx, tenantID, webhookIDs)
	if err != nil {
		slog.ErrorContext(ctx, "failed to batch-fetch subscriptions", "error", err)
		return map[string][]string{}
	}

	result := make(map[string][]string, len(webhookIDs))
	for _, sub := range subs {
		key := sub.WebhookID.String()
		result[key] = append(result[key], sub.EventName)
	}
	return result
}
