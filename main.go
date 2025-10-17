package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	grpcserver "github.com/sarathsp06/httpqueue/internal/grpc"
	"github.com/sarathsp06/httpqueue/internal/jobs"
	"github.com/sarathsp06/httpqueue/internal/queue"
	"github.com/sarathsp06/httpqueue/internal/webhooks"
	pb "github.com/sarathsp06/httpqueue/proto"
)

func main() {
	ctx := context.Background()

	// Database connection URL
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://localhost/riverqueue?sslmode=disable"
		fmt.Println("🔧 Using default database URL. Set DATABASE_URL environment variable for custom connection.")
	}

	// Initialize queue manager
	queueManager, err := queue.NewManager(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Failed to create queue manager: %v", err)
	}
	defer queueManager.Stop(ctx)

	// Start the queue processing
	if err := queueManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start queue manager: %v", err)
	}

	fmt.Println("🚀 River queue started successfully")

	// Get webhook repository from queue manager
	webhookRepo := queueManager.GetWebhookRepo()

	// Initialize gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen on port 50051: %v", err)
	}

	grpcServer := grpc.NewServer()
	webhookServer := grpcserver.NewWebhookServer(queueManager, webhookRepo)
	pb.RegisterWebhookServiceServer(grpcServer, webhookServer)

	fmt.Println("🌐 gRPC server starting on port 50051")

	// Start gRPC server in a goroutine
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Example: Register some test webhooks after startup
	go func() {
		time.Sleep(3 * time.Second)

		fmt.Println("📝 Registering example webhooks...")

		// Register webhook for user events
		userWebhook := &webhooks.WebhookRegistration{
			Namespace:   "user",
			Events:      []string{"signup", "login", "profile_update"},
			URL:         "https://httpbin.org/post",
			Headers:     map[string]string{"X-Event-Type": "user-events"},
			Timeout:     30,
			Active:      true,
			Description: "User activity notifications",
		}

		if err := webhookRepo.RegisterWebhook(ctx, userWebhook); err != nil {
			log.Printf("Failed to register user webhook: %v", err)
		} else {
			fmt.Printf("✅ Registered user webhook: %s\n", userWebhook.ID)
		}

		// Register webhook for order events
		orderWebhook := &webhooks.WebhookRegistration{
			Namespace:   "order",
			Events:      []string{"created", "updated", "cancelled"},
			URL:         "https://httpbin.org/post",
			Headers:     map[string]string{"X-Event-Type": "order-events"},
			Timeout:     30,
			Active:      true,
			Description: "Order lifecycle notifications",
		}

		if err := webhookRepo.RegisterWebhook(ctx, orderWebhook); err != nil {
			log.Printf("Failed to register order webhook: %v", err)
		} else {
			fmt.Printf("✅ Registered order webhook: %s\n", orderWebhook.ID)
		}
	}()

	// Example: Periodically push test events
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for range ticker.C {
			// Push a test user signup event
			testEvent := jobs.EventArgs{
				EventID:    fmt.Sprintf("event_%d", time.Now().Unix()),
				Namespace:  "user",
				Event:      "signup",
				Payload:    `{"user_id": "` + fmt.Sprintf("user_%d", time.Now().Unix()) + `", "email": "test@example.com"}`,
				TTLSeconds: 3600,
				Metadata:   map[string]string{"source": "test"},
				CreatedAt:  time.Now(),
			}

			_, err := queueManager.GetClient().Insert(ctx, testEvent, nil)
			if err != nil {
				log.Printf("Failed to insert test event: %v", err)
			} else {
				fmt.Printf("📨 Pushed test user signup event: %s\n", testEvent.EventID)
			}
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("🎯 HTTP Queue Server is running...")
	fmt.Println("   gRPC server: localhost:50051")
	fmt.Println("   Press Ctrl+C to stop...")
	<-sigChan

	fmt.Println("\n🛑 Shutting down...")
	ticker.Stop()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcServer.GracefulStop()
	queueManager.Stop(shutdownCtx)
	fmt.Println("👋 Shutdown complete")
}
