package grpc

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
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
	// Build filter from request
	filter := store.DeliveryFilter{
		Namespace: req.Namespace,
	}

	// Pagination
	if req.Pagination != nil {
		filter.Limit = int(req.Pagination.Limit)
		filter.Offset = int(req.Pagination.Offset)
	}

	// Webhook ID filter
	if req.WebhookId != "" {
		id, err := uuid.Parse(req.WebhookId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid webhook_id: %v", err)
		}
		filter.WebhookID = &id
	}

	// Event ID filter
	if req.EventId != "" {
		id, err := uuid.Parse(req.EventId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid event_id: %v", err)
		}
		filter.EventID = &id
	}

	// Status filter
	if req.Status != nil {
		filter.Status = req.Status
	}

	// Error category filter
	if req.ErrorCategory != nil {
		filter.ErrorCategory = req.ErrorCategory
	}

	// Subscription ID filter
	if req.SubscriptionId != nil {
		id, err := uuid.Parse(*req.SubscriptionId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid subscription_id: %v", err)
		}
		filter.SubscriptionID = &id
	}

	// Time range filters
	if req.CreatedAfter != nil {
		t := req.CreatedAfter.AsTime()
		filter.CreatedAfter = &t
	}
	if req.CreatedBefore != nil {
		t := req.CreatedBefore.AsTime()
		filter.CreatedBefore = &t
	}

	// Batch snapshot opt-in
	filter.PrepareRetry = req.PrepareRetry

	deliveries, totalCount, retryID, err := s.service.ListDeliveries(ctx, filter)
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
			Limit:      int32(filter.Limit),
			Offset:     int32(filter.Offset),
		},
		RetryId: retryID,
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

// RetryDeliveries starts a batch retry of deliveries previously snapshotted via ListDeliveries.
func (s *WebhookServer) RetryDeliveries(ctx context.Context, req *pb.RetryDeliveriesRequest) (*pb.RetryDeliveriesResponse, error) {
	if req.RetryId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "retry_id is required")
	}

	if err := s.service.RetryDeliveries(ctx, req.RetryId); err != nil {
		return nil, toGRPCError(ctx, err, "failed to start batch retry")
	}

	batch, err := s.service.GetRetryStatus(ctx, req.RetryId)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to get retry status")
	}

	return &pb.RetryDeliveriesResponse{
		RetryId: req.RetryId,
		Total:   int32(batch.Total),
		Status:  string(batch.Status),
	}, nil
}

// GetRetryStatus returns the current progress of a batch delivery retry.
func (s *WebhookServer) GetRetryStatus(ctx context.Context, req *pb.GetRetryStatusRequest) (*pb.GetRetryStatusResponse, error) {
	if req.RetryId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "retry_id is required")
	}

	batch, err := s.service.GetRetryStatus(ctx, req.RetryId)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to get retry status")
	}

	return &pb.GetRetryStatusResponse{
		Batch: batchJobToProto(batch),
	}, nil
}

// CancelRetry aborts a pending or in-progress batch delivery retry.
func (s *WebhookServer) CancelRetry(ctx context.Context, req *pb.CancelRetryRequest) (*pb.CancelRetryResponse, error) {
	if req.RetryId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "retry_id is required")
	}

	if err := s.service.CancelRetry(ctx, req.RetryId); err != nil {
		return nil, toGRPCError(ctx, err, "failed to cancel retry")
	}

	return &pb.CancelRetryResponse{
		Status: string(store.BatchStatusCancelled),
	}, nil
}
