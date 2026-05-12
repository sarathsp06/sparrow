package grpc

import (
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
		SignatureType: req.SignatureType,
	}
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
