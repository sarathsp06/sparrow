package grpc

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/sarathsp06/sparrow/proto"
)

// GetDeliveryStatus retrieves delivery status for specific delivery
func (s *WebhookServer) GetDeliveryStatus(ctx context.Context, req *pb.GetDeliveryStatusRequest) (*pb.GetDeliveryStatusResponse, error) {
	delivery, err := s.service.GetDeliveryStatus(ctx, req.DeliveryId, req.Namespace)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to get delivery status")
	}
	var pbDelivery *pb.WebhookDelivery
	if delivery != nil {
		pbDelivery = &pb.WebhookDelivery{
			DeliveryId:      delivery.ID.String(),
			WebhookId:       delivery.WebhookID.String(),
			EventId:         delivery.EventID.String(),
			Status:          convertDeliveryStatus(delivery.Status),
			AttemptCount:    int32(delivery.AttemptCount),
			MaxAttempts:     int32(delivery.MaxAttempts),
			CreatedAt:       convertTimeToProto(delivery.CreatedAt),
			LastAttemptedAt: convertPtrTimeToProto(delivery.LastAttemptedAt),
			NextRetryAt:     convertPtrTimeToProto(delivery.NextRetryAt),
			ExpiresAt:       convertTimeToProto(delivery.ExpiresAt),
			ResponseCode:    int32(delivery.ResponseCode),
			ResponseBody:    delivery.ResponseBody,
			ErrorMessage:    delivery.ErrorMessage,
			RequestBody:     delivery.RequestBody,
			ErrorCategory:   delivery.ErrorCategory,
		}
	}
	return &pb.GetDeliveryStatusResponse{
		Delivery: pbDelivery,
	}, nil
}

// ListDeliveries retrieves delivery history with filters
func (s *WebhookServer) ListDeliveries(ctx context.Context, req *pb.ListDeliveriesRequest) (*pb.ListDeliveriesResponse, error) {
	var limit, offset int32
	if req.Pagination != nil {
		limit = req.Pagination.Limit
		offset = req.Pagination.Offset
	}

	deliveries, totalCount, err := s.service.ListDeliveries(ctx, req.Namespace, req.WebhookId, req.EventId, limit, offset)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to list deliveries")
	}
	var pbDeliveries []*pb.WebhookDelivery
	for _, delivery := range deliveries {
		pbDelivery := &pb.WebhookDelivery{
			DeliveryId:      delivery.ID.String(),
			WebhookId:       delivery.WebhookID.String(),
			EventId:         delivery.EventID.String(),
			Status:          convertDeliveryStatus(delivery.Status),
			AttemptCount:    int32(delivery.AttemptCount),
			MaxAttempts:     int32(delivery.MaxAttempts),
			CreatedAt:       convertTimeToProto(delivery.CreatedAt),
			LastAttemptedAt: convertPtrTimeToProto(delivery.LastAttemptedAt),
			NextRetryAt:     convertPtrTimeToProto(delivery.NextRetryAt),
			ExpiresAt:       convertTimeToProto(delivery.ExpiresAt),
			ResponseCode:    int32(delivery.ResponseCode),
			ResponseBody:    delivery.ResponseBody,
			ErrorMessage:    delivery.ErrorMessage,
			RequestBody:     delivery.RequestBody,
			ErrorCategory:   delivery.ErrorCategory,
		}
		pbDeliveries = append(pbDeliveries, pbDelivery)
	}
	return &pb.ListDeliveriesResponse{
		Deliveries: pbDeliveries,
		Pagination: &pb.PaginationResponse{
			TotalCount: totalCount,
			Limit:      limit,
			Offset:     offset,
		},
	}, nil
}

// RetryDelivery manually retries failed or pending webhook deliveries
func (s *WebhookServer) RetryDelivery(ctx context.Context, req *pb.RetryDeliveryRequest) (*pb.RetryDeliveryResponse, error) {
	resubmittedIDs, resubmittedCount, err := s.service.RetryDelivery(ctx, req.Namespace, req.DeliveryId, req.WebhookId, req.Force)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to retry delivery")
	}
	_ = strings.Join(resubmittedIDs, ",") // suppress unused import warning if needed
	return &pb.RetryDeliveryResponse{
		RetriedCount: resubmittedCount,
		DeliveryIds:  resubmittedIDs,
	}, nil
}

// GetDeliveryAttempts retrieves individual attempt history for a delivery
func (s *WebhookServer) GetDeliveryAttempts(ctx context.Context, req *pb.GetDeliveryAttemptsRequest) (*pb.GetDeliveryAttemptsResponse, error) {
	if req.DeliveryId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "delivery_id is required")
	}

	attempts, err := s.service.GetDeliveryAttempts(ctx, req.DeliveryId)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to get delivery attempts")
	}

	var pbAttempts []*pb.DeliveryAttempt
	for _, attempt := range attempts {
		pbAttempt := &pb.DeliveryAttempt{
			AttemptId:     attempt.ID.String(),
			DeliveryId:    attempt.DeliveryID.String(),
			WebhookId:     attempt.WebhookID.String(),
			Success:       attempt.Success,
			ResponseTime:  int32(attempt.ResponseTime),
			ResponseCode:  int32(attempt.ResponseCode),
			ErrorMessage:  attempt.ErrorMessage,
			ErrorCategory: attempt.ErrorCategory,
			Timestamp:     timestamppb.New(attempt.Timestamp),
		}
		pbAttempts = append(pbAttempts, pbAttempt)
	}

	return &pb.GetDeliveryAttemptsResponse{
		Attempts: pbAttempts,
	}, nil
}
