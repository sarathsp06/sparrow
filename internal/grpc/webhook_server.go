package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
	"github.com/sarathsp06/httpqueue/internal/jobs"
	"github.com/sarathsp06/httpqueue/internal/logger"
	"github.com/sarathsp06/httpqueue/internal/queue"
	pb "github.com/sarathsp06/httpqueue/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WebhookServer implements the WebhookService gRPC interface
type WebhookServer struct {
	pb.UnimplementedWebhookServiceServer
	queueManager *queue.Manager
	logger       *slog.Logger
}

// NewWebhookServer creates a new WebhookServer instance
func NewWebhookServer(queueManager *queue.Manager) *WebhookServer {
	return &WebhookServer{
		queueManager: queueManager,
		logger:       logger.NewLogger("grpc-webhook-server"),
	}
}

// ScheduleWebhook schedules a single webhook to be sent
func (s *WebhookServer) ScheduleWebhook(ctx context.Context, req *pb.ScheduleWebhookRequest) (*pb.ScheduleWebhookResponse, error) {
	s.logger.Info("Received webhook schedule request",
		"url", req.Url,
		"method", req.Method,
		"queue", req.Queue,
	)

	// Validate required fields
	if req.Url == "" {
		return nil, status.Error(codes.InvalidArgument, "URL is required")
	}

	// Set default values
	method := req.Method
	if method == "" {
		method = "POST"
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30 // Default 30 seconds
	}

	queueName := req.Queue
	if queueName == "" {
		queueName = "webhooks"
	}

	// Parse payload JSON
	var payload map[string]interface{}
	if req.Payload != "" {
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid JSON payload: %v", err)
		}
	} else {
		payload = make(map[string]interface{})
	}

	// Create webhook job arguments
	webhookArgs := jobs.WebhookArgs{
		URL:     req.Url,
		Method:  method,
		Headers: req.Headers,
		Payload: payload,
		Timeout: int(timeout),
	}

	// Create insert options
	insertOpts := &river.InsertOpts{
		Queue:    queueName,
		Priority: int(req.Priority),
	}

	// Handle scheduled execution
	if req.ScheduledAt > 0 {
		insertOpts.ScheduledAt = time.Unix(req.ScheduledAt, 0)
	}

	// Insert the job
	result, err := s.queueManager.InsertWebhookJob(ctx, webhookArgs, insertOpts)
	if err != nil {
		s.logger.Error("Failed to schedule webhook job",
			"url", req.Url,
			"error", err,
		)
		return nil, status.Errorf(codes.Internal, "failed to schedule webhook: %v", err)
	}

	scheduledAt := time.Now().Unix()
	if req.ScheduledAt > 0 {
		scheduledAt = req.ScheduledAt
	}

	s.logger.Info("Webhook job scheduled successfully",
		"job_id", result.Job.ID,
		"url", req.Url,
		"method", method,
		"queue", queueName,
	)

	return &pb.ScheduleWebhookResponse{
		JobId:       result.Job.ID,
		Success:     true,
		Message:     "Webhook scheduled successfully",
		ScheduledAt: scheduledAt,
	}, nil
}

// ScheduleWebhookBatch schedules multiple webhooks to be sent
func (s *WebhookServer) ScheduleWebhookBatch(ctx context.Context, req *pb.ScheduleWebhookBatchRequest) (*pb.ScheduleWebhookBatchResponse, error) {
	s.logger.Info("Received batch webhook schedule request",
		"webhook_count", len(req.Webhooks),
	)

	if len(req.Webhooks) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one webhook is required")
	}

	results := make([]*pb.ScheduleWebhookResponse, 0, len(req.Webhooks))
	totalScheduled := int32(0)
	totalFailed := int32(0)

	// Process each webhook request
	for i, webhookReq := range req.Webhooks {
		response, err := s.ScheduleWebhook(ctx, webhookReq)
		if err != nil {
			// Create a failed response
			response = &pb.ScheduleWebhookResponse{
				JobId:   0,
				Success: false,
				Message: fmt.Sprintf("Failed to schedule webhook %d: %v", i+1, err),
			}
			totalFailed++
		} else {
			totalScheduled++
		}
		results = append(results, response)
	}

	s.logger.Info("Batch webhook scheduling completed",
		"total_webhooks", len(req.Webhooks),
		"scheduled", totalScheduled,
		"failed", totalFailed,
	)

	return &pb.ScheduleWebhookBatchResponse{
		Results:        results,
		TotalScheduled: totalScheduled,
		TotalFailed:    totalFailed,
	}, nil
}

// GetWebhookStatus gets the status of a webhook job
func (s *WebhookServer) GetWebhookStatus(ctx context.Context, req *pb.GetWebhookStatusRequest) (*pb.GetWebhookStatusResponse, error) {
	s.logger.Info("Received webhook status request",
		"job_id", req.JobId,
	)

	if req.JobId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "valid job ID is required")
	}

	// For now, return a placeholder response since River doesn't expose job status API directly
	// In a real implementation, you would query the River database for job status
	return &pb.GetWebhookStatusResponse{
		JobId:         req.JobId,
		Status:        pb.WebhookJobStatus_PENDING, // Default status
		Message:       "Job status querying not yet implemented",
		CreatedAt:     time.Now().Unix(),
		ScheduledAt:   time.Now().Unix(),
		AttemptedAt:   0,
		AttemptCount:  0,
		MaxAttempts:   5,
	}, nil
}