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

	fmt.Println("🚀 Testing Event Management API")
	fmt.Println("===============================")

	// 1. Register a new event
	fmt.Println("\n1. Registering new event...")
	registerResp, err := client.RegisterEvent(ctx, &pb.RegisterEventRequest{
		Name:        "user.created",
		Description: "Triggered when a new user account is created",
		Schema:      `{"type": "object", "properties": {"userId": {"type": "string"}, "email": {"type": "string"}}}`,
		Metadata: map[string]string{
			"category": "user_management",
			"source":   "user_service",
		},
		Active: true,
	})
	if err != nil {
		log.Fatalf("Failed to register event: %v", err)
	}
	fmt.Printf("✅ Event registered: %s (ID: %s)\n", registerResp.Message, registerResp.EventId)

	// 2. Register another event
	fmt.Println("\n2. Registering another event...")
	registerResp2, err := client.RegisterEvent(ctx, &pb.RegisterEventRequest{
		Name:        "order.completed",
		Description: "Triggered when an order is successfully completed",
		Schema:      `{"type": "object", "properties": {"orderId": {"type": "string"}, "amount": {"type": "number"}}}`,
		Metadata: map[string]string{
			"category": "order_management",
			"source":   "order_service",
		},
		Active: true,
	})
	if err != nil {
		log.Fatalf("Failed to register second event: %v", err)
	}
	fmt.Printf("✅ Event registered: %s (ID: %s)\n", registerResp2.Message, registerResp2.EventId)

	// 3. List all events
	fmt.Println("\n3. Listing all events...")
	listResp, err := client.ListEvents(ctx, &pb.ListEventsRequest{
		ActiveOnly: false,
	})
	if err != nil {
		log.Fatalf("Failed to list events: %v", err)
	}
	fmt.Printf("📋 Found %d events:\n", listResp.TotalCount)
	for i, event := range listResp.Events {
		fmt.Printf("   %d. %s - %s (Active: %v)\n", i+1, event.Name, event.Description, event.Active)
		fmt.Printf("      Schema: %s\n", event.Schema)
		if len(event.Metadata) > 0 {
			fmt.Printf("      Metadata: %v\n", event.Metadata)
		}
		fmt.Printf("      Created: %s\n", time.Unix(event.CreatedAt, 0).Format(time.RFC3339))
	}

	// 4. Update an event
	fmt.Println("\n4. Updating event...")
	updateResp, err := client.UpdateEvent(ctx, &pb.UpdateEventRequest{
		Name:        "user.created",
		Description: "Triggered when a new user account is created (updated description)",
		Schema:      `{"type": "object", "properties": {"userId": {"type": "string"}, "email": {"type": "string"}, "timestamp": {"type": "string"}}}`,
		Metadata: map[string]string{
			"category": "user_management",
			"source":   "user_service",
			"version":  "v2",
		},
		Active: true,
	})
	if err != nil {
		log.Fatalf("Failed to update event: %v", err)
	}
	fmt.Printf("✅ Event updated: %s\n", updateResp.Message)

	// 5. List only active events
	fmt.Println("\n5. Listing only active events...")
	activeListResp, err := client.ListEvents(ctx, &pb.ListEventsRequest{
		ActiveOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to list active events: %v", err)
	}
	fmt.Printf("📋 Found %d active events:\n", activeListResp.TotalCount)
	for i, event := range activeListResp.Events {
		fmt.Printf("   %d. %s - %s\n", i+1, event.Name, event.Description)
	}

	// 6. Try to register a duplicate event (should fail)
	fmt.Println("\n6. Testing duplicate event registration...")
	_, err = client.RegisterEvent(ctx, &pb.RegisterEventRequest{
		Name:        "user.created",
		Description: "This should fail",
		Active:      true,
	})
	if err != nil {
		fmt.Printf("❌ Expected error for duplicate event: %v\n", err)
	} else {
		fmt.Println("⚠️  Warning: Duplicate event registration should have failed")
	}

	// 7. Delete an event
	fmt.Println("\n7. Deleting event...")
	deleteResp, err := client.DeleteEvent(ctx, &pb.DeleteEventRequest{
		Name: "order.completed",
	})
	if err != nil {
		log.Fatalf("Failed to delete event: %v", err)
	}
	fmt.Printf("✅ Event deleted: %s\n", deleteResp.Message)

	// 8. Verify deletion by listing events again
	fmt.Println("\n8. Verifying deletion...")
	finalListResp, err := client.ListEvents(ctx, &pb.ListEventsRequest{
		ActiveOnly: false,
	})
	if err != nil {
		log.Fatalf("Failed to list events after deletion: %v", err)
	}
	fmt.Printf("📋 Found %d events after deletion:\n", finalListResp.TotalCount)
	for i, event := range finalListResp.Events {
		fmt.Printf("   %d. %s - %s (Active: %v)\n", i+1, event.Name, event.Description, event.Active)
	}

	fmt.Println("\n🎉 Event Management API testing completed successfully!")
}
