#!/bin/bash

# Quick start script for gRPC Docker environment
echo "🚀 Starting gRPC Webhook Queue Environment..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose is not installed. Please install docker-compose first."
    exit 1
fi

echo "✅ Docker is running"

# Start gRPC environment
echo "🐳 Starting gRPC environment (PostgreSQL + pgAdmin + River UI + gRPC Server)..."
docker-compose -f docker-compose.grpc.yml up -d

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL to be ready..."
max_attempts=30
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if docker-compose -f docker-compose.grpc.yml exec -T httpqueue-postgres pg_isready -U riveruser -d riverqueue > /dev/null 2>&1; then
        echo "✅ PostgreSQL is ready!"
        break
    fi
    attempt=$((attempt + 1))
    sleep 1
done

if [ $attempt -eq $max_attempts ]; then
    echo "❌ PostgreSQL failed to start within 30 seconds"
    exit 1
fi

# Display connection information
echo ""
echo "🎉 gRPC environment is ready!"
echo ""
echo "📊 Database Connection:"
echo "   Host: 0.0.0.0:5432"
echo "   Database: riverqueue"
echo "   Username: riveruser"
echo "   Password: riverpass"
echo "   URL: postgres://riveruser:riverpass@0.0.0.0:5432/riverqueue?sslmode=disable"
echo ""
echo "📡 gRPC Server:"
echo "   Address: localhost:50051"
echo "   Test with: make grpc-test"
echo ""
echo "🌐 pgAdmin (Database Management):"
echo "   URL: http://0.0.0.0:8081"
echo "   Email: admin@example.com"
echo "   Password: admin123"
echo ""
echo "📊 River UI (Job Monitoring):"
echo "   URL: http://0.0.0.0:8080"
echo ""
echo "💻 Run the gRPC server locally:"
echo "   DATABASE_URL='postgres://riveruser:riverpass@0.0.0.0:5432/riverqueue?sslmode=disable' go run cmd/grpc-server/main.go"
echo ""
echo "🧪 Test the gRPC API:"
echo "   go run examples/grpc_client.go"
echo ""
echo "🛑 To stop the environment:"
echo "   make grpc-down"