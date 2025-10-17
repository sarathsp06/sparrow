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
	@echo "Database Migrations:"
	@echo "  make migrate-up     - Run all pending migrations"
	@echo "  make migrate-down   - Rollback last migration"
	@echo "  make migrate-status - Check current migration status"
	@echo "  make migrate-create - Create new migration files"
	@echo ""
	@echo "Docker Development:"
	@echo "  make docker-dev  - Start development environment (PostgreSQL + pgAdmin + River UI)"
	@echo ""
	@echo "gRPC Mode:"
	@echo "  make grpc-up     - Start full gRPC system (queue + gRPC server)"
	@echo "  make grpc-down   - Stop gRPC system"
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
	@DATABASE_URL='postgres://riveruser:riverpass@0.0.0.0:5432/riverqueue?sslmode=disable' ./grpc-server

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


# Build protobuf files (for development)
proto:
	@echo "🔨 Generating protobuf files..."
	@protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/webhook.proto
	@echo "✅ Protobuf files generated"

# Database Migration Commands
migrate-up:
	@echo "⬆️  Running database migrations..."
	@DATABASE_URL=$${DATABASE_URL:-"postgres://riveruser:riverpass@localhost:5432/riverqueue?sslmode=disable"} go run cmd/migrate/main.go -direction=up
	@echo "✅ Migrations completed"

migrate-down:
	@echo "⬇️  Rolling back last migration..."
	@DATABASE_URL=$${DATABASE_URL:-"postgres://riveruser:riverpass@localhost:5432/riverqueue?sslmode=disable"} go run cmd/migrate/main.go -direction=down
	@echo "✅ Rollback completed"

migrate-status:
	@echo "📊 Checking migration status..."
	@DATABASE_URL=$${DATABASE_URL:-"postgres://riveruser:riverpass@localhost:5432/riverqueue?sslmode=disable"} docker-compose -f docker-compose.grpc.yml exec postgres psql -U riveruser -d riverqueue -c "SELECT version, dirty FROM schema_migrations;" 2>/dev/null || echo "❌ Cannot connect to database or schema_migrations table doesn't exist"

migrate-create:
	@echo "📝 Creating new migration files..."
	@read -p "Enter migration name (e.g., add_user_index): " name; \
	if [ -z "$$name" ]; then \
		echo "❌ Migration name cannot be empty"; \
		exit 1; \
	fi; \
	timestamp=$$(date +%s); \
	padded_num=$$(printf "%06d" $$((timestamp % 999999))); \
	echo "-- Add your SQL statements here" > db/migrations/$${padded_num}_$$name.up.sql; \
	echo "-- Add rollback SQL statements here" > db/migrations/$${padded_num}_$$name.down.sql; \
	echo "✅ Created migration files:"; \
	echo "   📄 db/migrations/$${padded_num}_$$name.up.sql"; \
	echo "   📄 db/migrations/$${padded_num}_$$name.down.sql"