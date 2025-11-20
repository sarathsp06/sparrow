package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/sarathsp06/sparrow/proto"
)

// GetWebhookStatus gets the status of webhook deliveries
func (s *WebhookServer) GetWebhookStatus(ctx context.Context, req *pb.GetWebhookStatusRequest) (*pb.GetWebhookStatusResponse, error) {
	if req.GetWebhookId() == "" {
		return nil, status.Error(codes.InvalidArgument, "webhook_id is required")
	}
	if req.GetNamespace() == "" {
		return nil, status.Error(codes.InvalidArgument, "namespace is required")
	}
	deliveries, totalDeliveries, err := s.service.GetWebhookStatus(ctx, req.GetNamespace(), req.GetWebhookId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get webhook status: %v", err)
	}
	pbDeliveries := make([]*pb.WebhookDelivery, len(deliveries))
	for i, d := range deliveries {
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
			RequestBody:  d.RequestBody,
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
		TotalDeliveries: totalDeliveries,
		Success:         true,
		Message:         "Webhook status retrieved successfully",
	}, nil
}

// GetWebhookDeliveryStatus retrieves delivery status for specific delivery
func (s *WebhookServer) GetWebhookDeliveryStatus(ctx context.Context, req *pb.GetWebhookDeliveryStatusRequest) (*pb.GetWebhookDeliveryStatusResponse, error) {
	delivery, err := s.service.GetWebhookDeliveryStatus(ctx, req.DeliveryId, req.Namespace)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get delivery status: %v", err)
	}
	var pbDelivery *pb.WebhookDelivery
	if delivery != nil {
		pbDelivery = &pb.WebhookDelivery{
			DeliveryId:   delivery.ID,
			WebhookId:    delivery.WebhookID,
			EventId:      delivery.EventID,
			Status:       convertDeliveryStatus(delivery.Status),
			AttemptCount: int32(delivery.AttemptCount),
			MaxAttempts:  int32(delivery.MaxAttempts),
			CreatedAt:    delivery.CreatedAt.Unix(),
			ExpiresAt:    delivery.ExpiresAt.Unix(),
			ResponseCode: int32(delivery.ResponseCode),
			ResponseBody: delivery.ResponseBody,
			ErrorMessage: delivery.ErrorMessage,
			RequestBody:  delivery.RequestBody,
		}
		if delivery.LastAttemptedAt != nil {
			pbDelivery.LastAttemptedAt = delivery.LastAttemptedAt.Unix()
		}
		if delivery.NextRetryAt != nil {
			pbDelivery.NextRetryAt = delivery.NextRetryAt.Unix()
		}
	}
	return &pb.GetWebhookDeliveryStatusResponse{
		Delivery: pbDelivery,
		Success:  true,
		Message:  "Delivery status retrieved successfully",
	}, nil
}

// ResendWebhook resends a failed webhook delivery
func (s *WebhookServer) ResendWebhook(ctx context.Context, req *pb.ResendWebhookRequest) (*pb.ResendWebhookResponse, error) {
	newDeliveryID, err := s.service.ResendWebhook(ctx, req.DeliveryId, req.Namespace, req.ForceResend)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resend webhook: %v", err)
	}
	return &pb.ResendWebhookResponse{
		NewDeliveryId: newDeliveryID,
		Success:       true,
		Message:       "Webhook resent successfully",
	}, nil
}

// GetWebhookDeliveryHistory retrieves delivery history for a webhook
func (s *WebhookServer) GetWebhookDeliveryHistory(ctx context.Context, req *pb.GetWebhookDeliveryHistoryRequest) (*pb.GetWebhookDeliveryHistoryResponse, error) {
	deliveries, totalCount, err := s.service.GetWebhookDeliveryHistory(ctx, req.WebhookId, req.Namespace, req.Limit, req.Offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get delivery history: %v", err)
	}
	var pbDeliveries []*pb.WebhookDelivery
	for _, delivery := range deliveries {
		pbDelivery := &pb.WebhookDelivery{
			DeliveryId:   delivery.ID,
			WebhookId:    delivery.WebhookID,
			EventId:      delivery.EventID,
			Status:       convertDeliveryStatus(delivery.Status),
			AttemptCount: int32(delivery.AttemptCount),
			MaxAttempts:  int32(delivery.MaxAttempts),
			CreatedAt:    delivery.CreatedAt.Unix(),
			ExpiresAt:    delivery.ExpiresAt.Unix(),
			ResponseCode: int32(delivery.ResponseCode),
			ResponseBody: delivery.ResponseBody,
			ErrorMessage: delivery.ErrorMessage,
			RequestBody:  delivery.RequestBody,
		}
		if delivery.LastAttemptedAt != nil {
			pbDelivery.LastAttemptedAt = delivery.LastAttemptedAt.Unix()
		}
		if delivery.NextRetryAt != nil {
			pbDelivery.NextRetryAt = delivery.NextRetryAt.Unix()
		}
		pbDeliveries = append(pbDeliveries, pbDelivery)
	}
	return &pb.GetWebhookDeliveryHistoryResponse{
		Deliveries: pbDeliveries,
		TotalCount: totalCount,
		Success:    true,
		Message:    "Delivery history retrieved successfully",
	}, nil
}

// ResubmitWebhook manually retries failed or pending webhook deliveries
func (s *WebhookServer) ResubmitWebhook(ctx context.Context, req *pb.ResubmitWebhookRequest) (*pb.ResubmitWebhookResponse, error) {
	if req.GetDeliveryId() == "" {
		return nil, status.Error(codes.InvalidArgument, "identifier cannot be empty")
	}

	newDeliveryID, err := s.service.ResendWebhook(ctx, req.GetDeliveryId(), req.Namespace, req.Force)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resubmit delivery: %v", err)
	}
	return &pb.ResubmitWebhookResponse{
		Success:          true,
		Message:          "Delivery resubmitted successfully",
		ResubmittedCount: 1,
		DeliveryIds:      []string{newDeliveryID},
	}, nil
}
