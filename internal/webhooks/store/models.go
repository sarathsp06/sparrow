package store

import (
	"time"

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
	ID          string                    `json:"id" db:"id"`
	Namespace   string                    `json:"namespace" db:"namespace"`
	Events      pq.StringArray            `json:"events" db:"events"` // Multiple events supported
	URL         string                    `json:"url" db:"url"`
	Headers     types.Map[string, string] `json:"headers" db:"headers"`
	Timeout     int                       `json:"timeout" db:"timeout"`
	Active      bool                      `json:"active" db:"active"`
	Description string                    `json:"description" db:"description"`
	Health      WebhookHealth             `json:"health" db:"health"`
	CreatedAt   time.Time                 `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at" db:"updated_at"`
}

// EventRecord represents an event that was pushed
type EventRecord struct {
	ID        string                    `json:"id" db:"id"`
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
	ID              string                `json:"id" db:"id"`
	WebhookID       string                `json:"webhook_id" db:"webhook_id"`
	EventID         string                `json:"event_id" db:"event_id"`
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
	ID           string    `json:"id" db:"id"`
	WebhookID    string    `json:"webhook_id" db:"webhook_id"`
	DeliveryID   string    `json:"delivery_id" db:"delivery_id"`
	Success      bool      `json:"success" db:"success"`
	ResponseTime int       `json:"response_time" db:"response_time"` // milliseconds
	ResponseCode int       `json:"response_code" db:"response_code"`
	ErrorMessage string    `json:"error_message" db:"error_message"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
}

// WebhookHealthSummary represents aggregated health metrics for a webhook
type WebhookHealthSummary struct {
	ID                   string    `json:"id" db:"id"`
	WebhookID            string    `json:"webhook_id" db:"webhook_id"`
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
	ID                  string     `json:"id" db:"id"`
	WebhookID           string     `json:"webhook_id" db:"webhook_id"`
	ConsecutiveFailures int        `json:"consecutive_failures" db:"consecutive_failures"`
	LastSuccessAt       *time.Time `json:"last_success_at" db:"last_success_at"`
	LastFailureAt       *time.Time `json:"last_failure_at" db:"last_failure_at"`
	LastEventAt         *time.Time `json:"last_event_at" db:"last_event_at"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
}

// EventRegistration represents a registered event type
type EventRegistration struct {
	ID          string                    `json:"id" db:"id"`
	Name        string                    `json:"name" db:"name"`
	Description string                    `json:"description" db:"description"`
	Schema      types.Map[string, any]    `json:"schema" db:"schema"` // JSON schema for validation
	Metadata    types.Map[string, string] `json:"metadata" db:"metadata"`
	Active      bool                      `json:"active" db:"active"`
	CreatedAt   time.Time                 `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at" db:"updated_at"`
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
