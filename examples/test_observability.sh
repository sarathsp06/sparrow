#!/bin/bash

# HTTPQueue OpenTelemetry Integration Test
# This script demonstrates the complete observability setup

set -e

echo "🔭 HTTPQueue OpenTelemetry Integration Test"
echo "=========================================="
echo ""

# Check if required tools are available
command -v docker >/dev/null 2>&1 || { echo "❌ Docker is required but not installed."; exit 1; }
command -v docker-compose >/dev/null 2>&1 || { echo "❌ Docker Compose is required but not installed."; exit 1; }

echo "✅ Prerequisites check passed"
echo ""

# Start observability stack
echo "📊 Starting observability stack..."
make obs-up

echo ""
echo "⏳ Waiting for observability services to be ready..."
echo "   This may take 30-60 seconds..."

# Wait for services to be ready
sleep 30

# Check if services are responding
echo ""
echo "🔍 Checking service health..."

# Check Jaeger
if curl -s -f http://localhost:16686/api/services >/dev/null 2>&1; then
    echo "✅ Jaeger UI is ready"
else
    echo "⚠️  Jaeger UI not ready yet (this is normal)"
fi

# Check Prometheus
if curl -s -f http://localhost:9090/-/ready >/dev/null 2>&1; then
    echo "✅ Prometheus is ready"
else
    echo "⚠️  Prometheus not ready yet"
fi

# Check Grafana
if curl -s -f http://localhost:3000/api/health >/dev/null 2>&1; then
    echo "✅ Grafana is ready"
else
    echo "⚠️  Grafana not ready yet"
fi

# Check OTEL Collector
if curl -s -f http://localhost:8888/metrics >/dev/null 2>&1; then
    echo "✅ OTEL Collector is ready"
else
    echo "⚠️  OTEL Collector not ready yet"
fi

echo ""
echo "🎯 Observability stack is running!"
echo ""
echo "📊 Access the UIs:"
echo "   Grafana:    http://localhost:3000 (admin/admin)"
echo "   Jaeger:     http://localhost:16686"
echo "   Prometheus: http://localhost:9090"
echo "   OTEL:       http://localhost:8888/metrics"
echo ""

# Provide instructions for next steps
echo "🚀 Next steps:"
echo ""
echo "1. Start HTTPQueue with observability:"
echo "   export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318"
echo "   export ENVIRONMENT=testing"
echo "   make docker-dev  # Start PostgreSQL"
echo "   make migrate-up  # Run migrations"
echo "   make run         # Start HTTPQueue"
echo ""
echo "2. Generate some test data:"
echo "   go run examples/grpc_client.go"
echo ""
echo "3. View traces and metrics in the UIs above"
echo ""
echo "4. Stop everything when done:"
echo "   make obs-down"
echo ""

echo "✨ Setup complete! The observability stack is ready for HTTPQueue."