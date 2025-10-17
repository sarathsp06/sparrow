package webhooks

import (
	"time"
)

// WebhookRegistration represents a registered webhook
type WebhookRegistration struct {
	ID          string            `json:"id" db:"id"`
	Namespace   string            `json:"namespace" db:"namespace"`
	Events      []string          `json:"events" db:"events"` // Multiple events supported
	URL         string            `json:"url" db:"url"`
	Headers     map[string]string `json:"headers" db:"headers"`
	Timeout     int               `json:"timeout" db:"timeout"`
	Active      bool              `json:"active" db:"active"`
	Description string            `json:"description" db:"description"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

// EventRecord represents an event that was pushed
type EventRecord struct {
	ID        string            `json:"id" db:"id"`
	Namespace string            `json:"namespace" db:"namespace"`
	Event     string            `json:"event" db:"event"`
	Payload   string            `json:"payload" db:"payload"`
	TTL       int64             `json:"ttl" db:"ttl"`
	Metadata  map[string]string `json:"metadata" db:"metadata"`
	CreatedAt time.Time         `json:"created_at" db:"created_at"`
	ExpiresAt time.Time         `json:"expires_at" db:"expires_at"`
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
