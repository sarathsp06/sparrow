
# Generic build target for any OS/arch

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
OUTPUT ?= build/server-$(GOOS)-$(GOARCH)


build: ## Build the server binary for current OS/arch
	mkdir -p build
	go build -o $(OUTPUT) ./cmd/server

build-ui: ## Build the frontend for embedding in the Go binary
	cd web && npm run build

build-with-ui: build-ui build ## Build frontend + server binary with embedded UI


build-all: build-ui ## Build for common OS/arch combinations
	mkdir -p build
	GOOS=linux GOARCH=amd64 go build -ldflags "-w" -o build/server-linux-amd64 ./cmd/server
	GOOS=linux GOARCH=arm64 go build -ldflags "-w" -o build/server-linux-arm64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build -ldflags "-w" -o build/server-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=arm64 go build -ldflags "-w" -o build/server-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 go build -ldflags "-w" -o build/server-windows-amd64.exe ./cmd/server



docker-purge: ## Stop and remove Docker containers, networks, volumes, and images created by Docker Compose for development
	docker-compose -f docker-compose.dev.yml  down -v

example: ## Run the gRPC client example
	DATABASE_URL='postgres://riveruser:riverpass@localhost:5432/riverqueue?sslmode=disable' go run examples/grpc_client.go

run-web: ## Run the web development server
	cd web &&  yarn dev

test: ## Run tests
	go test -v ./...

run:  ## Run the gRPC server
	SPARROW_SERVE_UI=true DATABASE_URL='postgres://riveruser:riverpass@0.0.0.0:5432/riverqueue?sslmode=disable'  go run ./cmd/server

migrate: ## Run database migrations
	DATABASE_URL='postgres://riveruser:riverpass@0.0.0.0:5432/riverqueue?sslmode=disable' go run ./cmd/migrate


clean: ## Clean up build artifacts and Go module cache
	rm -rf build
	go clean -modcache

generate: ## Generate protobuf code
	buf generate
	go generate ./...
	go run ./cmd/generate-docs

lint: ## Run golangci-lint for linting
	golangci-lint run -v --timeout 15m ./...

docker-dev: ## Run the development environment with Docker Compose
	docker-compose -f docker-compose.dev.yml up -d

fmt: ## Format the code
	goimports -local github.com/sarathsp06/sparrow/  -w .

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build build-ui build-with-ui build-all run test clean generate docker-dev example docker-purge migrate
