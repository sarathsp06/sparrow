package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	pb "github.com/sarathsp06/sparrow/proto"
)

// CreateSubscription creates a new event subscription for a webhook
func (s *WebhookServer) CreateSubscription(ctx context.Context, req *pb.CreateSubscriptionRequest) (*pb.CreateSubscriptionResponse, error) {
	subscriptionID, createdAt, err := s.service.CreateSubscription(ctx, req.WebhookId, req.EventName, req.Namespace, req.Headers, req.Method, int(req.Timeout), req.TransformEnabled, req.TransformTemplate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create subscription: %v", err)
	}

	return &pb.CreateSubscriptionResponse{
		SubscriptionId: subscriptionID,
		CreatedAt:      convertTimeToProto(createdAt),
	}, nil
}

// GetSubscription retrieves a specific subscription by ID
func (s *WebhookServer) GetSubscription(ctx context.Context, req *pb.GetSubscriptionRequest) (*pb.GetSubscriptionResponse, error) {
	sub, err := s.service.GetSubscription(ctx, req.SubscriptionId, req.Namespace)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get subscription: %v", err)
	}
	if sub == nil {
		return nil, status.Error(codes.NotFound, "subscription not found")
	}

	return &pb.GetSubscriptionResponse{
		Subscription: convertSubscriptionToProto(sub),
	}, nil
}

// ListSubscriptions lists all subscriptions for a webhook
func (s *WebhookServer) ListSubscriptions(ctx context.Context, req *pb.ListSubscriptionsRequest) (*pb.ListSubscriptionsResponse, error) {
	var limit, offset int32
	if req.Pagination != nil {
		limit = req.Pagination.Limit
		offset = req.Pagination.Offset
	}

	subs, totalCount, err := s.service.ListSubscriptions(ctx, req.Namespace, req.WebhookId, req.EventName, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list subscriptions: %v", err)
	}

	pbSubs := make([]*pb.EventSubscription, len(subs))
	for i, sub := range subs {
		pbSubs[i] = convertSubscriptionToProto(sub)
	}

	return &pb.ListSubscriptionsResponse{
		Subscriptions: pbSubs,
		Pagination: &pb.PaginationResponse{
			TotalCount: totalCount,
			Limit:      limit,
			Offset:     offset,
		},
	}, nil
}

// UpdateSubscription updates an existing subscription
func (s *WebhookServer) UpdateSubscription(ctx context.Context, req *pb.UpdateSubscriptionRequest) (*pb.UpdateSubscriptionResponse, error) {
	err := s.service.UpdateSubscription(ctx, req.SubscriptionId, req.Namespace, req.Headers, req.Method, int(req.Timeout), req.TransformEnabled, req.TransformTemplate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update subscription: %v", err)
	}

	return &pb.UpdateSubscriptionResponse{}, nil
}

// DeleteSubscription deletes a subscription
func (s *WebhookServer) DeleteSubscription(ctx context.Context, req *pb.DeleteSubscriptionRequest) (*pb.DeleteSubscriptionResponse, error) {
	err := s.service.DeleteSubscription(ctx, req.SubscriptionId, req.Namespace)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete subscription: %v", err)
	}

	return &pb.DeleteSubscriptionResponse{}, nil
}

// TestSubscriptionTemplate dry-runs a transformation template with sample data
func (s *WebhookServer) TestSubscriptionTemplate(ctx context.Context, req *pb.TestSubscriptionTemplateRequest) (*pb.TestSubscriptionTemplateResponse, error) {
	result, err := s.service.TestSubscriptionTemplate(ctx, req.EventName, req.TransformTemplate, req.Namespace)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to test subscription template: %v", err)
	}

	return &pb.TestSubscriptionTemplateResponse{
		TransformedPayload: result,
	}, nil
}

// Helper function to convert store.EventSubscription to protobuf
func convertSubscriptionToProto(sub *store.EventSubscription) *pb.EventSubscription {
	if sub == nil {
		return nil
	}
	return &pb.EventSubscription{
		SubscriptionId:    sub.ID.String(),
		WebhookId:         sub.WebhookID.String(),
		EventName:         sub.EventName,
		Namespace:         sub.Namespace,
		Headers:           sub.Headers,
		Method:            sub.Method,
		Timeout:           int32(sub.Timeout),
		TransformEnabled:  sub.TransformEnabled,
		TransformTemplate: sub.TransformTemplate,
		CreatedAt:         convertTimeToProto(sub.CreatedAt),
		UpdatedAt:         convertTimeToProto(sub.UpdatedAt),
	}
}
