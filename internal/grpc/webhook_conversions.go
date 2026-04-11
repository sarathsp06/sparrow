package grpc

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sarathsp06/sparrow/internal/webhooks"
	pb "github.com/sarathsp06/sparrow/proto"
)

// ConvertProtoHTTPConfig converts protobuf WebhookHTTPConfig to internal WebhookHTTPConfig
func ConvertProtoHTTPConfig(protoConfig *pb.WebhookHTTPConfig) *webhooks.WebhookHTTPConfig {
	if protoConfig == nil {
		// Return default configuration if not provided
		defaultConfig := webhooks.DefaultWebhookHTTPConfig()
		return &defaultConfig
	}

	config := &webhooks.WebhookHTTPConfig{
		MaxRetries:            int(protoConfig.MaxRetries),
		RetryBackoffSeconds:   int(protoConfig.RetryBackoffSeconds),
		CaptureResponseBody:   protoConfig.CaptureResponseBody,
		FollowRedirects:       protoConfig.FollowRedirects,
		VerifySSL:             protoConfig.VerifySsl,
		RequestTimeoutSeconds: int(protoConfig.RequestTimeoutSeconds),
		WebhookSecret:         protoConfig.WebhookSecret,
		UserAgent:             protoConfig.UserAgent,
		ContentType:           protoConfig.ContentType,
		RateLimitRPS:          float32PtrToFloat64Ptr(protoConfig.RateLimitRps),
	}

	// Convert expected status codes
	if len(protoConfig.ExpectedStatusCodes) > 0 {
		expectedCodes := make([]int, len(protoConfig.ExpectedStatusCodes))
		for i, code := range protoConfig.ExpectedStatusCodes {
			expectedCodes[i] = int(code)
		}
		config.ExpectedStatusCodes = webhooks.IntArray(expectedCodes)
	}

	// Apply defaults for zero values
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryBackoffSeconds == 0 {
		config.RetryBackoffSeconds = 60
	}
	if config.RequestTimeoutSeconds == 0 {
		config.RequestTimeoutSeconds = 30
	}
	if len(config.ExpectedStatusCodes) == 0 {
		config.ExpectedStatusCodes = webhooks.IntArray{200, 201, 202, 204}
	}
	if config.UserAgent == "" {
		config.UserAgent = "Sparrow-Webhook/1.0"
	}
	if config.ContentType == "" {
		config.ContentType = "application/json"
	}

	return config
}

// ConvertInternalHTTPConfig converts internal WebhookHTTPConfig to protobuf WebhookHTTPConfig
func ConvertInternalHTTPConfig(config *webhooks.WebhookHTTPConfig) *pb.WebhookHTTPConfig {
	if config == nil {
		return nil
	}

	protoConfig := &pb.WebhookHTTPConfig{
		MaxRetries:            int32(config.MaxRetries),
		RetryBackoffSeconds:   int32(config.RetryBackoffSeconds),
		CaptureResponseBody:   config.CaptureResponseBody,
		FollowRedirects:       config.FollowRedirects,
		VerifySsl:             config.VerifySSL,
		RequestTimeoutSeconds: int32(config.RequestTimeoutSeconds),
		WebhookSecret:         config.WebhookSecret,
		UserAgent:             config.UserAgent,
		ContentType:           config.ContentType,
		RateLimitRps:          float64PtrToFloat32Ptr(config.RateLimitRPS),
	}

	// Convert expected status codes
	if len(config.ExpectedStatusCodes) > 0 {
		expectedCodes := make([]int32, len(config.ExpectedStatusCodes))
		for i, code := range config.ExpectedStatusCodes {
			expectedCodes[i] = int32(code)
		}
		protoConfig.ExpectedStatusCodes = expectedCodes
	}

	return protoConfig
}

// CreateWebhookRegistrationRequest creates a WebhookRegistrationRequest from protobuf
func CreateWebhookRegistrationRequest(req *pb.RegisterWebhookRequest) webhooks.WebhookRegistrationRequest {
	// Convert headers map[string]string to map[string]any
	headers := make(map[string]any)
	for k, v := range req.Headers {
		headers[k] = v
	}

	// Convert active bool to *bool
	var active *bool
	if req.Active {
		active = &req.Active
	}

	return webhooks.WebhookRegistrationRequest{
		Namespace:     req.Namespace,
		Events:        req.Events,
		URL:           req.Url,
		Headers:       headers,
		Active:        active,
		Description:   req.Description,
		HTTPConfig:    ConvertProtoHTTPConfig(req.HttpConfig),
		SecretHeaders: req.SecretHeaders,
	}
}

// ConvertWebhookRegistrationToProto converts internal WebhookRegistration to protobuf RegisteredWebhook
func ConvertWebhookRegistrationToProto(webhook *webhooks.WebhookRegistration) *pb.RegisteredWebhook {
	if webhook == nil {
		return nil
	}

	// Convert headers map[string]any to map[string]string
	headers := make(map[string]string)
	for k, v := range webhook.Headers {
		if str, ok := v.(string); ok {
			headers[k] = str
		}
	}

	protoWebhook := &pb.RegisteredWebhook{
		WebhookId:   webhook.ID,
		Namespace:   webhook.Namespace,
		Events:      []string(webhook.Events),
		Url:         webhook.URL,
		Headers:     headers,
		Timeout:     int32(webhook.Timeout), // Legacy field
		Active:      webhook.Active,
		Description: webhook.Description,
		CreatedAt:   timestamppb.New(webhook.CreatedAt),
		UpdatedAt:   timestamppb.New(webhook.UpdatedAt),
		HttpConfig:  ConvertInternalHTTPConfig(&webhook.HTTPConfig),
	}

	// Convert health status
	switch webhook.Health {
	case "healthy":
		protoWebhook.Health = pb.WebhookHealth_HEALTH_HEALTHY
	case "degraded":
		protoWebhook.Health = pb.WebhookHealth_HEALTH_DEGRADED
	case "unhealthy":
		protoWebhook.Health = pb.WebhookHealth_HEALTH_UNHEALTHY
	default:
		protoWebhook.Health = pb.WebhookHealth_HEALTH_UNSPECIFIED
	}

	return protoWebhook
}

// float32PtrToFloat64Ptr converts *float32 (proto) to *float64 (internal model).
func float32PtrToFloat64Ptr(f *float32) *float64 {
	if f == nil {
		return nil
	}
	v := float64(*f)
	return &v
}

// float64PtrToFloat32Ptr converts *float64 (internal model) to *float32 (proto).
func float64PtrToFloat32Ptr(f *float64) *float32 {
	if f == nil {
		return nil
	}
	v := float32(*f)
	return &v
}
