package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/sarathsp06/httpqueue/proto"
)

func main() {
	// Connect to the gRPC server
	conn, err := grpc.Dial("0.0.0.0:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewWebhookServiceClient(conn)
	ctx := context.Background()

	// Example 1: Register a webhook for multiple user events
	log.Println("=== Example 1: Register Webhook for Multiple User Events ===")
	registerReq := &pb.RegisterWebhookRequest{
		Namespace: "user",
		Events:    []string{"signup", "login", "profile_update"},
		Url:       "https://webhooks.sarathsadasivan.com/32c5c978-30ed-49d6-aafc-fda9e7fcdc33",
		Headers: map[string]string{
			"Authorization": "Bearer secret-token",
			"X-App-Name":    "MyApp",
		},
		Timeout:     30,
		Active:      true,
		Description: "Webhook for user-related events",
	}

	registerResp, err := client.RegisterWebhook(ctx, registerReq)
	if err != nil {
		log.Printf("Failed to register webhook: %v", err)
	} else {
		log.Printf("Webhook registered successfully:")
		log.Printf("  Webhook ID: %s", registerResp.WebhookId)
		log.Printf("  Success: %t", registerResp.Success)
		log.Printf("  Message: %s", registerResp.Message)
		log.Printf("  Created At: %s", time.Unix(registerResp.CreatedAt, 0))
	}

	// Example 2: Register another webhook for order events
	log.Println("\n=== Example 2: Register Webhook for Order Events ===")
	registerReq2 := &pb.RegisterWebhookRequest{
		Namespace: "order",
		Events:    []string{"created", "updated", "cancelled"},
		Url:       "https://webhooks.sarathsadasivan.com/32c5c978-30ed-49d6-aafc-fda9e7fcdc33",
		Headers: map[string]string{
			"Content-Type":   "application/json",
			"X-Service-Name": "OrderProcessor",
		},
		Timeout:     15,
		Active:      true,
		Description: "Webhook for order lifecycle events",
	}

	registerResp2, err := client.RegisterWebhook(ctx, registerReq2)
	if err != nil {
		log.Printf("Failed to register order webhook: %v", err)
	} else {
		log.Printf("Order webhook registered successfully:")
		log.Printf("  Webhook ID: %s", registerResp2.WebhookId)
		log.Printf("  Success: %t", registerResp2.Success)
	}

	// Example 3: Register webhook for payment events
	log.Println("\n=== Example 3: Register Webhook for Payment Events ===")
	registerReq3 := &pb.RegisterWebhookRequest{
		Namespace: "payment",
		Events:    []string{"processed", "failed", "refunded"},
		Url:       "https://webhooks.sarathsadasivan.com/32c5c978-30ed-49d6-aafc-fda9e7fcdc33",
		Headers: map[string]string{
			"X-Event-Type": "payment-events",
			"X-Secret":     "payment-webhook-secret",
		},
		Timeout:     20,
		Active:      true,
		Description: "Webhook for payment processing events",
	}

	registerResp3, err := client.RegisterWebhook(ctx, registerReq3)
	if err != nil {
		log.Printf("Failed to register payment webhook: %v", err)
	} else {
		log.Printf("Payment webhook registered successfully:")
		log.Printf("  Webhook ID: %s", registerResp3.WebhookId)
	}

	// Example 4: List registered webhooks
	log.Println("\n=== Example 4: List Webhooks in User Namespace ===")
	listReq := &pb.ListWebhooksRequest{
		Namespace:  "user",
		ActiveOnly: true,
	}

	listResp, err := client.ListWebhooks(ctx, listReq)
	if err != nil {
		log.Printf("Failed to list webhooks: %v", err)
	} else {
		log.Printf("Found %d webhooks in user namespace:", listResp.TotalCount)
		for i, webhook := range listResp.Webhooks {
			log.Printf("  Webhook %d:", i+1)
			log.Printf("    ID: %s", webhook.WebhookId)
			log.Printf("    Events: %v", webhook.Events)
			log.Printf("    URL: %s", webhook.Url)
			log.Printf("    Active: %t", webhook.Active)
		}
	}

	// Wait a moment before pushing events
	time.Sleep(2 * time.Second)

	// Example 5: Push a user signup event
	log.Println("\n=== Example 5: Push User Signup Event ===")
	eventPayload := map[string]interface{}{
		"user_id":   "user_12345",
		"email":     "john.doe@example.com",
		"name":      "John Doe",
		"signup_at": time.Now().Unix(),
		"plan":      "premium",
		"source":    "web",
	}

	payloadJSON, _ := json.Marshal(eventPayload)

	pushReq := &pb.PushEventRequest{
		Namespace:  "user",
		Event:      "signup",
		Payload:    string(payloadJSON),
		TtlSeconds: 3600, // 1 hour TTL
		Metadata: map[string]string{
			"source":   "api",
			"region":   "us-east-1",
			"trace_id": "trace_abc123",
		},
	}

	pushResp, err := client.PushEvent(ctx, pushReq)
	if err != nil {
		log.Printf("Failed to push event: %v", err)
	} else {
		log.Printf("Event pushed successfully:")
		log.Printf("  Event ID: %s", pushResp.EventId)
		log.Printf("  Webhooks Triggered: %d", pushResp.WebhooksTriggered)
		log.Printf("  Success: %t", pushResp.Success)
		log.Printf("  Message: %s", pushResp.Message)
		log.Printf("  Triggered Webhook IDs: %v", pushResp.WebhookIds)
	}

	// Example 6: Push an order created event
	log.Println("\n=== Example 6: Push Order Created Event ===")
	orderPayload := map[string]interface{}{
		"order_id":     "order_67890",
		"customer_id":  "user_12345",
		"total_amount": 99.99,
		"currency":     "USD",
		"items": []map[string]interface{}{
			{"product_id": "prod_1", "quantity": 2, "price": 29.99},
			{"product_id": "prod_2", "quantity": 1, "price": 39.99},
		},
		"created_at": time.Now().Unix(),
	}

	orderPayloadJSON, _ := json.Marshal(orderPayload)

	pushOrderReq := &pb.PushEventRequest{
		Namespace:  "order",
		Event:      "created",
		Payload:    string(orderPayloadJSON),
		TtlSeconds: 1800, // 30 minutes TTL
		Metadata: map[string]string{
			"payment_method":  "credit_card",
			"shipping_method": "express",
		},
	}

	pushOrderResp, err := client.PushEvent(ctx, pushOrderReq)
	if err != nil {
		log.Printf("Failed to push order event: %v", err)
	} else {
		log.Printf("Order event pushed successfully:")
		log.Printf("  Event ID: %s", pushOrderResp.EventId)
		log.Printf("  Webhooks Triggered: %d", pushOrderResp.WebhooksTriggered)
		log.Printf("  Message: %s", pushOrderResp.Message)
	}

	// Wait for webhook processing
	time.Sleep(5 * time.Second)

	// Example 7: Check webhook delivery status
	log.Println("\n=== Example 7: Check Webhook Delivery Status ===")
	if registerResp != nil && registerResp.Success {
		statusReq := &pb.GetWebhookStatusRequest{
			Identifier: &pb.GetWebhookStatusRequest_WebhookId{
				WebhookId: registerResp.WebhookId,
			},
		}

		statusResp, err := client.GetWebhookStatus(ctx, statusReq)
		if err != nil {
			log.Printf("Failed to get webhook status: %v", err)
		} else {
			log.Printf("Webhook delivery status:")
			log.Printf("  Total Deliveries: %d", statusResp.TotalDeliveries)
			log.Printf("  Success: %t", statusResp.Success)
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

	// Example 8: Unregister a webhook
	log.Println("\n=== Example 8: Unregister Webhook ===")
	if registerResp2 != nil && registerResp2.Success {
		unregisterReq := &pb.UnregisterWebhookRequest{
			WebhookId: registerResp2.WebhookId,
		}

		unregisterResp, err := client.UnregisterWebhook(ctx, unregisterReq)
		if err != nil {
			log.Printf("Failed to unregister webhook: %v", err)
		} else {
			log.Printf("Webhook unregistered successfully:")
			log.Printf("  Success: %t", unregisterResp.Success)
			log.Printf("  Message: %s", unregisterResp.Message)
		}
	}

	log.Println("\n=== All examples completed ===")
}
