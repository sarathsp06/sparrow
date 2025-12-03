package grpc

import (
	"context"

	"github.com/google/uuid"
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

// Helper function to get events for a webhook from subscriptions
// Used by webhook handlers in other files
func (s *WebhookServer) getWebhookEvents(ctx context.Context, webhookID string) []string {
	// Access repository through service interface
	serviceWithRepo, ok := s.service.(interface {
		GetWebhookRepo() store.RepositoryInterface
	})
	if !ok {
		return []string{}
	}

	subs, err := serviceWithRepo.GetWebhookRepo().ListSubscriptions(ctx, uuid.MustParse(webhookID))
	if err != nil {
		return []string{}
	}
	events := make([]string, 0, len(subs))
	for _, sub := range subs {
		events = append(events, sub.EventName)
	}
	return events
}
