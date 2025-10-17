package jobs

import "time"

// WebhookArgs represents arguments for webhook delivery jobs
type WebhookArgs struct {
	DeliveryID string            `json:"delivery_id"`
	WebhookID  string            `json:"webhook_id"`
	EventID    string            `json:"event_id"`
	URL        string            `json:"url"`
	Headers    map[string]string `json:"headers"`
	Payload    string            `json:"payload"`    // JSON string
	Timeout    int               `json:"timeout"`    // timeout in seconds
	ExpiresAt  time.Time         `json:"expires_at"` // TTL expiration
	Namespace  string            `json:"namespace"`
	Event      string            `json:"event"`
}

// Kind returns the job type name
func (WebhookArgs) Kind() string { return "webhook_delivery" }

// EventArgs represents arguments for event processing jobs
type EventArgs struct {
	EventID    string            `json:"event_id"`
	Namespace  string            `json:"namespace"`
	Event      string            `json:"event"`
	Payload    string            `json:"payload"` // JSON string
	TTLSeconds int64             `json:"ttl_seconds"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  time.Time         `json:"created_at"`
}

// Kind returns the job type name
func (EventArgs) Kind() string { return "event_processing" }
