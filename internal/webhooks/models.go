package webhooks

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"

	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

// WebhookRegistration represents a webhook registration with HTTP configuration
type WebhookRegistration struct {
	ID          string      `db:"id" json:"id"`
	Namespace   string      `db:"namespace" json:"namespace"`
	Events      StringArray `db:"events" json:"events"`
	URL         string      `db:"url" json:"url"`
	Headers     JSONBMap    `db:"headers" json:"headers"`
	Active      bool        `db:"active" json:"active"`
	Description string      `db:"description" json:"description"`
	Health      string      `db:"health" json:"health"`

	// HTTP Configuration
	HTTPConfig WebhookHTTPConfig `json:"http_config"`

	// Ed25519EncryptedPrivateKey holds the envelope-encrypted Ed25519 private key.
	// Only populated on creation; used to derive the public key for API responses.
	Ed25519EncryptedPrivateKey []byte `json:"-"`

	// SignatureType controls which signing scheme is used: "hmac" (default) or "ed25519".
	SignatureType string `json:"signature_type"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// WebhookHTTPConfig contains HTTP-specific configuration for webhook delivery
type WebhookHTTPConfig struct {
	MaxRetries          int `db:"max_retries" json:"max_retries"`
	RetryBackoffSeconds int `db:"retry_backoff_seconds" json:"retry_backoff_seconds"`
	// CaptureResponseBody controls the stored response body size limit per delivery attempt.
	// false (default): stores up to 1 KB. true: stores up to 1 MB.
	// The response body is always read regardless of this setting (required for HTTP connection reuse).
	CaptureResponseBody   bool     `db:"capture_response_body" json:"capture_response_body"`
	FollowRedirects       bool     `db:"follow_redirects" json:"follow_redirects"`
	VerifySSL             bool     `db:"verify_ssl" json:"verify_ssl"`
	RequestTimeoutSeconds int      `db:"request_timeout_seconds" json:"request_timeout_seconds"`
	ExpectedStatusCodes   IntArray `db:"expected_status_codes" json:"expected_status_codes"`
	WebhookSecret         string   `db:"webhook_secret" json:"webhook_secret,omitempty"`
	UserAgent             string   `db:"user_agent" json:"user_agent"`
	ContentType           string   `db:"content_type" json:"content_type"`
	RateLimitRPS          *float64 `db:"rate_limit_rps" json:"rate_limit_rps,omitempty"`
}

// DefaultWebhookHTTPConfig returns default HTTP configuration
func DefaultWebhookHTTPConfig() WebhookHTTPConfig {
	return WebhookHTTPConfig{
		MaxRetries:            3,
		RetryBackoffSeconds:   60,
		CaptureResponseBody:   false, // stores up to 1 KB of response body; set true for up to 1 MB
		FollowRedirects:       true,
		VerifySSL:             true,
		RequestTimeoutSeconds: 30,
		ExpectedStatusCodes:   IntArray{200, 201, 202, 204},
		UserAgent:             "Sparrow-Webhook/1.0",
		ContentType:           "application/json",
	}
}

// GetRequestTimeout returns the request timeout as a time.Duration
func (config WebhookHTTPConfig) GetRequestTimeout() time.Duration {
	return time.Duration(config.RequestTimeoutSeconds) * time.Second
}

// GetRetryBackoff returns the retry backoff as a time.Duration
func (config WebhookHTTPConfig) GetRetryBackoff() time.Duration {
	return time.Duration(config.RetryBackoffSeconds) * time.Second
}

// IsSuccessStatusCode checks if the given status code is considered successful
func (config WebhookHTTPConfig) IsSuccessStatusCode(statusCode int) bool {
	for _, code := range config.ExpectedStatusCodes {
		if code == statusCode {
			return true
		}
	}
	return false
}

// ValidateConfig validates the HTTP configuration
func (config WebhookHTTPConfig) ValidateConfig() error {
	if config.MaxRetries < 0 || config.MaxRetries > 10 {
		return svcerrors.InvalidInputf("max_retries must be between 0 and 10, got %d", config.MaxRetries)
	}

	if config.RetryBackoffSeconds <= 0 || config.RetryBackoffSeconds > 3600 {
		return svcerrors.InvalidInputf("retry_backoff_seconds must be between 1 and 3600, got %d", config.RetryBackoffSeconds)
	}

	if config.RequestTimeoutSeconds <= 0 || config.RequestTimeoutSeconds > 300 {
		return svcerrors.InvalidInputf("request_timeout_seconds must be between 1 and 300, got %d", config.RequestTimeoutSeconds)
	}

	if len(config.ExpectedStatusCodes) == 0 {
		return svcerrors.InvalidInput("expected_status_codes cannot be empty")
	}

	for _, code := range config.ExpectedStatusCodes {
		if code < 100 || code > 599 {
			return svcerrors.InvalidInputf("invalid HTTP status code: %d", code)
		}
	}

	if config.ContentType == "" {
		return svcerrors.InvalidInput("content_type cannot be empty")
	}

	// RateLimitRPS: nil is fine (no limit), but if set, must be positive.
	// Matches the DB constraint: CHECK (rate_limit_rps IS NULL OR rate_limit_rps > 0)
	if config.RateLimitRPS != nil && *config.RateLimitRPS <= 0 {
		return svcerrors.InvalidInputf("rate_limit_rps must be positive, got %f", *config.RateLimitRPS)
	}

	return nil
}

// ApplyConfig applies configuration from another config, only overriding non-zero/non-empty values
func (config *WebhookHTTPConfig) ApplyConfig(other *WebhookHTTPConfig) {
	if other == nil {
		return
	}

	// Override non-zero numeric values
	if other.MaxRetries > 0 {
		config.MaxRetries = other.MaxRetries
	}
	if other.RetryBackoffSeconds > 0 {
		config.RetryBackoffSeconds = other.RetryBackoffSeconds
	}
	if other.RequestTimeoutSeconds > 0 {
		config.RequestTimeoutSeconds = other.RequestTimeoutSeconds
	}

	// Override non-empty arrays
	if len(other.ExpectedStatusCodes) > 0 {
		config.ExpectedStatusCodes = other.ExpectedStatusCodes
	}

	// Override non-empty strings (empty string means not set, keep default)
	if other.WebhookSecret != "" {
		config.WebhookSecret = other.WebhookSecret
	}
	if other.UserAgent != "" {
		config.UserAgent = other.UserAgent
	}
	if other.ContentType != "" {
		config.ContentType = other.ContentType
	}

	// For booleans, we can't easily distinguish "explicitly set to false" from "default false"
	// So we'll apply them directly, which means:
	// - CaptureResponseBody: false by default, user can set to true
	// - FollowRedirects: true by default, user can set to false
	// - VerifySSL: true by default, user can set to false
	// This works for most cases since users typically want to enable capture or disable SSL/redirects
	config.CaptureResponseBody = other.CaptureResponseBody
	config.FollowRedirects = other.FollowRedirects
	config.VerifySSL = other.VerifySSL

	// RateLimitRPS is a pointer: nil means "don't change", non-nil overrides (including to set a limit)
	if other.RateLimitRPS != nil {
		config.RateLimitRPS = other.RateLimitRPS
	}
}

// StringArray is a wrapper for PostgreSQL string arrays
type StringArray []string

// Scan implements the sql.Scanner interface
func (a *StringArray) Scan(value any) error {
	if value == nil {
		*a = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	case pq.StringArray:
		*a = StringArray(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into StringArray", value)
	}
}

// Value implements the driver.Valuer interface
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return pq.StringArray(a).Value()
}

// IntArray is a wrapper for PostgreSQL integer arrays
type IntArray []int

// Scan implements the sql.Scanner interface
func (a *IntArray) Scan(value any) error {
	if value == nil {
		*a = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	case pq.Int64Array:
		*a = make(IntArray, len(v))
		for i, val := range v {
			(*a)[i] = int(val)
		}
		return nil
	default:
		return fmt.Errorf("cannot scan %T into IntArray", value)
	}
}

// Value implements the driver.Valuer interface
func (a IntArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}

	int64Array := make(pq.Int64Array, len(a))
	for i, val := range a {
		int64Array[i] = int64(val)
	}
	return int64Array.Value()
}

// JSONBMap is a wrapper for PostgreSQL JSONB maps
type JSONBMap map[string]any

// Scan implements the sql.Scanner interface
func (m *JSONBMap) Scan(value any) error {
	if value == nil {
		*m = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, m)
	case string:
		return json.Unmarshal([]byte(v), m)
	default:
		return fmt.Errorf("cannot scan %T into JSONBMap", value)
	}
}

// Value implements the driver.Valuer interface
func (m JSONBMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// HTTPConfigUpdate represents HTTP configuration fields that can be updated.
// Used as an optional parameter in UpdateWebhookConfig to apply http_config changes.
type HTTPConfigUpdate struct {
	MaxRetries            int
	RetryBackoffSeconds   int
	CaptureResponseBody   bool
	FollowRedirects       bool
	VerifySSL             bool
	RequestTimeoutSeconds int
	ExpectedStatusCodes   []int
	WebhookSecret         string
	UserAgent             string
	ContentType           string
	RateLimitRPS          *float64
}

// WebhookRegistrationRequest represents a request to create/update a webhook registration
type WebhookRegistrationRequest struct {
	ID            string             `json:"id,omitempty"`
	Namespace     string             `json:"namespace" validate:"required"`
	Events        []string           `json:"events" validate:"required,min=1"`
	URL           string             `json:"url" validate:"required,url"`
	Headers       map[string]any     `json:"headers,omitempty"`
	SecretHeaders map[string]string  `json:"secret_headers,omitempty"`
	Active        *bool              `json:"active,omitempty"`
	Description   string             `json:"description,omitempty"`
	HTTPConfig    *WebhookHTTPConfig `json:"http_config,omitempty"`
	RateLimitRPS  *float64           `json:"rate_limit_rps,omitempty"`
	SignatureType string             `json:"signature_type,omitempty"` // "hmac" (default) or "ed25519"
}

// ToWebhookRegistration converts the request to a WebhookRegistration
func (req WebhookRegistrationRequest) ToWebhookRegistration() (*WebhookRegistration, error) {
	webhook := &WebhookRegistration{
		ID:            req.ID,
		Namespace:     req.Namespace,
		Events:        StringArray(req.Events),
		URL:           req.URL,
		Headers:       JSONBMap(req.Headers),
		Description:   req.Description,
		Active:        true,
		Health:        "unknown",
		SignatureType: req.SignatureType,
	}

	if req.Active != nil {
		webhook.Active = *req.Active
	}

	if req.HTTPConfig != nil {
		// Start with defaults
		webhook.HTTPConfig = DefaultWebhookHTTPConfig()

		// Apply provided configuration (this will override defaults)
		webhook.HTTPConfig.ApplyConfig(req.HTTPConfig)

		// Validate the final configuration
		if err := webhook.HTTPConfig.ValidateConfig(); err != nil {
			return nil, fmt.Errorf("invalid HTTP config: %w", err)
		}
	} else {
		webhook.HTTPConfig = DefaultWebhookHTTPConfig()
	}

	return webhook, nil
}
