package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

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
			return nil, toGRPCError(ctx, err, "failed to register webhook")
		}
		return &pb.RegisterWebhookResponse{
			WebhookId: webhook.ID,
			CreatedAt: convertTimeToProto(webhook.CreatedAt),
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
		return nil, toGRPCError(ctx, err, "failed to register webhook")
	}
	return &pb.RegisterWebhookResponse{
		WebhookId: webhookID,
		CreatedAt: timestamppb.New(createdAt),
	}, nil
}

// UnregisterWebhook removes a webhook registration
func (s *WebhookServer) UnregisterWebhook(ctx context.Context, req *pb.UnregisterWebhookRequest) (*pb.UnregisterWebhookResponse, error) {
	err := s.service.UnregisterWebhook(ctx, req.WebhookId, req.Namespace)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to unregister webhook")
	}
	return &pb.UnregisterWebhookResponse{}, nil
}

// ListWebhooks lists all registered webhooks for a namespace
func (s *WebhookServer) ListWebhooks(ctx context.Context, req *pb.ListWebhooksRequest) (*pb.ListWebhooksResponse, error) {
	var limit, offset int32
	if req.Pagination != nil {
		limit = req.Pagination.Limit
		offset = req.Pagination.Offset
	}

	regs, totalCount, err := s.service.ListWebhooks(ctx, req.Namespace, req.WebhookId, req.Event, req.ActiveOnly, limit, offset)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to list webhooks")
	}
	pbWebhooks := make([]*pb.RegisteredWebhook, len(regs))
	for i, reg := range regs {
		pbWebhooks[i] = &pb.RegisteredWebhook{
			WebhookId:   reg.ID.String(),
			Namespace:   reg.Namespace,
			Events:      s.getWebhookEvents(ctx, reg.ID.String(), reg.Namespace),
			Url:         reg.URL,
			Headers:     reg.Headers,
			Timeout:     int32(reg.Timeout),
			Active:      reg.Active,
			Description: reg.Description,
			Health:      convertWebhookHealth(reg.Health),
			CreatedAt:   convertTimeToProto(reg.CreatedAt),
			UpdatedAt:   convertTimeToProto(reg.UpdatedAt),
			HttpConfig: &pb.WebhookHTTPConfig{
				MaxRetries:            int32(reg.MaxRetries),
				RetryBackoffSeconds:   int32(reg.RetryBackoffSeconds),
				CaptureResponseBody:   reg.CaptureResponseBody,
				FollowRedirects:       reg.FollowRedirects,
				VerifySsl:             reg.VerifySSL,
				RequestTimeoutSeconds: int32(reg.RequestTimeoutSeconds),
				ExpectedStatusCodes:   convertExpectedStatusCodes(reg.ExpectedStatusCodes),
				WebhookSecret:         maskSecret(reg.WebhookSecret),
				UserAgent:             reg.UserAgent,
				ContentType:           reg.ContentType,
			},
		}
	}
	return &pb.ListWebhooksResponse{
		Webhooks: pbWebhooks,
		Pagination: &pb.PaginationResponse{
			TotalCount: totalCount,
			Limit:      limit,
			Offset:     offset,
		},
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
	var httpConfig *webhooks.HTTPConfigUpdate
	if req.Updates != nil {
		events = req.Updates.Events
		url = req.Updates.Url
		headers = req.Updates.Headers
		timeout = int(req.Updates.Timeout)
		active = req.Updates.Active
		description = req.Updates.Description
		if req.Updates.HttpConfig != nil {
			httpConfig = &webhooks.HTTPConfigUpdate{
				MaxRetries:            int(req.Updates.HttpConfig.MaxRetries),
				RetryBackoffSeconds:   int(req.Updates.HttpConfig.RetryBackoffSeconds),
				CaptureResponseBody:   req.Updates.HttpConfig.CaptureResponseBody,
				FollowRedirects:       req.Updates.HttpConfig.FollowRedirects,
				VerifySSL:             req.Updates.HttpConfig.VerifySsl,
				RequestTimeoutSeconds: int(req.Updates.HttpConfig.RequestTimeoutSeconds),
				ExpectedStatusCodes:   convertStatusCodesToInt(req.Updates.HttpConfig.ExpectedStatusCodes),
				WebhookSecret:         req.Updates.HttpConfig.WebhookSecret,
				UserAgent:             req.Updates.HttpConfig.UserAgent,
				ContentType:           req.Updates.HttpConfig.ContentType,
			}
		}
	}
	err := s.service.UpdateWebhookConfig(ctx, req.WebhookId, req.Namespace, events, url, headers, timeout, active, description, httpConfig)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to update webhook config")
	}
	return &pb.UpdateWebhookConfigResponse{}, nil
}

// PauseWebhook temporarily disables a webhook
func (s *WebhookServer) PauseWebhook(ctx context.Context, req *pb.PauseWebhookRequest) (*pb.PauseWebhookResponse, error) {
	err := s.service.PauseWebhook(ctx, req.WebhookId, req.Namespace, req.Reason)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to pause webhook")
	}
	return &pb.PauseWebhookResponse{}, nil
}

// ResumeWebhook re-enables a paused webhook
func (s *WebhookServer) ResumeWebhook(ctx context.Context, req *pb.ResumeWebhookRequest) (*pb.ResumeWebhookResponse, error) {
	err := s.service.ResumeWebhook(ctx, req.WebhookId, req.Namespace)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to resume webhook")
	}
	return &pb.ResumeWebhookResponse{}, nil
}

// GetTemplateFunctions returns all available template functions with their descriptions
func (s *WebhookServer) GetTemplateFunctions(ctx context.Context, req *pb.GetTemplateFunctionsRequest) (*pb.GetTemplateFunctionsResponse, error) {
	templateFunctions := s.service.GetTemplateFunctions()

	var pbFunctions []*pb.TemplateFunction
	for _, tf := range templateFunctions {
		pbFunctions = append(pbFunctions, &pb.TemplateFunction{
			Name:        tf.Name,
			Description: tf.Description,
		})
	}

	return &pb.GetTemplateFunctionsResponse{
		Functions:  pbFunctions,
		TotalCount: int32(len(pbFunctions)),
	}, nil
}
