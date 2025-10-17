package jobs

// WebhookArgs represents arguments for webhook jobs
type WebhookArgs struct {
	URL     string                 `json:"url"`
	Payload map[string]interface{} `json:"payload"`
	Headers map[string]string      `json:"headers,omitempty"`
	Method  string                 `json:"method,omitempty"`  // defaults to POST
	Timeout int                    `json:"timeout,omitempty"` // timeout in seconds, defaults to 30
}

// Kind returns the job type name
func (WebhookArgs) Kind() string { return "webhook" }
