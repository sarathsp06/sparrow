package store

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/sarathsp06/sparrow/pkg/types"
)

// WebhookHealth represents the health status of a webhook
type WebhookHealth string

const (
	HealthHealthy   WebhookHealth = "healthy"   // Recent deliveries successful
	HealthDegraded  WebhookHealth = "degraded"  // Some failures but not critical
	HealthUnhealthy WebhookHealth = "unhealthy" // High failure rate or consecutive failures
	HealthUnknown   WebhookHealth = "unknown"   // No recent delivery attempts
)

// WebhookRegistration represents a registered webhook
type WebhookRegistration struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Namespace string    `json:"namespace" db:"namespace"`
	// Events removed in favor of EventSubscription
	URL         string                    `json:"url" db:"url"`
	Headers     types.Map[string, string] `json:"headers" db:"headers"`
	Timeout     int                       `json:"timeout" db:"timeout"`
	Active      bool                      `json:"active" db:"active"`
	Description string                    `json:"description" db:"description"`
	Health      WebhookHealth             `json:"health" db:"health"`
	// HTTP Configuration
	MaxRetries            int           `json:"max_retries" db:"max_retries"`
	RetryBackoffSeconds   int           `json:"retry_backoff_seconds" db:"retry_backoff_seconds"`
	CaptureResponseBody   bool          `json:"capture_response_body" db:"capture_response_body"`
	FollowRedirects       bool          `json:"follow_redirects" db:"follow_redirects"`
	VerifySSL             bool          `json:"verify_ssl" db:"verify_ssl"`
	RequestTimeoutSeconds int           `json:"request_timeout_seconds" db:"request_timeout_seconds"`
	ExpectedStatusCodes   pq.Int64Array `json:"expected_status_codes" db:"expected_status_codes"`
	WebhookSecret         string        `json:"webhook_secret" db:"webhook_secret"`
	UserAgent             string        `json:"user_agent" db:"user_agent"`
	ContentType           string        `json:"content_type" db:"content_type"`
	CreatedAt             time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at" db:"updated_at"`
}

// EventRecord represents an event that was pushed
type EventRecord struct {
	ID        uuid.UUID                 `json:"id" db:"id"`
	Namespace string                    `json:"namespace" db:"namespace"`
	Event     string                    `json:"event" db:"event"`
	Payload   types.Map[string, any]    `json:"payload" db:"payload"`
	TTL       int64                     `json:"ttl" db:"ttl"`
	Metadata  types.Map[string, string] `json:"metadata" db:"metadata"`
	CreatedAt time.Time                 `json:"created_at" db:"created_at"`
	ExpiresAt time.Time                 `json:"expires_at" db:"expires_at"`
}

// WebhookDelivery represents a webhook delivery attempt
type WebhookDelivery struct {
	ID              uuid.UUID             `json:"id" db:"id"`
	WebhookID       uuid.UUID             `json:"webhook_id" db:"webhook_id"`
	EventID         uuid.UUID             `json:"event_id" db:"event_id"`
	SubscriptionID  *uuid.UUID            `json:"subscription_id,omitempty" db:"subscription_id"`
	Status          WebhookDeliveryStatus `json:"status" db:"status"`
	AttemptCount    int                   `json:"attempt_count" db:"attempt_count"`
	MaxAttempts     int                   `json:"max_attempts" db:"max_attempts"`
	CreatedAt       time.Time             `json:"created_at" db:"created_at"`
	LastAttemptedAt *time.Time            `json:"last_attempted_at" db:"last_attempted_at"`
	NextRetryAt     *time.Time            `json:"next_retry_at" db:"next_retry_at"`
	ExpiresAt       time.Time             `json:"expires_at" db:"expires_at"`
	ResponseCode    int                   `json:"response_code" db:"response_code"`
	ResponseBody    string                `json:"response_body" db:"response_body"`
	ErrorMessage    string                `json:"error_message" db:"error_message"`
	RequestBody     string                `json:"request_body" db:"request_body"`
}

// WebhookDeliveryStatus represents the status of a webhook delivery
type WebhookDeliveryStatus string

const (
	StatusPending  WebhookDeliveryStatus = "pending"
	StatusSending  WebhookDeliveryStatus = "sending"
	StatusSuccess  WebhookDeliveryStatus = "success"
	StatusFailed   WebhookDeliveryStatus = "failed"
	StatusRetrying WebhookDeliveryStatus = "retrying"
	StatusExpired  WebhookDeliveryStatus = "expired"
)

// WebhookHealthEvent represents a single delivery event for time-series tracking
type WebhookHealthEvent struct {
	ID           uuid.UUID `json:"id" db:"id"`
	WebhookID    uuid.UUID `json:"webhook_id" db:"webhook_id"`
	DeliveryID   uuid.UUID `json:"delivery_id" db:"delivery_id"`
	Success      bool      `json:"success" db:"success"`
	ResponseTime int       `json:"response_time" db:"response_time"` // milliseconds
	ResponseCode int       `json:"response_code" db:"response_code"`
	ErrorMessage string    `json:"error_message" db:"error_message"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
}

// EventReportWithStats represents an event with delivery statistics
type EventReportWithStats struct {
	EventRecord
	WebhookCount         int32 `json:"webhook_count" db:"webhook_count"`
	SuccessfulDeliveries int32 `json:"successful_deliveries" db:"successful_deliveries"`
	FailedDeliveries     int32 `json:"failed_deliveries" db:"failed_deliveries"`
	PendingDeliveries    int32 `json:"pending_deliveries" db:"pending_deliveries"`
}

// WebhookHealthSummary represents aggregated health metrics for a webhook
type WebhookHealthSummary struct {
	ID                   uuid.UUID `json:"id" db:"id"`
	WebhookID            uuid.UUID `json:"webhook_id" db:"webhook_id"`
	WindowStart          time.Time `json:"window_start" db:"window_start"`
	WindowEnd            time.Time `json:"window_end" db:"window_end"`
	TotalDeliveries      int       `json:"total_deliveries" db:"total_deliveries"`
	SuccessfulDeliveries int       `json:"successful_deliveries" db:"successful_deliveries"`
	FailedDeliveries     int       `json:"failed_deliveries" db:"failed_deliveries"`
	SuccessRate          float64   `json:"success_rate" db:"success_rate"`           // 0.0 to 1.0
	AvgResponseTime      int       `json:"avg_response_time" db:"avg_response_time"` // milliseconds
	MinResponseTime      int       `json:"min_response_time" db:"min_response_time"` // milliseconds
	MaxResponseTime      int       `json:"max_response_time" db:"max_response_time"` // milliseconds
	P95ResponseTime      int       `json:"p95_response_time" db:"p95_response_time"` // milliseconds
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
}

// WebhookHealthMetrics represents current health state and recent metrics
type WebhookHealthMetrics struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	WebhookID           uuid.UUID  `json:"webhook_id" db:"webhook_id"`
	ConsecutiveFailures int        `json:"consecutive_failures" db:"consecutive_failures"`
	LastSuccessAt       *time.Time `json:"last_success_at" db:"last_success_at"`
	LastFailureAt       *time.Time `json:"last_failure_at" db:"last_failure_at"`
	LastEventAt         *time.Time `json:"last_event_at" db:"last_event_at"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
}

// EventRegistration represents a registered event type
type EventRegistration struct {
	ID            uuid.UUID                 `json:"id" db:"id"`
	Name          string                    `json:"name" db:"name"`
	Description   string                    `json:"description" db:"description"`
	Schema        types.Map[string, any]    `json:"schema" db:"schema"`                 // JSON schema for validation
	SamplePayload types.Map[string, any]    `json:"sample_payload" db:"sample_payload"` // Auto-generated sample payload
	Metadata      types.Map[string, string] `json:"metadata" db:"metadata"`
	Active        bool                      `json:"active" db:"active"`
	CreatedAt     time.Time                 `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at" db:"updated_at"`
}

// WebhookUpdateFields represents fields that can be updated for a webhook
type WebhookUpdateFields struct {
	Events      []string                  `json:"events"`
	URL         string                    `json:"url"`
	Headers     types.Map[string, string] `json:"headers"`
	Timeout     int                       `json:"timeout"`
	Active      bool                      `json:"active"`
	Description string                    `json:"description"`
}

// EventSubscription represents a subscription to an event for a webhook
type EventSubscription struct {
	ID                uuid.UUID                 `json:"id" db:"id"`
	WebhookID         uuid.UUID                 `json:"webhook_id" db:"webhook_id"`
	EventName         string                    `json:"event_name" db:"event_name"`
	Namespace         string                    `json:"namespace" db:"namespace"`
	Headers           types.Map[string, string] `json:"headers" db:"headers"`
	Method            string                    `json:"method" db:"method"`
	TransformEnabled  bool                      `json:"transform_enabled" db:"transform_enabled"`
	TransformTemplate string                    `json:"transform_template" db:"transform_template"`
	Timeout           int                       `json:"timeout" db:"timeout"`
	CreatedAt         time.Time                 `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at" db:"updated_at"`
}

// NamespaceStats represents statistics for a namespace
type NamespaceStats struct {
	TotalWebhooks        int     `db:"total_webhooks"`
	ActiveWebhooks       int     `db:"active_webhooks"`
	TotalDeliveries      int     `db:"total_deliveries"`
	SuccessfulDeliveries int     `db:"successful_deliveries"`
	FailedDeliveries     int     `db:"failed_deliveries"`
	PendingDeliveries    int     `db:"pending_deliveries"`
	SuccessRate          float64 `db:"success_rate"`
}

// SubscriptionWithWebhook represents a subscription joined with its webhook configuration
type SubscriptionWithWebhook struct {
	Subscription *EventSubscription
	Webhook      *WebhookRegistration
}
