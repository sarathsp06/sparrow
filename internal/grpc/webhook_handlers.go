package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sarathsp06/sparrow/internal/webhooks"
	pb "github.com/sarathsp06/sparrow/proto"
)

// RegisterWebhook registers a URL for specific events in a namespace
func (s *WebhookServer) RegisterWebhook(ctx context.Context, req *pb.RegisterWebhookRequest) (*pb.RegisterWebhookResponse, error) {
	// Use new service interface if it has CreateWebhook method (enhanced), fallback to legacy method
	if enhancedService, ok := s.service.(interface {
		CreateWebhook(ctx context.Context, req webhooks.WebhookRegistrationRequest) (*webhooks.WebhookRegistration, error)
	}); ok {
		// Use enhanced service with HTTP config support
		webhookReq := CreateWebhookRegistrationRequest(req)
		webhook, err := enhancedService.CreateWebhook(ctx, webhookReq)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to register webhook: %v", err)
		}
		return &pb.RegisterWebhookResponse{
			WebhookId: webhook.ID,
			Success:   true,
			Message:   "Webhook registered successfully",
			CreatedAt: webhook.CreatedAt.Unix(),
		}, nil
	}

	// Fallback to legacy method for backward compatibility
	timeout := int(req.Timeout)
	if timeout == 0 && req.HttpConfig != nil && req.HttpConfig.RequestTimeoutSeconds > 0 {
		timeout = int(req.HttpConfig.RequestTimeoutSeconds)
	}
	if timeout == 0 {
		timeout = 30 // Default timeout
	}

	webhookID, createdAt, err := s.service.RegisterWebhook(ctx, req.Namespace, req.Events, req.Url, req.Headers, timeout, req.Active, req.Description)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register webhook: %v", err)
	}
	return &pb.RegisterWebhookResponse{
		WebhookId: webhookID,
		Success:   true,
		Message:   "Webhook registered successfully",
		CreatedAt: createdAt,
	}, nil
}

// UnregisterWebhook removes a webhook registration
func (s *WebhookServer) UnregisterWebhook(ctx context.Context, req *pb.UnregisterWebhookRequest) (*pb.UnregisterWebhookResponse, error) {
	err := s.service.UnregisterWebhook(ctx, req.WebhookId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unregister webhook: %v", err)
	}
	return &pb.UnregisterWebhookResponse{
		Success: true,
		Message: "Webhook unregistered successfully",
	}, nil
}

// ListWebhooks lists all registered webhooks for a namespace
func (s *WebhookServer) ListWebhooks(ctx context.Context, req *pb.ListWebhooksRequest) (*pb.ListWebhooksResponse, error) {
	regs, err := s.service.ListWebhooks(ctx, req.Namespace, req.Event, req.ActiveOnly)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list webhooks: %v", err)
	}
	pbWebhooks := make([]*pb.RegisteredWebhook, len(regs))
	for i, reg := range regs {
		pbWebhooks[i] = &pb.RegisteredWebhook{
			WebhookId:   reg.ID,
			Namespace:   reg.Namespace,
			Events:      s.getWebhookEvents(ctx, reg.ID),
			Url:         reg.URL,
			Headers:     reg.Headers,
			Timeout:     int32(reg.Timeout),
			Active:      reg.Active,
			Description: reg.Description,
			Health:      convertWebhookHealth(reg.Health),
			CreatedAt:   reg.CreatedAt.Unix(),
			UpdatedAt:   reg.UpdatedAt.Unix(),
			HttpConfig: &pb.WebhookHTTPConfig{
				MaxRetries:            int32(reg.MaxRetries),
				RetryBackoffSeconds:   int32(reg.RetryBackoffSeconds),
				CaptureResponseBody:   reg.CaptureResponseBody,
				FollowRedirects:       reg.FollowRedirects,
				VerifySsl:             reg.VerifySSL,
				RequestTimeoutSeconds: int32(reg.RequestTimeoutSeconds),
				ExpectedStatusCodes:   convertExpectedStatusCodes(reg.ExpectedStatusCodes),
				WebhookSecret:         reg.WebhookSecret,
				UserAgent:             reg.UserAgent,
				ContentType:           reg.ContentType,
			},
		}
	}
	return &pb.ListWebhooksResponse{
		Webhooks:   pbWebhooks,
		TotalCount: int32(len(pbWebhooks)),
		Success:    true,
		Message:    "Webhooks listed successfully",
	}, nil
}

// GetRegisteredWebhooks retrieves registered webhooks by ID or namespace
func (s *WebhookServer) GetRegisteredWebhooks(ctx context.Context, req *pb.GetRegisteredWebhooksRequest) (*pb.GetRegisteredWebhooksResponse, error) {
	regs, err := s.service.GetRegisteredWebhooks(ctx, req.Namespace, req.WebhookId, req.ActiveOnly)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get registered webhooks: %v", err)
	}
	var webhooks []*pb.RegisteredWebhook
	for _, webhook := range regs {
		webhooks = append(webhooks, &pb.RegisteredWebhook{
			WebhookId:   webhook.ID,
			Namespace:   webhook.Namespace,
			Events:      s.getWebhookEvents(ctx, webhook.ID),
			Url:         webhook.URL,
			Headers:     webhook.Headers,
			Timeout:     int32(webhook.Timeout),
			Active:      webhook.Active,
			Description: webhook.Description,
			Health:      convertWebhookHealth(webhook.Health),
			CreatedAt:   webhook.CreatedAt.Unix(),
			UpdatedAt:   webhook.UpdatedAt.Unix(),
		})
	}
	return &pb.GetRegisteredWebhooksResponse{
		Webhooks:   webhooks,
		TotalCount: int32(len(webhooks)),
		Success:    true,
		Message:    "Webhooks retrieved successfully",
	}, nil
}

// ListRegisteredWebhooksByEvent retrieves webhooks registered for specific events
func (s *WebhookServer) ListRegisteredWebhooksByEvent(ctx context.Context, req *pb.ListRegisteredWebhooksByEventRequest) (*pb.ListRegisteredWebhooksByEventResponse, error) {
	regs, err := s.service.ListRegisteredWebhooksByEvent(ctx, req.Namespace, req.Event, req.ActiveOnly)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list webhooks by event: %v", err)
	}
	var webhooks []*pb.RegisteredWebhook
	for _, webhook := range regs {
		webhooks = append(webhooks, &pb.RegisteredWebhook{
			WebhookId:   webhook.ID,
			Namespace:   webhook.Namespace,
			Events:      s.getWebhookEvents(ctx, webhook.ID),
			Url:         webhook.URL,
			Headers:     webhook.Headers,
			Timeout:     int32(webhook.Timeout),
			Active:      webhook.Active,
			Description: webhook.Description,
			Health:      convertWebhookHealth(webhook.Health),
			CreatedAt:   webhook.CreatedAt.Unix(),
			UpdatedAt:   webhook.UpdatedAt.Unix(),
		})
	}
	return &pb.ListRegisteredWebhooksByEventResponse{
		Webhooks:   webhooks,
		Event:      req.Event,
		Namespace:  req.Namespace,
		TotalCount: int32(len(webhooks)),
		Success:    true,
		Message:    "Webhooks by event retrieved successfully",
	}, nil
}

// UpdateWebhookConfig updates webhook configuration
func (s *WebhookServer) UpdateWebhookConfig(ctx context.Context, req *pb.UpdateWebhookConfigRequest) (*pb.UpdateWebhookConfigResponse, error) {
	var events []string
	var url string
	var headers map[string]string
	var timeout int
	var active bool
	var description string
	if req.Updates != nil {
		events = req.Updates.Events
		url = req.Updates.Url
		headers = req.Updates.Headers
		timeout = int(req.Updates.Timeout)
		active = req.Updates.Active
		description = req.Updates.Description
	}
	err := s.service.UpdateWebhookConfig(ctx, req.WebhookId, req.Namespace, events, url, headers, timeout, active, description)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update webhook config: %v", err)
	}
	return &pb.UpdateWebhookConfigResponse{
		Success: true,
		Message: "Webhook config updated successfully",
	}, nil
}

// PauseWebhook temporarily disables a webhook
func (s *WebhookServer) PauseWebhook(ctx context.Context, req *pb.PauseWebhookRequest) (*pb.PauseWebhookResponse, error) {
	err := s.service.PauseWebhook(ctx, req.WebhookId, req.Namespace, req.Reason)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to pause webhook: %v", err)
	}
	return &pb.PauseWebhookResponse{
		Success: true,
		Message: "Webhook paused successfully",
	}, nil
}

// ResumeWebhook re-enables a paused webhook
func (s *WebhookServer) ResumeWebhook(ctx context.Context, req *pb.ResumeWebhookRequest) (*pb.ResumeWebhookResponse, error) {
	err := s.service.ResumeWebhook(ctx, req.WebhookId, req.Namespace)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resume webhook: %v", err)
	}
	return &pb.ResumeWebhookResponse{
		Success: true,
		Message: "Webhook resumed successfully",
	}, nil
}
