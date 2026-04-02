package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	structpb "google.golang.org/protobuf/types/known/structpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/sarathsp06/sparrow/proto"
)

func MainGRPC() {
	// Connect to the gRPC server
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close() //nolint:errcheck

	webhookClient := pb.NewWebhookServiceClient(conn)
	eventClient := pb.NewEventServiceClient(conn)
	subscriptionClient := pb.NewSubscriptionServiceClient(conn)
	deliveryClient := pb.NewDeliveryServiceClient(conn)
	healthClient := pb.NewHealthServiceClient(conn)

	ctx := context.Background()

	// Example 0: Register events
	log.Println("=== Example 0: Register Events ===")
	events := []struct {
		name        string
		description string
		schema      string
		metadata    map[string]string
	}{
		{
			name:        "signup",
			description: "User signup event",
			schema:      `{"type": "object", "properties": {"user_id": {"type": "string"}, "email": {"type": "string"}}, "required": ["user_id", "email"]}`,
			metadata: map[string]string{
				"category": "user",
			},
		},
		{
			name:        "login",
			description: "User login event",
			schema:      `{"type": "object", "properties": {"user_id": {"type": "string"}, "login_time": {"type": "string"}}, "required": ["user_id", "login_time"]}`,
			metadata: map[string]string{
				"category": "user",
			},
		},
		{
			name:        "order.created",
			description: "Order created event",
			schema:      `{"type": "object", "properties": {"order_id": {"type": "string"}, "total": {"type": "number"}}}`,
			metadata: map[string]string{
				"category": "order",
			},
		},
	}

	for _, event := range events {
		var schemaMap map[string]any
		if err := json.Unmarshal([]byte(event.schema), &schemaMap); err != nil {
			log.Fatalf("Failed to parse schema for %s: %v", event.name, err)
		}
		schemaStruct, err := structpb.NewStruct(schemaMap)
		if err != nil {
			log.Fatalf("Failed to create schema struct for %s: %v", event.name, err)
		}

		regEventReq := &pb.RegisterEventRequest{
			Name:        event.name,
			Description: event.description,
			Schema:      schemaStruct,
			Metadata:    event.metadata,
			Active:      true,
		}
		regEventResp, err := eventClient.RegisterEvent(ctx, regEventReq)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				log.Printf("Event %s already registered (skipping)", event.name)
			} else {
				log.Printf("Failed to register event %s: %v", event.name, err)
			}
		} else {
			log.Printf("Event %s registered: (ID: %s)", event.name, regEventResp.GetEventId()) //nolint:staticcheck // deprecated field, no replacement available yet
		}
	}

	// Example 1: Register a webhook (creates subscriptions automatically)
	log.Println("\n=== Example 1: Register Webhook for Multiple User Events ===")
	registerReq := &pb.RegisterWebhookRequest{
		Namespace: "default",
		Events:    []string{"signup", "login"}, // This creates 2 subscriptions
		Url:       "https://testhooks.sarathsadasivan.com/hooks",
		Headers: map[string]string{
			"Authorization": "Bearer secret-token",
			"X-App-Name":    "MyApp",
		},
		Timeout:     30,
		Active:      true,
		Description: "Webhook for user-related events",
	}

	registerResp, err := webhookClient.RegisterWebhook(ctx, registerReq)
	if err != nil {
		log.Printf("Failed to register webhook: %v", err)
	} else {
		log.Printf("Webhook registered successfully:")
		log.Printf("  Webhook ID: %s", registerResp.WebhookId)
		log.Printf("  Created At: %s", registerResp.CreatedAt.AsTime())
		log.Printf("  Note: Created 2 event subscriptions (signup, login)")
	}

	// Example 1b: Create a direct subscription with template transformation + label filters
	log.Println("\n=== Example 1b: Create Subscription with Template + Label Filters ===")
	if registerResp != nil {
		// Create a subscription with Slack BlockKit template that only fires for
		// premium signups in the us-east-1 region (label-based filtering).
		slackTemplate := `{
  "blocks": [
    {
      "type": "header",
      "text": {
        "type": "plain_text",
        "text": "🎉 New User Signup"
      }
    },
    {
      "type": "section",
      "fields": [
        {
          "type": "mrkdwn",
          "text": "*User ID:*\n{{ .Payload.user_id }}"
        },
        {
          "type": "mrkdwn",
          "text": "*Email:*\n{{ .Payload.email }}"
        }
      ]
    }
  ]
}`

		createSubReq := &pb.CreateSubscriptionRequest{
			WebhookId:         registerResp.WebhookId,
			EventName:         "signup",
			Namespace:         "default",
			TransformEnabled:  true,
			TransformTemplate: slackTemplate,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			// Label filters: only events with BOTH these labels will trigger this subscription.
			// Events without labels, or with different label values, are skipped.
			LabelFilters: map[string]string{
				"plan":   "premium",
				"region": "us-east-1",
			},
		}

		createSubResp, err := subscriptionClient.CreateSubscription(ctx, createSubReq)
		if err != nil {
			log.Printf("Failed to create subscription: %v", err)
		} else {
			log.Printf("Subscription created successfully:")
			log.Printf("  Subscription ID: %s", createSubResp.SubscriptionId)
			log.Printf("  Features: Template transformation + label-based filtering")
			log.Printf("  Label Filters: plan=premium, region=us-east-1")
		}
	}

	// Example 1c: List subscriptions for a webhook
	log.Println("\n=== Example 1c: List Subscriptions for Webhook ===")
	if registerResp != nil {
		listSubsReq := &pb.ListSubscriptionsRequest{
			WebhookId: registerResp.WebhookId,
			Namespace: "default",
		}

		listSubsResp, err := subscriptionClient.ListSubscriptions(ctx, listSubsReq)
		if err != nil {
			log.Printf("Failed to list subscriptions: %v", err)
		} else {
			log.Printf("Found %d subscriptions:", listSubsResp.Pagination.TotalCount)
			for i, sub := range listSubsResp.Subscriptions {
				log.Printf("  Subscription %d:", i+1)
				log.Printf("    ID: %s", sub.SubscriptionId)
				log.Printf("    Event: %s", sub.EventName)
				log.Printf("    Transform Enabled: %t", sub.TransformEnabled)
				if sub.TransformEnabled {
					log.Printf("    Has Template: yes (%d chars)", len(sub.TransformTemplate))
				}
			}
		}
	}

	// Example 2: Register webhook with advanced HTTP configuration
	log.Println("\n=== Example 2: Register Webhook with Advanced HTTP Config ===")
	registerReq2 := &pb.RegisterWebhookRequest{
		Namespace: "default",
		Events:    []string{"order.created"},
		Url:       "https://testhooks.sarathsadasivan.com/hooks",
		Headers: map[string]string{
			"Content-Type":   "application/json",
			"X-Service-Name": "OrderProcessor",
		},
		Timeout:     15,
		Active:      true,
		Description: "Webhook for order events with custom config",
		HttpConfig: &pb.WebhookHTTPConfig{
			MaxRetries:            3,
			RetryBackoffSeconds:   5,
			CaptureResponseBody:   true,
			FollowRedirects:       false,
			VerifySsl:             true,
			RequestTimeoutSeconds: 15,
			ExpectedStatusCodes:   []int32{200, 201, 202},
			WebhookSecret:         "my-webhook-secret",
			UserAgent:             "Sparrow-Webhook/2.0",
			ContentType:           "application/json",
		},
	}

	registerResp2, err := webhookClient.RegisterWebhook(ctx, registerReq2)
	if err != nil {
		log.Printf("Failed to register order webhook: %v", err)
	} else {
		log.Printf("Order webhook registered successfully:")
		log.Printf("  Webhook ID: %s", registerResp2.WebhookId)
		log.Printf("  Features: HMAC signing, custom retry logic, response capture")
	}

	// Example 3: List registered webhooks (shows events from subscriptions)
	log.Println("\n=== Example 3: List Webhooks in Namespace ===")
	listReq := &pb.ListWebhooksRequest{
		Namespace:  "default",
		ActiveOnly: true,
	}

	listResp, err := webhookClient.ListWebhooks(ctx, listReq)
	if err != nil {
		log.Printf("Failed to list webhooks: %v", err)
	} else {
		log.Printf("Found %d webhooks in default namespace:", listResp.Pagination.TotalCount)
		for i, webhook := range listResp.Webhooks {
			log.Printf("  Webhook %d:", i+1)
			log.Printf("    ID: %s", webhook.WebhookId)
			log.Printf("    Events (from subscriptions): %v", webhook.Events)
			log.Printf("    URL: %s", webhook.Url)
			log.Printf("    Active: %t", webhook.Active)
		}
	}

	// Wait a moment before pushing events
	time.Sleep(2 * time.Second)

	// Example 4: Push a user signup event with labels
	log.Println("\n=== Example 4: Push User Signup Event (with labels) ===")
	eventPayload := map[string]any{
		"user_id":   "user_12345",
		"email":     "john.doe@default.com",
		"name":      "John Doe",
		"signup_at": time.Now().Unix(),
		"plan":      "premium",
		"source":    "web",
	}
	payload, err := structpb.NewStruct(eventPayload)
	if err != nil {
		log.Fatalf("Failed to create payload: %v", err)
	}

	pushReq := &pb.PushEventRequest{
		Namespace:  "default",
		Event:      "signup",
		Payload:    payload,
		TtlSeconds: 3600, // 1 hour TTL
		Metadata: map[string]string{
			"source":   "api",
			"trace_id": "trace_abc123",
		},
		// Labels enable selective subscription matching.
		// The subscription from Example 1b has label_filters: plan=premium, region=us-east-1.
		// This event matches because its labels are a superset of the subscription's filters.
		// Subscriptions without label_filters (Example 1) still match all events.
		Labels: map[string]string{
			"plan":   "premium",
			"region": "us-east-1",
			"tier":   "enterprise", // extra labels are fine -- only filter keys must match
		},
	}

	pushResp, err := eventClient.PushEvent(ctx, pushReq)
	if err != nil {
		log.Printf("Failed to push event: %v", err)
	} else {
		log.Printf("Event pushed successfully:")
		log.Printf("  Event ID: %s", pushResp.EventId)
		log.Printf("  Labels: plan=premium, region=us-east-1, tier=enterprise")
		log.Printf("  Will match: subscription from Ex 1 (no filters) + Ex 1b (labels match)")
	}

	// Example 5: Push an order created event
	log.Println("\n=== Example 5: Push Order Created Event ===")
	orderPayload := map[string]any{
		"order_id":     "order_67890",
		"customer_id":  "user_12345",
		"total_amount": 99.99,
		"currency":     "USD",
		"items": []any{
			map[string]any{"product_id": "prod_1", "quantity": 2, "price": 29.99},
			map[string]any{"product_id": "prod_2", "quantity": 1, "price": 39.99},
		},
		"created_at": time.Now().Unix(),
	}

	payload, err = structpb.NewStruct(orderPayload)
	if err != nil {
		log.Fatalf("Failed to create payload: %v", err)
	}

	pushOrderReq := &pb.PushEventRequest{
		Namespace:  "default",
		Event:      "order.created",
		Payload:    payload,
		TtlSeconds: 1800, // 30 minutes TTL
		Metadata: map[string]string{
			"payment_method":  "credit_card",
			"shipping_method": "express",
		},
	}

	pushOrderResp, err := eventClient.PushEvent(ctx, pushOrderReq)
	if err != nil {
		log.Printf("Failed to push order event: %v", err)
	} else {
		log.Printf("Order event pushed successfully:")
		log.Printf("  Event ID: %s", pushOrderResp.EventId)
	}

	// Wait for webhook processing
	time.Sleep(5 * time.Second)

	// Example 6: Check webhook delivery history
	log.Println("\n=== Example 6: Check Webhook Delivery History ===")
	if registerResp != nil {
		statusReq := &pb.ListDeliveriesRequest{
			WebhookId: registerResp.WebhookId,
			Namespace: "default",
		}

		statusResp, err := deliveryClient.ListDeliveries(ctx, statusReq)
		if err != nil {
			log.Printf("Failed to get deliveries: %v", err)
		} else {
			log.Printf("Webhook deliveries:")
			log.Printf("  Total Deliveries: %d", statusResp.Pagination.TotalCount)
			for i, delivery := range statusResp.Deliveries {
				log.Printf("  Delivery %d:", i+1)
				log.Printf("    ID: %s", delivery.DeliveryId)
				log.Printf("    Status: %s", delivery.Status)
				log.Printf("    Attempt Count: %d/%d", delivery.AttemptCount, delivery.MaxAttempts)
				log.Printf("    Response Code: %d", delivery.ResponseCode)
				if delivery.ErrorMessage != "" {
					log.Printf("    Error: %s", delivery.ErrorMessage)
				}
			}
		}
	}

	// Example 7: Get Webhook Health Metrics
	log.Println("\n=== Example 7: Get Webhook Health Metrics ===")
	if registerResp != nil {
		healthReq := &pb.GetWebhookHealthRequest{
			WebhookId: registerResp.WebhookId,
			Namespace: "default",
		}

		healthResp, err := healthClient.GetWebhookHealth(ctx, healthReq)
		if err != nil {
			log.Printf("Failed to get webhook health: %v", err)
		} else {
			log.Printf("Webhook health status:")
			log.Printf("  Webhook ID: %s", healthResp.WebhookId)
			log.Printf("  Health: %s", healthResp.Health)
			if healthResp.Metrics != nil {
				log.Printf("  Health Metrics:")
				log.Printf("    Total Deliveries: %d", healthResp.Metrics.TotalDeliveries)
				log.Printf("    Successful Deliveries: %d", healthResp.Metrics.SuccessfulDeliveries)
				log.Printf("    Failed Deliveries: %d", healthResp.Metrics.FailedDeliveries)
				log.Printf("    Success Rate: %.2f%%", healthResp.Metrics.SuccessRate*100)
				log.Printf("    Consecutive Failures: %d", healthResp.Metrics.ConsecutiveFailures)
				log.Printf("    Avg Response Time: %dms", healthResp.Metrics.AvgResponseTime)
			}
		}
	}

	// Example 8: Update webhook configuration (updates subscriptions)
	log.Println("\n=== Example 8: Update Webhook Configuration ===")
	if registerResp != nil {
		updateReq := &pb.UpdateWebhookConfigRequest{
			WebhookId: registerResp.WebhookId,
			Namespace: "default",
			Updates: &pb.WebhookUpdateFields{
				Events:      []string{"signup", "login", "order.created"}, // Add order.created subscription
				Url:         "https://testhooks.sarathsadasivan.com/hooks",
				Active:      true,
				Description: "Updated webhook with order events",
			},
		}

		_, err := webhookClient.UpdateWebhookConfig(ctx, updateReq)
		if err != nil {
			log.Printf("Failed to update webhook: %v", err)
		} else {
			log.Printf("Webhook updated successfully:")
			log.Printf("  Note: Old subscriptions deleted, new ones created")
		}
	}

	// Example 9: Get Health Summary
	log.Println("\n=== Example 9: Get Health Summary ===")
	healthSummaryReq := &pb.GetHealthSummaryRequest{}

	healthSummaryResp, err := healthClient.GetHealthSummary(ctx, healthSummaryReq)
	if err != nil {
		log.Printf("Failed to get health summary: %v", err)
	} else {
		log.Printf("Webhook health summary:")
		if healthSummaryResp.Summary != nil {
			log.Printf("  Total Webhooks: %d", healthSummaryResp.Summary.TotalCount)
			log.Printf("  Healthy: %d", healthSummaryResp.Summary.HealthyCount)
			log.Printf("  Degraded: %d", healthSummaryResp.Summary.DegradedCount)
			log.Printf("  Unhealthy: %d", healthSummaryResp.Summary.UnhealthyCount)
			log.Printf("  Unknown: %d", healthSummaryResp.Summary.UnknownCount)
		}
	}

	// Example 10: Unregister a webhook (deletes subscriptions)
	log.Println("\n=== Example 10: Unregister Webhook ===")
	if registerResp2 != nil {
		unregisterReq := &pb.UnregisterWebhookRequest{
			WebhookId: registerResp2.WebhookId,
			Namespace: "default",
		}

		_, err := webhookClient.UnregisterWebhook(ctx, unregisterReq)
		if err != nil {
			log.Printf("Failed to unregister webhook: %v", err)
		} else {
			log.Printf("Webhook unregistered successfully:")
			log.Printf("  Note: Associated subscriptions also deleted")
		}
	}

	// Example 11: Get Template Functions
	log.Println("\n=== Example 11: Get Template Functions ===")
	templateFuncsReq := &pb.GetTemplateFunctionsRequest{}
	templateFuncsResp, err := webhookClient.GetTemplateFunctions(ctx, templateFuncsReq)
	if err != nil {
		log.Printf("Failed to get template functions: %v", err)
	} else {
		log.Printf("Template functions retrieved successfully:")
		log.Printf("  Total Functions: %d", templateFuncsResp.TotalCount)
		log.Println("  Available Functions:")
		for i, fn := range templateFuncsResp.Functions {
			if i < 5 { // Show first 5 functions to keep output manageable
				log.Printf("    - %s: %s", fn.Name, extractFirstLine(fn.Description))
			}
		}
		if len(templateFuncsResp.Functions) > 5 {
			log.Printf("    ... and %d more functions", len(templateFuncsResp.Functions)-5)
		}
	}

	log.Println("\n=== All examples completed ===")
	log.Println("\nKey Concepts Demonstrated:")
	log.Println("  - Namespace isolation: webhooks and events scoped to namespaces")
	log.Println("  - Label-based filtering: subscriptions match events by labels (key=value pairs)")
	log.Println("  - Subscriptions with no label_filters receive ALL events for their event type")
	log.Println("  - Template-based payload transformation per subscription")
	log.Println("  - HMAC-SHA256 signing for webhook security")
	log.Println("  - Health tracking and delivery retries with error classification")
	log.Println("  - Test your webhooks at https://testhooks.sarathsadasivan.com/")
}

// extractFirstLine extracts the first line from a multi-line description
func extractFirstLine(description string) string {
	lines := strings.SplitSeq(description, "\n")
	for line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			return trimmed
		}
	}
	return "No description available"
}

func main() {
	MainGRPC()
}
