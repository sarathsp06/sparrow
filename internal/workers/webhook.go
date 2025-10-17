package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/riverqueue/river"
	"github.com/sarathsp06/httpqueue/internal/jobs"
	"github.com/sarathsp06/httpqueue/internal/logger"
)

// WebhookWorker handles webhook jobs
type WebhookWorker struct {
	river.WorkerDefaults[jobs.WebhookArgs]
}

// Work processes the webhook job
func (w WebhookWorker) Work(ctx context.Context, job *river.Job[jobs.WebhookArgs]) error {
	log := logger.NewLogger("webhook-worker")

	method := w.getMethod(job.Args)
	timeout := w.getTimeout(job.Args)

	log.Info("Processing webhook job",
		"job_id", job.ID,
		"url", job.Args.URL,
		"method", method,
		"payload_keys", len(job.Args.Payload),
		"timeout", timeout,
	)

	// Prepare the request
	payloadBytes, err := json.Marshal(job.Args.Payload)
	if err != nil {
		log.Error("Failed to marshal payload",
			"job_id", job.ID,
			"error", err,
		)
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, job.Args.URL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Error("Failed to create request",
			"job_id", job.ID,
			"url", job.Args.URL,
			"method", method,
			"error", err,
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set default Content-Type
	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range job.Args.Headers {
		req.Header.Set(key, value)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// Send the request
	startTime := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		log.Error("Failed to send webhook",
			"job_id", job.ID,
			"url", job.Args.URL,
			"method", method,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Log the response
	log.Info("Webhook response received",
		"job_id", job.ID,
		"url", job.Args.URL,
		"method", method,
		"status_code", resp.StatusCode,
		"status", resp.Status,
		"duration_ms", duration.Milliseconds(),
	)

	// Consider 2xx status codes as success
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Info("Webhook sent successfully",
			"job_id", job.ID,
			"url", job.Args.URL,
			"status_code", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
		)
		return nil
	}

	// For non-2xx responses, return an error to potentially retry
	log.Warn("Webhook returned non-success status",
		"job_id", job.ID,
		"url", job.Args.URL,
		"status_code", resp.StatusCode,
		"status", resp.Status,
		"duration_ms", duration.Milliseconds(),
	)
	return fmt.Errorf("webhook returned non-success status: %d %s", resp.StatusCode, resp.Status)
}

func (w WebhookWorker) getMethod(args jobs.WebhookArgs) string {
	if args.Method == "" {
		return "POST"
	}
	return args.Method
}

func (w WebhookWorker) getTimeout(args jobs.WebhookArgs) int {
	if args.Timeout <= 0 {
		return 30 // default 30 seconds
	}
	return args.Timeout
}
