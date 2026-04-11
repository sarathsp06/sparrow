package webhooks

import (
	"testing"
)

func TestValidateConfig_RateLimitRPS(t *testing.T) {
	validConfig := func() WebhookHTTPConfig {
		return DefaultWebhookHTTPConfig()
	}

	t.Run("nil rate limit is valid", func(t *testing.T) {
		config := validConfig()
		config.RateLimitRPS = nil
		if err := config.ValidateConfig(); err != nil {
			t.Errorf("ValidateConfig() with nil RateLimitRPS returned error: %v", err)
		}
	})

	t.Run("positive rate limit is valid", func(t *testing.T) {
		config := validConfig()
		rps := 10.0
		config.RateLimitRPS = &rps
		if err := config.ValidateConfig(); err != nil {
			t.Errorf("ValidateConfig() with RateLimitRPS=10 returned error: %v", err)
		}
	})

	t.Run("fractional rate limit is valid", func(t *testing.T) {
		config := validConfig()
		rps := 0.5 // 1 request per 2 seconds
		config.RateLimitRPS = &rps
		if err := config.ValidateConfig(); err != nil {
			t.Errorf("ValidateConfig() with RateLimitRPS=0.5 returned error: %v", err)
		}
	})

	t.Run("very small positive rate limit is valid", func(t *testing.T) {
		config := validConfig()
		rps := 0.001 // 1 request per 1000 seconds
		config.RateLimitRPS = &rps
		if err := config.ValidateConfig(); err != nil {
			t.Errorf("ValidateConfig() with RateLimitRPS=0.001 returned error: %v", err)
		}
	})

	t.Run("zero rate limit is invalid", func(t *testing.T) {
		config := validConfig()
		rps := 0.0
		config.RateLimitRPS = &rps
		if err := config.ValidateConfig(); err == nil {
			t.Error("ValidateConfig() with RateLimitRPS=0 should return error")
		}
	})

	t.Run("negative rate limit is invalid", func(t *testing.T) {
		config := validConfig()
		rps := -5.0
		config.RateLimitRPS = &rps
		if err := config.ValidateConfig(); err == nil {
			t.Error("ValidateConfig() with RateLimitRPS=-5 should return error")
		}
	})
}

func TestApplyConfig_RateLimitRPS(t *testing.T) {
	t.Run("nil does not override existing", func(t *testing.T) {
		rps := 10.0
		config := WebhookHTTPConfig{RateLimitRPS: &rps}
		other := WebhookHTTPConfig{RateLimitRPS: nil}
		config.ApplyConfig(&other)

		if config.RateLimitRPS == nil || *config.RateLimitRPS != 10.0 {
			t.Errorf("ApplyConfig with nil RateLimitRPS should preserve existing, got %v", config.RateLimitRPS)
		}
	})

	t.Run("non-nil overrides nil", func(t *testing.T) {
		config := WebhookHTTPConfig{RateLimitRPS: nil}
		rps := 5.0
		other := WebhookHTTPConfig{RateLimitRPS: &rps}
		config.ApplyConfig(&other)

		if config.RateLimitRPS == nil || *config.RateLimitRPS != 5.0 {
			t.Errorf("ApplyConfig should set RateLimitRPS to 5.0, got %v", config.RateLimitRPS)
		}
	})

	t.Run("non-nil overrides existing", func(t *testing.T) {
		old := 10.0
		config := WebhookHTTPConfig{RateLimitRPS: &old}
		newRPS := 20.0
		other := WebhookHTTPConfig{RateLimitRPS: &newRPS}
		config.ApplyConfig(&other)

		if config.RateLimitRPS == nil || *config.RateLimitRPS != 20.0 {
			t.Errorf("ApplyConfig should override to 20.0, got %v", config.RateLimitRPS)
		}
	})

	t.Run("nil other config is safe", func(t *testing.T) {
		rps := 10.0
		config := WebhookHTTPConfig{RateLimitRPS: &rps}
		config.ApplyConfig(nil)

		if config.RateLimitRPS == nil || *config.RateLimitRPS != 10.0 {
			t.Errorf("ApplyConfig(nil) should preserve existing, got %v", config.RateLimitRPS)
		}
	})
}

func TestDefaultWebhookHTTPConfig_RateLimitRPS(t *testing.T) {
	config := DefaultWebhookHTTPConfig()
	if config.RateLimitRPS != nil {
		t.Errorf("Default config should have nil RateLimitRPS, got %v", config.RateLimitRPS)
	}
}

func TestToWebhookRegistration_RateLimitRPS(t *testing.T) {
	t.Run("rate limit flows through HTTPConfig", func(t *testing.T) {
		rps := 5.0
		req := WebhookRegistrationRequest{
			Namespace: "test",
			Events:    []string{"event.test"},
			URL:       "https://example.com/webhook",
			HTTPConfig: &WebhookHTTPConfig{
				MaxRetries:            3,
				RetryBackoffSeconds:   60,
				RequestTimeoutSeconds: 30,
				ExpectedStatusCodes:   IntArray{200},
				ContentType:           "application/json",
				UserAgent:             "test",
				FollowRedirects:       true,
				VerifySSL:             true,
				RateLimitRPS:          &rps,
			},
		}

		webhook, err := req.ToWebhookRegistration()
		if err != nil {
			t.Fatalf("ToWebhookRegistration() returned error: %v", err)
		}

		if webhook.HTTPConfig.RateLimitRPS == nil {
			t.Fatal("Expected RateLimitRPS to be set on webhook")
		}
		if *webhook.HTTPConfig.RateLimitRPS != 5.0 {
			t.Errorf("Expected RateLimitRPS=5.0, got %f", *webhook.HTTPConfig.RateLimitRPS)
		}
	})

	t.Run("no rate limit by default", func(t *testing.T) {
		req := WebhookRegistrationRequest{
			Namespace: "test",
			Events:    []string{"event.test"},
			URL:       "https://example.com/webhook",
		}

		webhook, err := req.ToWebhookRegistration()
		if err != nil {
			t.Fatalf("ToWebhookRegistration() returned error: %v", err)
		}

		if webhook.HTTPConfig.RateLimitRPS != nil {
			t.Errorf("Expected nil RateLimitRPS by default, got %v", webhook.HTTPConfig.RateLimitRPS)
		}
	})

	t.Run("invalid rate limit rejected", func(t *testing.T) {
		rps := -1.0
		req := WebhookRegistrationRequest{
			Namespace: "test",
			Events:    []string{"event.test"},
			URL:       "https://example.com/webhook",
			HTTPConfig: &WebhookHTTPConfig{
				MaxRetries:            3,
				RetryBackoffSeconds:   60,
				RequestTimeoutSeconds: 30,
				ExpectedStatusCodes:   IntArray{200},
				ContentType:           "application/json",
				UserAgent:             "test",
				FollowRedirects:       true,
				VerifySSL:             true,
				RateLimitRPS:          &rps,
			},
		}

		_, err := req.ToWebhookRegistration()
		if err == nil {
			t.Error("Expected error for negative RateLimitRPS, got nil")
		}
	})
}
