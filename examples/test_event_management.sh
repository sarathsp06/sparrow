#!/bin/bash

# Event Management API Testing Script
# This script demonstrates the event registration and management functionality

echo "🚀 Starting Event Management API Test"
echo "======================================"

# Check if server is running
echo "Checking if gRPC server is running on localhost:50051..."
if ! nc -z localhost 50051; then
    echo "❌ gRPC server is not running on localhost:50051"
    echo "Please start the server first with: make grpc-up"
    exit 1
fi

echo "✅ Server is running"
echo ""

# Build and run the event management client
echo "Building event management client..."
go build -o /tmp/event_client examples/event_management_client.go

if [ $? -ne 0 ]; then
    echo "❌ Failed to build event management client"
    exit 1
fi

echo "✅ Client built successfully"
echo ""

echo "Running event management tests..."
echo "================================"
/tmp/event_client

# Clean up
rm -f /tmp/event_client

echo ""
echo "🎉 Event Management API testing completed!"