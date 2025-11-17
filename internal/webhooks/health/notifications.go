package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/lib/pq"
)

// WebhookHealthEventNotification represents a webhook health event notification from PostgreSQL
type WebhookHealthEventNotification struct {
	WebhookID    string    `json:"webhook_id"`
	Success      bool      `json:"success"`
	ResponseCode int       `json:"response_code"`
	ResponseTime int       `json:"response_time"`
	Timestamp    time.Time `json:"timestamp"`
}

// WebhookRegistrationChangeNotification represents a webhook registration change notification
type WebhookRegistrationChangeNotification struct {
	Operation string `json:"operation"`
	WebhookID string `json:"webhook_id"`
	Namespace string `json:"namespace"`
	URL       string `json:"url,omitempty"`
	Active    bool   `json:"active,omitempty"`
	Health    string `json:"health,omitempty"`
	OldActive *bool  `json:"old_active,omitempty"`
	OldHealth string `json:"old_health,omitempty"`
}

// NotificationHandler handles different types of notifications
type NotificationHandler interface {
	HandleWebhookHealthEvent(ctx context.Context, event *WebhookHealthEventNotification) error
}

// NotificationListener listens for PostgreSQL notifications
type NotificationListener struct {
	listener *pq.Listener
	handler  NotificationHandler
	logger   *slog.Logger
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewNotificationListener creates a new notification listener
func NewNotificationListener(databaseURL string, handler NotificationHandler, logger *slog.Logger) *NotificationListener {
	listener := pq.NewListener(databaseURL, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			logger.Error("PostgreSQL listener error", "error", err, "event", ev)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())

	return &NotificationListener{
		listener: listener,
		handler:  handler,
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins listening for notifications
func (nl *NotificationListener) Start() error {
	// Listen for webhook health events
	err := nl.listener.Listen("webhook_health_event")
	if err != nil {
		return err
	}
	nl.logger.Info("Started listening for PostgreSQL notifications")
	go nl.listen()
	return nil
}

// Stop stops listening for notifications
func (nl *NotificationListener) Stop() error {
	nl.cancel()
	err := nl.listener.Close()
	if err != nil {
		nl.logger.Error("Error closing notification listener", "error", err)
	}
	nl.logger.Info("Stopped listening for PostgreSQL notifications")
	return err
}

// listen is the main event loop for processing notifications
func (nl *NotificationListener) listen() {
	for {
		select {
		case <-nl.ctx.Done():
			return
		case notification := <-nl.listener.Notify:
			if notification == nil {
				continue
			}

			switch notification.Channel {
			case "webhook_health_event":
				if err := nl.handleHealthEventNotification(notification.Extra); err != nil {
					nl.logger.Error("Failed to handle webhook health event notification",
						"error", err, "payload", notification.Extra)
				}
			default:
				nl.logger.Warn("Unknown notification channel", "channel", notification.Channel)
			}
		}
	}
}

// handleHealthEventNotification processes webhook health event notifications
func (nl *NotificationListener) handleHealthEventNotification(payload string) error {
	var event WebhookHealthEventNotification
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return err
	}

	return nl.handler.HandleWebhookHealthEvent(nl.ctx, &event)
}

// DefaultNotificationHandler provides a basic implementation of NotificationHandler
type DefaultNotificationHandler struct {
	logger *slog.Logger
}

// NewDefaultNotificationHandler creates a new default notification handler
func NewDefaultNotificationHandler(logger *slog.Logger) *DefaultNotificationHandler {
	return &DefaultNotificationHandler{logger: logger}
}

// HandleWebhookHealthEvent handles webhook health event notifications
func (dnh *DefaultNotificationHandler) HandleWebhookHealthEvent(ctx context.Context, event *WebhookHealthEventNotification) error {
	dnh.logger.Info("Webhook health event received",
		"webhook_id", event.WebhookID,
		"success", event.Success,
		"response_code", event.ResponseCode,
		"response_time", event.ResponseTime,
		"timestamp", event.Timestamp)
	return nil
}
