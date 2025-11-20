package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	pb "github.com/sarathsp06/sparrow/proto"
)

// CreateSubscription creates a new event subscription for a webhook
func (s *WebhookServer) CreateSubscription(ctx context.Context, req *pb.CreateSubscriptionRequest) (*pb.CreateSubscriptionResponse, error) {
	// Validate request
	if req.WebhookId == "" {
		return nil, status.Error(codes.InvalidArgument, "webhook_id is required")
	}
	if req.EventName == "" {
		return nil, status.Error(codes.InvalidArgument, "event_name is required")
	}
	if req.Namespace == "" {
		return nil, status.Error(codes.InvalidArgument, "namespace is required")
	}

	// Validate template if transformation is enabled
	if req.TransformEnabled && req.TransformTemplate != "" {
		templateEngine := client.NewTemplateEngine()
		if err := templateEngine.ValidateTemplateWithTestData(req.TransformTemplate); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid template: %v", err)
		}
	}

	// Access repository through service interface
	repo := s.service.GetWebhookRepo()

	// Create subscription
	sub := &store.EventSubscription{
		WebhookID:         req.WebhookId,
		EventName:         req.EventName,
		Namespace:         req.Namespace,
		Headers:           req.Headers,
		Method:            req.Method,
		Timeout:           int(req.Timeout),
		TransformEnabled:  req.TransformEnabled,
		TransformTemplate: req.TransformTemplate,
	}

	if err := repo.CreateSubscription(ctx, sub); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create subscription: %v", err)
	}

	return &pb.CreateSubscriptionResponse{
		SubscriptionId: sub.ID,
		Success:        true,
		Message:        "Subscription created successfully",
		CreatedAt:      sub.CreatedAt.Unix(),
	}, nil
}

// GetSubscription retrieves a specific subscription by ID
func (s *WebhookServer) GetSubscription(ctx context.Context, req *pb.GetSubscriptionRequest) (*pb.GetSubscriptionResponse, error) {
	if req.SubscriptionId == "" {
		return nil, status.Error(codes.InvalidArgument, "subscription_id is required")
	}

	repo := s.service.GetWebhookRepo()

	sub, err := repo.GetSubscription(ctx, req.SubscriptionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get subscription: %v", err)
	}
	if sub == nil {
		return nil, status.Error(codes.NotFound, "subscription not found")
	}

	return &pb.GetSubscriptionResponse{
		Subscription: convertSubscriptionToProto(sub),
		Success:      true,
		Message:      "Subscription retrieved successfully",
	}, nil
}

// ListSubscriptions lists all subscriptions for a webhook
func (s *WebhookServer) ListSubscriptions(ctx context.Context, req *pb.ListSubscriptionsRequest) (*pb.ListSubscriptionsResponse, error) {
	if req.WebhookId == "" {
		return nil, status.Error(codes.InvalidArgument, "webhook_id is required")
	}

	repo := s.service.GetWebhookRepo()

	subs, err := repo.ListSubscriptions(ctx, req.WebhookId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list subscriptions: %v", err)
	}

	pbSubs := make([]*pb.EventSubscription, len(subs))
	for i, sub := range subs {
		pbSubs[i] = convertSubscriptionToProto(sub)
	}

	return &pb.ListSubscriptionsResponse{
		Subscriptions: pbSubs,
		TotalCount:    int32(len(pbSubs)),
		Success:       true,
		Message:       "Subscriptions listed successfully",
	}, nil
}

// UpdateSubscription updates an existing subscription
func (s *WebhookServer) UpdateSubscription(ctx context.Context, req *pb.UpdateSubscriptionRequest) (*pb.UpdateSubscriptionResponse, error) {
	if req.SubscriptionId == "" {
		return nil, status.Error(codes.InvalidArgument, "subscription_id is required")
	}

	// Validate template if transformation is enabled
	if req.TransformEnabled && req.TransformTemplate != "" {
		templateEngine := client.NewTemplateEngine()
		if err := templateEngine.ValidateTemplateWithTestData(req.TransformTemplate); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid template: %v", err)
		}
	}

	repo := s.service.GetWebhookRepo()

	// Get existing subscription
	sub, err := repo.GetSubscription(ctx, req.SubscriptionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get subscription: %v", err)
	}
	if sub == nil {
		return nil, status.Error(codes.NotFound, "subscription not found")
	}

	// Update fields
	if req.Headers != nil {
		sub.Headers = req.Headers
	}
	if req.Method != "" {
		sub.Method = req.Method
	}
	if req.Timeout > 0 {
		sub.Timeout = int(req.Timeout)
	}
	sub.TransformEnabled = req.TransformEnabled
	if req.TransformTemplate != "" {
		sub.TransformTemplate = req.TransformTemplate
	}

	if err := repo.UpdateSubscription(ctx, sub); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update subscription: %v", err)
	}

	return &pb.UpdateSubscriptionResponse{
		Success: true,
		Message: "Subscription updated successfully",
	}, nil
}

// DeleteSubscription deletes a subscription
func (s *WebhookServer) DeleteSubscription(ctx context.Context, req *pb.DeleteSubscriptionRequest) (*pb.DeleteSubscriptionResponse, error) {
	if req.SubscriptionId == "" {
		return nil, status.Error(codes.InvalidArgument, "subscription_id is required")
	}

	repo := s.service.GetWebhookRepo()

	if err := repo.DeleteSubscription(ctx, req.SubscriptionId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete subscription: %v", err)
	}

	return &pb.DeleteSubscriptionResponse{
		Success: true,
		Message: "Subscription deleted successfully",
	}, nil
}

// ListSubscriptionsByEvent lists all subscriptions for a specific event
func (s *WebhookServer) ListSubscriptionsByEvent(ctx context.Context, req *pb.ListSubscriptionsByEventRequest) (*pb.ListSubscriptionsByEventResponse, error) {
	if req.Namespace == "" {
		return nil, status.Error(codes.InvalidArgument, "namespace is required")
	}
	if req.EventName == "" {
		return nil, status.Error(codes.InvalidArgument, "event_name is required")
	}

	repo := s.service.GetWebhookRepo()

	subs, err := repo.GetSubscriptionsByEvent(ctx, req.Namespace, req.EventName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list subscriptions: %v", err)
	}

	pbSubs := make([]*pb.EventSubscription, len(subs))
	for i, sub := range subs {
		pbSubs[i] = convertSubscriptionToProto(sub)
	}

	return &pb.ListSubscriptionsByEventResponse{
		Subscriptions: pbSubs,
		TotalCount:    int32(len(pbSubs)),
		Success:       true,
		Message:       "Subscriptions listed successfully",
	}, nil
}

// Helper function to convert store.EventSubscription to protobuf
func convertSubscriptionToProto(sub *store.EventSubscription) *pb.EventSubscription {
	return &pb.EventSubscription{
		SubscriptionId:    sub.ID,
		WebhookId:         sub.WebhookID,
		EventName:         sub.EventName,
		Namespace:         sub.Namespace,
		Headers:           sub.Headers,
		Method:            sub.Method,
		Timeout:           int32(sub.Timeout),
		TransformEnabled:  sub.TransformEnabled,
		TransformTemplate: sub.TransformTemplate,
		CreatedAt:         sub.CreatedAt.Unix(),
		UpdatedAt:         sub.UpdatedAt.Unix(),
	}
}
