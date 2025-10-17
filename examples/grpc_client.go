package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/sarathsp06/httpqueue/proto"
)

func main() {
	// Connect to the gRPC server
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create client
	client := pb.NewWebhookServiceClient(conn)

	// Example 1: Schedule a simple webhook
	fmt.Println("=== Example 1: Simple Webhook ===")
	payload := map[string]interface{}{
		"event":   "user_signup",
		"user_id": 12345,
		"email":   "user@example.com",
	}
	payloadJSON, _ := json.Marshal(payload)

	response, err := client.ScheduleWebhook(context.Background(), &pb.ScheduleWebhookRequest{
		Url:     "https://httpbin.org/post",
		Method:  "POST",
		Payload: string(payloadJSON),
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"X-Event-Type": "user_signup",
		},
		Timeout: 30,
		Queue:   "webhooks",
	})

	if err != nil {
		log.Printf("Failed to schedule webhook: %v", err)
	} else {
		fmt.Printf("Webhook scheduled successfully:\n")
		fmt.Printf("  Job ID: %d\n", response.JobId)
		fmt.Printf("  Success: %t\n", response.Success)
		fmt.Printf("  Message: %s\n", response.Message)
		fmt.Printf("  Scheduled At: %s\n", time.Unix(response.ScheduledAt, 0))
	}

	// Example 2: Schedule a delayed webhook
	fmt.Println("\n=== Example 2: Delayed Webhook ===")
	delayedPayload := map[string]interface{}{
		"event":     "delayed_notification",
		"timestamp": time.Now().Unix(),
		"message":   "This is a delayed webhook",
	}
	delayedPayloadJSON, _ := json.Marshal(delayedPayload)

	futureTime := time.Now().Add(10 * time.Second)
	response2, err := client.ScheduleWebhook(context.Background(), &pb.ScheduleWebhookRequest{
		Url:         "https://httpbin.org/post",
		Method:      "POST",
		Payload:     string(delayedPayloadJSON),
		Timeout:     15,
		Queue:       "webhooks",
		ScheduledAt: futureTime.Unix(),
		Priority:    1,
	})

	if err != nil {
		log.Printf("Failed to schedule delayed webhook: %v", err)
	} else {
		fmt.Printf("Delayed webhook scheduled successfully:\n")
		fmt.Printf("  Job ID: %d\n", response2.JobId)
		fmt.Printf("  Success: %t\n", response2.Success)
		fmt.Printf("  Message: %s\n", response2.Message)
		fmt.Printf("  Scheduled At: %s\n", time.Unix(response2.ScheduledAt, 0))
	}

	// Example 3: Schedule batch webhooks
	fmt.Println("\n=== Example 3: Batch Webhooks ===")
	webhooks := make([]*pb.ScheduleWebhookRequest, 3)
	for i := 0; i < 3; i++ {
		batchPayload := map[string]interface{}{
			"batch_id": i + 1,
			"event":    "batch_webhook",
			"data":     fmt.Sprintf("Batch webhook #%d", i+1),
		}
		batchPayloadJSON, _ := json.Marshal(batchPayload)

		webhooks[i] = &pb.ScheduleWebhookRequest{
			Url:     "https://httpbin.org/post",
			Method:  "POST",
			Payload: string(batchPayloadJSON),
			Headers: map[string]string{
				"X-Batch-ID": fmt.Sprintf("%d", i+1),
			},
			Timeout: 20,
			Queue:   "webhooks",
		}
	}

	batchResponse, err := client.ScheduleWebhookBatch(context.Background(), &pb.ScheduleWebhookBatchRequest{
		Webhooks: webhooks,
	})

	if err != nil {
		log.Printf("Failed to schedule batch webhooks: %v", err)
	} else {
		fmt.Printf("Batch webhooks scheduled:\n")
		fmt.Printf("  Total Scheduled: %d\n", batchResponse.TotalScheduled)
		fmt.Printf("  Total Failed: %d\n", batchResponse.TotalFailed)
		
		for i, result := range batchResponse.Results {
			fmt.Printf("  Webhook %d: Job ID %d, Success: %t\n", i+1, result.JobId, result.Success)
		}
	}

	// Example 4: Get webhook status (placeholder)
	fmt.Println("\n=== Example 4: Get Webhook Status ===")
	if response.Success {
		statusResponse, err := client.GetWebhookStatus(context.Background(), &pb.GetWebhookStatusRequest{
			JobId: response.JobId,
		})

		if err != nil {
			log.Printf("Failed to get webhook status: %v", err)
		} else {
			fmt.Printf("Webhook status:\n")
			fmt.Printf("  Job ID: %d\n", statusResponse.JobId)
			fmt.Printf("  Status: %s\n", statusResponse.Status)
			fmt.Printf("  Message: %s\n", statusResponse.Message)
			fmt.Printf("  Attempt Count: %d/%d\n", statusResponse.AttemptCount, statusResponse.MaxAttempts)
		}
	}

	fmt.Println("\n=== All examples completed ===")
}