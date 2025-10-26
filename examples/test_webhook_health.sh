#!/bin/bash

# Webhook Health Management API Testing Script
# This script demonstrates the webhook health monitoring functionality

echo "🏥 Starting Webhook Health Management API Test"
echo "=============================================="

# Check if server is running
echo "Checking if gRPC server is running on localhost:50051..."
if ! nc -z localhost 50051; then
    echo "❌ gRPC server is not running on localhost:50051"
    echo "Please start the server first with: make grpc-up"
    exit 1
fi

echo "✅ Server is running"
echo ""

# Build and run the health client
echo "Building webhook health client..."
go build -o /tmp/health_client examples/webhook_health_client.go

if [ $? -ne 0 ]; then
    echo "❌ Failed to build webhook health client"
    exit 1
fi

echo "✅ Client built successfully"
echo ""

echo "Running webhook health tests..."
echo "==============================="
/tmp/health_client

# Clean up
rm -f /tmp/health_client

echo ""
echo "🎉 Webhook Health Management API testing completed!"