package grpc

import (
	"context"

	"github.com/sarathsp06/sparrow/internal/webhooks"
	pb "github.com/sarathsp06/sparrow/proto"
)

// RegisterWebhook registers a URL for specific events in a namespace
func (s *WebhookServer) RegisterWebhook(ctx context.Context, req *pb.RegisterWebhookRequest) (*pb.RegisterWebhookResponse, error) {
	webhookReq := CreateWebhookRegistrationRequest(req)
	webhook, err := s.service.CreateWebhook(ctx, webhookReq)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to register webhook")
	}
	return &pb.RegisterWebhookResponse{
		WebhookId:        webhook.ID,
		CreatedAt:        convertTimeToProto(webhook.CreatedAt),
		SigningPublicKey: deriveEd25519PublicKeyHex(webhook.Ed25519EncryptedPrivateKey, s.service),
		SignatureType:    webhook.SignatureType,
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
	limit, offset := extractPagination(req.Pagination)

	regs, totalCount, err := s.service.ListWebhooks(ctx, req.Namespace, req.WebhookId, req.Event, req.ActiveOnly, limit, offset)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to list webhooks")
	}

	// Batch-fetch events for all webhooks (single query instead of N+1)
	eventsMap := s.getWebhookEventsMap(ctx, regs)

	pbWebhooks := make([]*pb.RegisteredWebhook, len(regs))
	for i, reg := range regs {
		pbWebhooks[i] = convertWebhookRegToProto(reg, eventsMap[reg.ID.String()], s.service)
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
	var secretHeaders map[string]string
	var timeout int
	var active bool
	var description string
	var signatureType string
	var httpConfig *webhooks.HTTPConfigUpdate
	if req.Updates != nil {
		events = req.Updates.Events
		url = req.Updates.Url
		headers = req.Updates.Headers
		secretHeaders = req.Updates.SecretHeaders
		active = req.Updates.Active
		description = req.Updates.Description
		signatureType = req.Updates.SignatureType
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
				RateLimitRPS:          float32PtrToFloat64Ptr(req.Updates.HttpConfig.RateLimitRps),
			}
		}
	}
	// Extract field mask paths
	var updateMask []string
	if req.UpdateMask != nil {
		updateMask = req.UpdateMask.GetPaths()
	}
	err := s.service.UpdateWebhookConfig(ctx, req.WebhookId, req.Namespace, events, url, headers, timeout, active, description, httpConfig, secretHeaders, signatureType, updateMask)
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
