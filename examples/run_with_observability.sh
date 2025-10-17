#!/bin/bash

# Example script to demonstrate HTTPQueue with OpenTelemetry
# This script starts the observability stack and HTTPQueue with tracing enabled

echo "🔭 Starting HTTPQueue with OpenTelemetry observability..."

# Start the observability stack
echo "1. Starting observability stack (Jaeger, Prometheus, Grafana)..."
make obs-up

echo ""
echo "⏳ Waiting for observability stack to be ready..."
sleep 10

# Start HTTPQueue with development database
echo ""
echo "2. Starting development database..."
make docker-dev

echo ""
echo "⏳ Waiting for database to be ready..."
sleep 5

# Run migrations
echo ""
echo "3. Running database migrations..."
make migrate-up

# Start HTTPQueue with OpenTelemetry enabled
echo ""
echo "4. Starting HTTPQueue with OpenTelemetry..."
echo "   OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318"
echo "   ENVIRONMENT=development"
echo ""

# Set environment variables for OpenTelemetry
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4318"
export ENVIRONMENT="development"
export DATABASE_URL="postgres://riveruser:riverpass@localhost:5432/riverqueue?sslmode=disable"

# Start the server
echo "🚀 Starting HTTPQueue server..."
echo "   gRPC server will be available at: localhost:50051"
echo "   Traces will be sent to: http://localhost:4318"
echo ""
echo "📊 Access the observability UIs:"
echo "   Grafana:    http://localhost:3000 (admin/admin)"
echo "   Jaeger:     http://localhost:16686" 
echo "   Prometheus: http://localhost:9090"
echo ""
echo "🧪 Run the example client in another terminal:"
echo "   go run examples/grpc_client.go"
echo ""
echo "Press Ctrl+C to stop the server..."

go run main.go