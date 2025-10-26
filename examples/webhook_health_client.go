package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/sarathsp06/sparrow/proto"
)

func main() {
	// Connect to the gRPC server
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewWebhookServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("🏥 Testing Webhook Health Management API")
	fmt.Println("========================================")

	// 1. Register a webhook first
	fmt.Println("\n1. Registering a webhook for health testing...")
	registerResp, err := client.RegisterWebhook(ctx, &pb.RegisterWebhookRequest{
		Namespace:   "health-test",
		Events:      []string{"user.created", "user.updated"},
		Url:         "https://httpbin.org/post",
		Headers:     map[string]string{"Content-Type": "application/json"},
		Timeout:     30,
		Active:      true,
		Description: "Test webhook for health monitoring",
	})
	if err != nil {
		log.Fatalf("Failed to register webhook: %v", err)
	}
	webhookID := registerResp.WebhookId
	fmt.Printf("✅ Webhook registered: %s (ID: %s)\n", registerResp.Message, webhookID)

	// 2. Get initial health status
	fmt.Println("\n2. Getting initial webhook health...")
	healthResp, err := client.GetWebhookHealth(ctx, &pb.GetWebhookHealthRequest{
		WebhookId: webhookID,
		Namespace: "health-test",
	})
	if err != nil {
		log.Fatalf("Failed to get webhook health: %v", err)
	}
	fmt.Printf("✅ Initial health status: %s\n", healthResp.Health.String())
	if healthResp.Metrics != nil {
		fmt.Printf("   Total deliveries: %d\n", healthResp.Metrics.TotalDeliveries)
		fmt.Printf("   Success rate: %.2f%%\n", healthResp.Metrics.SuccessRate*100)
	} else {
		fmt.Println("   No metrics available yet (expected for new webhook)")
	}

	// 3. Trigger some events to generate health data
	fmt.Println("\n3. Triggering events to generate health metrics...")
	for i := 0; i < 3; i++ {
		eventResp, err := client.PushEvent(ctx, &pb.PushEventRequest{
			Namespace:  "health-test",
			Event:      "user.created",
			Payload:    fmt.Sprintf(`{"user_id": "user-%d", "email": "user%d@example.com", "timestamp": "%s"}`, i+1, i+1, time.Now().Format(time.RFC3339)),
			TtlSeconds: 3600,
		})
		if err != nil {
			log.Printf("Failed to push event %d: %v", i+1, err)
		} else {
			fmt.Printf("   Event %d pushed: %s\n", i+1, eventResp.Message)
		}
		time.Sleep(1 * time.Second) // Small delay between events
	}

	// Wait a bit for processing
	fmt.Println("\n   Waiting for webhook deliveries to process...")
	time.Sleep(5 * time.Second)

	// 4. Check health after deliveries
	fmt.Println("\n4. Checking health after deliveries...")
	healthResp, err = client.GetWebhookHealth(ctx, &pb.GetWebhookHealthRequest{
		WebhookId: webhookID,
		Namespace: "health-test",
	})
	if err != nil {
		log.Fatalf("Failed to get webhook health: %v", err)
	}
	fmt.Printf("✅ Updated health status: %s\n", healthResp.Health.String())
	if healthResp.Metrics != nil {
		fmt.Printf("   Total deliveries: %d\n", healthResp.Metrics.TotalDeliveries)
		fmt.Printf("   Successful deliveries: %d\n", healthResp.Metrics.SuccessfulDeliveries)
		fmt.Printf("   Failed deliveries: %d\n", healthResp.Metrics.FailedDeliveries)
		fmt.Printf("   Consecutive failures: %d\n", healthResp.Metrics.ConsecutiveFailures)
		fmt.Printf("   Success rate: %.2f%%\n", healthResp.Metrics.SuccessRate*100)
		fmt.Printf("   Average response time: %dms\n", healthResp.Metrics.AvgResponseTime)

		if healthResp.Metrics.LastSuccessAt > 0 {
			fmt.Printf("   Last success: %s\n", time.Unix(healthResp.Metrics.LastSuccessAt, 0).Format(time.RFC3339))
		}
		if healthResp.Metrics.LastFailureAt > 0 {
			fmt.Printf("   Last failure: %s\n", time.Unix(healthResp.Metrics.LastFailureAt, 0).Format(time.RFC3339))
		}
	}

	// 5. Get health summary
	fmt.Println("\n5. Getting overall health summary...")
	summaryResp, err := client.GetHealthSummary(ctx, &pb.GetHealthSummaryRequest{})
	if err != nil {
		log.Fatalf("Failed to get health summary: %v", err)
	}
	fmt.Printf("✅ Health Summary:\n")
	if summaryResp.Summary != nil {
		fmt.Printf("   Healthy webhooks: %d\n", summaryResp.Summary.HealthyCount)
		fmt.Printf("   Degraded webhooks: %d\n", summaryResp.Summary.DegradedCount)
		fmt.Printf("   Unhealthy webhooks: %d\n", summaryResp.Summary.UnhealthyCount)
		fmt.Printf("   Unknown status webhooks: %d\n", summaryResp.Summary.UnknownCount)
		fmt.Printf("   Total webhooks: %d\n", summaryResp.Summary.TotalCount)
	}

	// 6. List webhooks by health status
	fmt.Println("\n6. Listing healthy webhooks...")
	healthyResp, err := client.ListWebhooksByHealth(ctx, &pb.ListWebhooksByHealthRequest{
		Health: pb.WebhookHealth_HEALTH_HEALTHY,
	})
	if err != nil {
		log.Printf("Note: Failed to list healthy webhooks (might be expected): %v", err)
	} else {
		fmt.Printf("✅ Found %d healthy webhooks\n", healthyResp.TotalCount)
		for i, webhook := range healthyResp.Webhooks {
			fmt.Printf("   %d. %s (%s) - %s\n", i+1, webhook.Url, webhook.Health.String(), webhook.Description)
		}
	}

	// 7. List unknown health webhooks
	fmt.Println("\n7. Listing unknown health webhooks...")
	unknownResp, err := client.ListWebhooksByHealth(ctx, &pb.ListWebhooksByHealthRequest{
		Health: pb.WebhookHealth_HEALTH_UNKNOWN,
	})
	if err != nil {
		log.Printf("Failed to list unknown health webhooks: %v", err)
	} else {
		fmt.Printf("✅ Found %d unknown health webhooks\n", unknownResp.TotalCount)
		for i, webhook := range unknownResp.Webhooks {
			fmt.Printf("   %d. %s (%s) - %s\n", i+1, webhook.Url, webhook.Health.String(), webhook.Description)
		}
	}

	// 8. Test with a webhook that will fail (bad URL)
	fmt.Println("\n8. Testing with a failing webhook...")
	failingResp, err := client.RegisterWebhook(ctx, &pb.RegisterWebhookRequest{
		Namespace:   "health-test",
		Events:      []string{"user.deleted"},
		Url:         "https://invalid-webhook-url-that-will-fail.nonexistent/webhook",
		Headers:     map[string]string{"Content-Type": "application/json"},
		Timeout:     5,
		Active:      true,
		Description: "Webhook designed to fail for health testing",
	})
	if err != nil {
		log.Printf("Failed to register failing webhook: %v", err)
	} else {
		failingWebhookID := failingResp.WebhookId
		fmt.Printf("✅ Failing webhook registered: %s\n", failingWebhookID)

		// Trigger an event for the failing webhook
		_, err = client.PushEvent(ctx, &pb.PushEventRequest{
			Namespace:  "health-test",
			Event:      "user.deleted",
			Payload:    `{"user_id": "user-fail", "reason": "account_closed"}`,
			TtlSeconds: 3600,
		})
		if err != nil {
			log.Printf("Failed to push event for failing webhook: %v", err)
		} else {
			fmt.Println("   Event pushed for failing webhook")
		}

		// Wait and check health
		time.Sleep(10 * time.Second)

		failHealthResp, err := client.GetWebhookHealth(ctx, &pb.GetWebhookHealthRequest{
			WebhookId: failingWebhookID,
			Namespace: "health-test",
		})
		if err != nil {
			log.Printf("Failed to get failing webhook health: %v", err)
		} else {
			fmt.Printf("   Failing webhook health: %s\n", failHealthResp.Health.String())
			if failHealthResp.Metrics != nil {
				fmt.Printf("   Failed deliveries: %d\n", failHealthResp.Metrics.FailedDeliveries)
				fmt.Printf("   Success rate: %.2f%%\n", failHealthResp.Metrics.SuccessRate*100)
			}
		}
	}

	fmt.Println("\n🎉 Webhook Health Management API testing completed!")
}
