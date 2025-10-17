# Makefile for River Queue example

.PHONY: build run test clean help docker-dev grpc-up grpc-down grpc-logs grpc-test grpc-db-shell grpc-jobs proto

# Default target
help:
	@echo "Available commands:"
	@echo ""
	@echo "Local Development:"
	@echo "  make build       - Build the gRPC server"
	@echo "  make run         - Run the gRPC server locally"
	@echo "  make test        - Run tests"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make proto       - Generate protobuf files"
	@echo ""
	@echo "Docker Development:"
	@echo "  make docker-dev  - Start development environment (PostgreSQL + pgAdmin + River UI)"
	@echo ""
	@echo "gRPC Mode:"
	@echo "  make grpc-up     - Start full gRPC system (queue + gRPC server)"
	@echo "  make grpc-down   - Stop gRPC system"
	@echo "  make grpc-logs   - View gRPC system logs"
	@echo "  make grpc-test   - Test gRPC API with client examples"
	@echo "  make grpc-db-shell - Open PostgreSQL shell for gRPC environment"
	@echo "  make grpc-jobs   - Show recent jobs in gRPC environment"
	@echo "  make proto       - Generate protobuf files (for development)"

# Setup development environment
setup:
	@./setup.sh

# Build the gRPC server
build:
	@echo "🔨 Building gRPC server..."
	@go build -o grpc-server ./cmd/grpc-server
	@echo "✅ Build complete"

# Run the gRPC server
run: build
	@echo "🚀 Starting gRPC server..."
	@./grpc-server

# Run tests
test:
	@echo "🧪 Running tests..."
	@go test -v

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -f grpc-server
	@echo "✅ Clean complete"

# Development mode with auto-restart (requires entr)
dev:
	@echo "🔄 Starting development mode (install 'entr' for auto-restart)..."
	@command -v entr >/dev/null 2>&1 || (echo "❌ 'entr' not found. Install with: brew install entr" && exit 1)
	@find . -name "*.go" | entr -r make run

# Docker commands
docker-dev:
	@echo "🐳 Starting development environment with Docker..."
	@docker-compose -f docker-compose.dev.yml up -d
	@echo "✅ Development environment started!"
	@echo "   PostgreSQL: 0.0.0.0:5432 (riveruser/riverpass)"
	@echo "   pgAdmin: http://0.0.0.0:8081 (admin@example.com/admin123)"
	@echo "   River UI: http://0.0.0.0:8082"





# gRPC Mode Commands
grpc-up:
	@echo "🚀 Starting gRPC system with Docker Compose..."
	@docker-compose -f docker-compose.grpc.yml up -d
	@echo "✅ gRPC system started!"
	@echo ""
	@echo "Services available:"
	@echo "   📡 gRPC Server: localhost:50051"
	@echo "   📊 River UI: http://localhost:8080"
	@echo "   🗄️  pgAdmin: http://localhost:8081 (admin@example.com/admin123)"
	@echo "   🐘 PostgreSQL: localhost:5432 (riveruser/riverpass)"
	@echo ""
	@echo "Test the API with: make grpc-test"

grpc-down:
	@echo "🛑 Stopping gRPC system..."
	@docker-compose -f docker-compose.grpc.yml down
	@echo "✅ gRPC system stopped"

grpc-logs:
	@echo "📋 gRPC system logs (press Ctrl+C to exit):"
	@echo "Choose which service to view:"
	@echo "  [1] All services"
	@echo "  [2] gRPC server only"
	@echo "  [3] Database only"
	@echo "  [4] River UI only"
	@read -p "Enter choice [1-4]: " choice; \
	case $$choice in \
		1) docker-compose -f docker-compose.grpc.yml logs -f ;; \
		2) docker-compose -f docker-compose.grpc.yml logs -f httpqueue-grpc ;; \
		3) docker-compose -f docker-compose.grpc.yml logs -f httpqueue-postgres ;; \
		4) docker-compose -f docker-compose.grpc.yml logs -f httpqueue-riverui ;; \
		*) echo "Invalid choice, showing all logs..."; docker-compose -f docker-compose.grpc.yml logs -f ;; \
	esac

grpc-test:
	@echo "🧪 Testing gRPC API..."
	@echo "Waiting for services to be ready..."
	@sleep 3
	@echo ""
	@echo "Running gRPC client examples:"
	@go run examples/grpc_client.go
	@echo ""
	@echo "✅ gRPC API test completed!"

# gRPC database shell
grpc-db-shell:
	@echo "📊 Opening PostgreSQL shell for gRPC environment..."
	@docker-compose -f docker-compose.grpc.yml exec httpqueue-postgres psql -U riveruser -d riverqueue

# Show jobs in gRPC environment
grpc-jobs:
	@echo "📋 Recent jobs in gRPC environment:"
	@docker-compose -f docker-compose.grpc.yml exec httpqueue-postgres psql -U riveruser -d riverqueue -c "SELECT id, kind, state, created_at, scheduled_at FROM river_job ORDER BY created_at DESC LIMIT 10;" 2>/dev/null || echo "❌ Cannot connect to database"

# Build protobuf files (for development)
proto:
	@echo "🔨 Generating protobuf files..."
	@protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/webhook.proto
	@echo "✅ Protobuf files generated"