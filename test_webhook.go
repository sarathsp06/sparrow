package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/sarathsp06/sparrow/proto"
)

func main() {
	// Connect to the gRPC server
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewWebhookServiceClient(conn)
	ctx := context.Background()

	// Test webhook registration
	registerReq := &pb.RegisterWebhookRequest{
		Namespace: "default",
		Events:    []string{"signup", "login"}, // these events should exist
		Url:       "https://test.example.com/webhook",
		Headers: map[string]string{
			"Authorization": "Bearer test",
		},
		Timeout:     30,
		Active:      true,
		Description: "Test webhook",
	}

	fmt.Printf("Registering webhook with events: %v\n", registerReq.Events)
	registerResp, err := client.RegisterWebhook(ctx, registerReq)
	if err != nil {
		log.Printf("Failed to register webhook: %v", err)
	} else {
		log.Printf("Webhook registered successfully: ID=%s", registerResp.WebhookId)
	}
}
