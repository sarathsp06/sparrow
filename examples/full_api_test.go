package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/sarathsp06/sparrow/proto"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewWebhookServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	testRegisterEventWithSchema(client, ctx)
	testRegisterWebhookForEvent(client, ctx)
	testPushEventValidPayload(client, ctx)
	testPushEventInvalidPayload(client, ctx)
}

func testRegisterEventWithSchema(client pb.WebhookServiceClient, ctx context.Context) {
	fmt.Println("\n--- Register Event with JSON Schema ---")
	eventSchema := `{"type": "object", "properties": {"userId": {"type": "string"}, "email": {"type": "string"}}, "required": ["userId", "email"]}`
	regEventReq := &pb.RegisterEventRequest{
		Name:        "user.created",
		Description: "Triggered when a new user is created",
		Schema:      eventSchema,
		Metadata:    map[string]string{"category": "user"},
		Active:      true,
	}
	regEventResp, err := client.RegisterEvent(ctx, regEventReq)
	if err != nil {
		log.Fatalf("Failed to register event: %v", err)
	} else {
		fmt.Printf("Event registered: %s (ID: %s)\n", regEventResp.Message, regEventResp.EventId)
	}
}

func testRegisterWebhookForEvent(client pb.WebhookServiceClient, ctx context.Context) {
	fmt.Println("\n--- Register Webhook for Registered Event ---")
	regWebhookReq := &pb.RegisterWebhookRequest{
		Namespace:   "default",
		Events:      []string{"user.created"},
		Url:         "https://webhooks.sarathsadasivan.com/test-user-created",
		Headers:     map[string]string{"X-Test": "true"},
		Timeout:     10,
		Active:      true,
		Description: "Webhook for user.created event",
	}
	regWebhookResp, err := client.RegisterWebhook(ctx, regWebhookReq)
	if err != nil {
		log.Fatalf("Failed to register webhook: %v", err)
	} else {
		fmt.Printf("Webhook registered: %s (ID: %s)\n", regWebhookResp.Message, regWebhookResp.WebhookId)
	}
}

func testPushEventValidPayload(client pb.WebhookServiceClient, ctx context.Context) {
	fmt.Println("\n--- Push user.created Event with Valid Payload ---")
	validPayload := map[string]interface{}{"userId": "user_001", "email": "user@example.com"}
	validPayloadJSON, _ := json.Marshal(validPayload)
	pushValidReq := &pb.PushEventRequest{
		Namespace:  "default",
		Event:      "user.created",
		Payload:    string(validPayloadJSON),
		TtlSeconds: 600,
	}
	pushValidResp, err := client.PushEvent(ctx, pushValidReq)
	if err != nil {
		fmt.Printf("Failed to push valid event: %v\n", err)
	} else {
		fmt.Printf("Valid event pushed: %s\n", pushValidResp.Message)
	}
}

func testPushEventInvalidPayload(client pb.WebhookServiceClient, ctx context.Context) {
	fmt.Println("\n--- Push user.created Event with Invalid Payload ---")
	invalidPayload := map[string]interface{}{"userId": "user_002"} // missing email
	invalidPayloadJSON, _ := json.Marshal(invalidPayload)
	pushInvalidReq := &pb.PushEventRequest{
		Namespace:  "default",
		Event:      "user.created",
		Payload:    string(invalidPayloadJSON),
		TtlSeconds: 600,
	}
	pushInvalidResp, err := client.PushEvent(ctx, pushInvalidReq)
	if err != nil {
		fmt.Printf("Expected failure for invalid event: %v\n", err)
	} else {
		fmt.Printf("Unexpected success for invalid event: %s\n", pushInvalidResp.Message)
	}
}
