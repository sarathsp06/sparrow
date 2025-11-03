build: ## Build the server binary
	go build -o server ./cmd/server

docker-purge: ## Stop and remove Docker containers, networks, volumes, and images created by Docker Compose for development
	docker-compose -f docker-compose.dev.yml  down -v

example: ## Run the gRPC client example
	DATABASE_URL='postgres://riveruser:riverpass@localhost:5432/riverqueue?sslmode=disable' go run examples/grpc_client.go 

run-web: ## Run the web development server
	cd web &&  yarn dev

test: ## Run tests
	go test -v ./...

run:  ## Run the gRPC server
	DATABASE_URL='postgres://riveruser:riverpass@0.0.0.0:5432/riverqueue?sslmode=disable'  go run ./cmd/server

migrate: ## Run database migrations
	DATABASE_URL='postgres://riveruser:riverpass@0.0.0.0:5432/riverqueue?sslmode=disable' go run ./cmd/migrate


clean: ## Clean up build artifacts
	rm -f grpc-server

proto: ## Generate protobuf code
	buf generate

docker-dev: ## Run the development environment with Docker Compose
	docker-compose -f docker-compose.dev.yml up -d

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	
.PHONY: build run test clean proto docker-dev example