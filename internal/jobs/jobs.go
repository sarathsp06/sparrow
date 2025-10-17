package jobs

import (
	"time"
)

// EventArgs represents an event processing job
type EventArgs struct {
	EventID    string            `json:"event_id"`
	Namespace  string            `json:"namespace"`
	Event      string            `json:"event"`
	Payload    string            `json:"payload"`
	TTLSeconds int64             `json:"ttl_seconds"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  time.Time         `json:"created_at"`
}

// Kind returns the job kind for River queue
func (EventArgs) Kind() string {
	return "event_processing"
}

// WebhookArgs represents a webhook delivery job
type WebhookArgs struct {
	DeliveryID string            `json:"delivery_id"`
	WebhookID  string            `json:"webhook_id"`
	EventID    string            `json:"event_id"`
	URL        string            `json:"url"`
	Headers    map[string]string `json:"headers"`
	Payload    string            `json:"payload"`
	Timeout    int               `json:"timeout"`
	ExpiresAt  time.Time         `json:"expires_at"`
	Namespace  string            `json:"namespace"`
	Event      string            `json:"event"`
}

// Kind returns the job kind for River queue
func (WebhookArgs) Kind() string {
	return "webhook_delivery"
}

// DataProcessingArgs represents a data processing job (for compatibility)
type DataProcessingArgs struct {
	DataID   int    `json:"data_id"`
	DataType string `json:"data_type"`
}

// Kind returns the job kind for River queue
func (DataProcessingArgs) Kind() string {
	return "data_processing"
}
