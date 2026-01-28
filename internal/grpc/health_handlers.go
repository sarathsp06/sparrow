package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	pb "github.com/sarathsp06/sparrow/proto"
)

// GetWebhookHealth gets health metrics for a webhook
func (s *WebhookServer) GetWebhookHealth(ctx context.Context, req *pb.GetWebhookHealthRequest) (*pb.GetWebhookHealthResponse, error) {
	healthData, err := s.service.GetWebhookHealth(ctx, req.WebhookId, req.Namespace)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get webhook health: %v", err)
	}
	var pbMetrics *pb.WebhookHealthMetrics
	if healthData != nil {
		pbMetrics = &pb.WebhookHealthMetrics{
			WebhookId:            healthData.WebhookID,
			TotalDeliveries:      int32(healthData.TotalDeliveries),
			SuccessfulDeliveries: int32(healthData.SuccessfulDeliveries),
			FailedDeliveries:     int32(healthData.FailedDeliveries),
			ConsecutiveFailures:  int32(healthData.ConsecutiveFailures),
			SuccessRate:          healthData.SuccessRate,
			AvgResponseTime:      int32(healthData.AvgResponseTime),
			CreatedAt:            convertTimeToProto(healthData.CreatedAt),
			UpdatedAt:            convertTimeToProto(healthData.UpdatedAt),
			LastSuccessAt:        convertPtrTimeToProto(healthData.LastSuccessAt),
			LastFailureAt:        convertPtrTimeToProto(healthData.LastFailureAt),
		}
	}
	return &pb.GetWebhookHealthResponse{
		Success:   true,
		Message:   "Webhook health retrieved successfully",
		WebhookId: req.WebhookId,
		Health:    convertWebhookHealth(healthData.Health),
		Metrics:   pbMetrics,
	}, nil
}

// GetHealthSummary gets a summary of webhook health
func (s *WebhookServer) GetHealthSummary(ctx context.Context, req *pb.GetHealthSummaryRequest) (*pb.GetHealthSummaryResponse, error) {
	summaryData, err := s.service.GetHealthSummary(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get health summary: %v", err)
	}
	var pbSummary *pb.HealthSummary
	if summaryData != nil {
		pbSummary = &pb.HealthSummary{
			HealthyCount:   int32(summaryData.HealthyCount),
			DegradedCount:  int32(summaryData.DegradedCount),
			UnhealthyCount: int32(summaryData.UnhealthyCount),
			UnknownCount:   int32(summaryData.UnknownCount),
			TotalCount:     int32(summaryData.TotalCount),
		}
	}
	return &pb.GetHealthSummaryResponse{
		Success: true,
		Message: "Health summary retrieved successfully",
		Summary: pbSummary,
	}, nil
}

// ListWebhooksByHealth lists webhooks filtered by health status
func (s *WebhookServer) ListWebhooksByHealth(ctx context.Context, req *pb.ListWebhooksByHealthRequest) (*pb.ListWebhooksByHealthResponse, error) {
	// Convert protobuf health enum to store health enum
	var storeHealth store.WebhookHealth
	switch req.Health {
	case pb.WebhookHealth_HEALTH_HEALTHY:
		storeHealth = store.HealthHealthy
	case pb.WebhookHealth_HEALTH_DEGRADED:
		storeHealth = store.HealthDegraded
	case pb.WebhookHealth_HEALTH_UNHEALTHY:
		storeHealth = store.HealthUnhealthy
	case pb.WebhookHealth_HEALTH_UNSPECIFIED:
		storeHealth = store.HealthUnknown
	default:
		storeHealth = store.HealthUnknown
	}

	webhooks, err := s.service.ListWebhooksByHealth(ctx, storeHealth)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list webhooks by health: %v", err)
	}

	pbWebhooks := make([]*pb.RegisteredWebhook, len(webhooks))
	for i, webhook := range webhooks {
		pbWebhooks[i] = &pb.RegisteredWebhook{
			WebhookId:   webhook.ID.String(),
			Namespace:   webhook.Namespace,
			Events:      s.getWebhookEvents(ctx, webhook.ID.String()),
			Url:         webhook.URL,
			Headers:     webhook.Headers,
			Timeout:     int32(webhook.Timeout),
			Active:      webhook.Active,
			Description: webhook.Description,
			Health:      convertWebhookHealth(webhook.Health),
			CreatedAt:   convertTimeToProto(webhook.CreatedAt),
			UpdatedAt:   convertTimeToProto(webhook.UpdatedAt),
		}
	}

	return &pb.ListWebhooksByHealthResponse{
		Webhooks:   pbWebhooks,
		TotalCount: int32(len(pbWebhooks)),
		Success:    true,
	}, nil
}

// GetNamespaceStats retrieves statistics for a namespace
func (s *WebhookServer) GetNamespaceStats(ctx context.Context, req *pb.GetNamespaceStatsRequest) (*pb.GetNamespaceStatsResponse, error) {
	stats, err := s.service.GetNamespaceStats(ctx, req.Namespace)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get namespace stats: %v", err)
	}
	var pbStats *pb.NamespaceStats
	if stats != nil {
		pbStats = &pb.NamespaceStats{
			TotalWebhooks:        int32(stats.TotalWebhooks),
			ActiveWebhooks:       int32(stats.ActiveWebhooks),
			TotalDeliveries:      int32(stats.TotalDeliveries),
			SuccessfulDeliveries: int32(stats.SuccessfulDeliveries),
			FailedDeliveries:     int32(stats.FailedDeliveries),
			PendingDeliveries:    int32(stats.PendingDeliveries),
			SuccessRate:          stats.SuccessRate,
		}
	}
	return &pb.GetNamespaceStatsResponse{
		Namespace: req.Namespace,
		Stats:     pbStats,
		Success:   true,
		Message:   "Namespace stats retrieved successfully",
	}, nil
}
