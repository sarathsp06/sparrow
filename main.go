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
	"github.com/sarathsp06/httpqueue/internal/queue"
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

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("🎯 HTTP Queue Server is running...")
	fmt.Println("   gRPC server: localhost:50051")
	fmt.Println("   Press Ctrl+C to stop...")
	<-sigChan

	fmt.Println("\n🛑 Shutting down...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcServer.GracefulStop()
	queueManager.Stop(shutdownCtx)
	fmt.Println("👋 Shutdown complete")
}
