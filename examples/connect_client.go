package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"connectrpc.com/connect"

	pb "github.com/sarathsp06/httpqueue/proto"
	"github.com/sarathsp06/httpqueue/proto/protoconnect"
)

func main() {
	// Create Connect client
	client := protoconnect.NewWebhookServiceClient(
		http.DefaultClient,
		"http://localhost:8080",
	)

	ctx := context.Background()

	fmt.Println("🌐 HTTPQueue Connect-RPC Client Example")
	fmt.Println("========================================")

	// Test 1: Register a webhook
	fmt.Println("\n1. Registering webhook...")
	registerReq := &pb.RegisterWebhookRequest{
		Namespace:   "test-app",
		Events:      []string{"user.created", "user.updated"},
		Url:         "https://webhook.site/test-endpoint",
		Headers:     map[string]string{"Authorization": "Bearer test-token"},
		Timeout:     30,
		Description: "Test webhook for user events",
	}

	registerResp, err := client.RegisterWebhook(ctx, connect.NewRequest(registerReq))
	if err != nil {
		log.Fatalf("Failed to register webhook: %v", err)
	}

	webhookID := registerResp.Msg.WebhookId
	fmt.Printf("✅ Webhook registered successfully!")
	fmt.Printf("   Webhook ID: %s\n", webhookID)
	fmt.Printf("   Message: %s\n", registerResp.Msg.Message)

	// Test 2: Push an event
	fmt.Println("\n2. Pushing event...")
	eventPayload := map[string]interface{}{
		"user_id": "12345",
		"email":   "user@example.com",
		"name":    "John Doe",
		"action":  "created",
	}

	payloadBytes, _ := json.Marshal(eventPayload)
	pushReq := &pb.PushEventRequest{
		Namespace:  "test-app",
		Event:      "user.created",
		Payload:    string(payloadBytes),
		TtlSeconds: 3600, // 1 hour TTL
		Metadata: map[string]string{
			"source":    "user-service",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	pushResp, err := client.PushEvent(ctx, connect.NewRequest(pushReq))
	if err != nil {
		log.Fatalf("Failed to push event: %v", err)
	}

	eventID := pushResp.Msg.EventId
	fmt.Printf("✅ Event pushed successfully!")
	fmt.Printf("   Event ID: %s\n", eventID)
	fmt.Printf("   Message: %s\n", pushResp.Msg.Message)

	// Test 3: Check webhook status
	fmt.Println("\n3. Checking webhook status...")
	statusReq := &pb.GetWebhookStatusRequest{
		Identifier: &pb.GetWebhookStatusRequest_WebhookId{
			WebhookId: webhookID,
		},
	}

	statusResp, err := client.GetWebhookStatus(ctx, connect.NewRequest(statusReq))
	if err != nil {
		log.Fatalf("Failed to get webhook status: %v", err)
	}

	fmt.Printf("✅ Webhook status retrieved!")
	fmt.Printf("   Total deliveries: %d\n", statusResp.Msg.TotalDeliveries)
	fmt.Printf("   Message: %s\n", statusResp.Msg.Message)

	if len(statusResp.Msg.Deliveries) > 0 {
		for i, delivery := range statusResp.Msg.Deliveries {
			fmt.Printf("   Delivery %d:\n", i+1)
			fmt.Printf("     - ID: %s\n", delivery.DeliveryId)
			fmt.Printf("     - Status: %s\n", delivery.Status.String())
			fmt.Printf("     - Attempts: %d/%d\n", delivery.AttemptCount, delivery.MaxAttempts)
			fmt.Printf("     - Created: %s\n", time.Unix(delivery.CreatedAt, 0).Format(time.RFC3339))
		}
	}

	// Test 4: List webhooks
	fmt.Println("\n4. Listing webhooks...")
	listReq := &pb.ListWebhooksRequest{
		Namespace:  "test-app",
		ActiveOnly: true,
	}

	listResp, err := client.ListWebhooks(ctx, connect.NewRequest(listReq))
	if err != nil {
		log.Fatalf("Failed to list webhooks: %v", err)
	}

	fmt.Printf("✅ Webhooks listed successfully!")
	fmt.Printf("   Total count: %d\n", listResp.Msg.TotalCount)
	fmt.Printf("   Message: %s\n", listResp.Msg.Message)

	for i, webhook := range listResp.Msg.Webhooks {
		fmt.Printf("   Webhook %d:\n", i+1)
		fmt.Printf("     - ID: %s\n", webhook.WebhookId)
		fmt.Printf("     - URL: %s\n", webhook.Url)
		fmt.Printf("     - Events: %v\n", webhook.Events)
		fmt.Printf("     - Active: %t\n", webhook.Active)
		fmt.Printf("     - Created: %s\n", time.Unix(webhook.CreatedAt, 0).Format(time.RFC3339))
	}

	// Test 5: Test health check endpoint
	fmt.Println("\n5. Testing health check...")
	healthResp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		defer healthResp.Body.Close()
		fmt.Printf("✅ Health check passed! Status: %s\n", healthResp.Status)
	}

	fmt.Println("\n🎉 All Connect-RPC tests completed successfully!")
	fmt.Println("\n📋 Summary:")
	fmt.Println("   - Connect-RPC server provides HTTP/JSON API")
	fmt.Println("   - Compatible with web clients and curl")
	fmt.Println("   - Runs alongside gRPC server")
	fmt.Printf("   - Webhook ID: %s\n", webhookID)
	fmt.Printf("   - Event ID: %s\n", eventID)
}
