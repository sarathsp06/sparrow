#!/bin/bash

# Start server in background
echo "Starting server..."
DATABASE_URL='postgres://riveruser:riverpass@0.0.0.0:5432/riverqueue?sslmode=disable' go run ./cmd/server &
SERVER_PID=$!

# Wait for server to start
sleep 3

# Run the example
echo "Running example..."
DATABASE_URL='postgres://riveruser:riverpass@localhost:5432/riverqueue?sslmode=disable' go run examples/grpc_client.go

# Kill server
echo "Stopping server..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null