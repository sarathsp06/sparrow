
# Generic build target for any OS/arch

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
OUTPUT ?= build/server-$(GOOS)-$(GOARCH)
DATABASE_URL ?= 'postgres://riveruser:riverpass@localhost:5432/riverqueue?sslmode=disable'
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X github.com/sarathsp06/sparrow.Version=$(VERSION)


build: ## Build the server binary for current OS/arch
	mkdir -p build
	go build -ldflags "$(LDFLAGS)" -o $(OUTPUT) ./cmd/server

build-ui: ## Build the frontend for embedding in the Go binary
	cd web && npm ci && npm run build

build-with-ui: build-ui build ## Build frontend + server binary with embedded UI

build-binaries: ## Cross-compile server binaries (assumes UI is already built)
	mkdir -p build
	GOOS=linux GOARCH=amd64 go build -ldflags "-w -s $(LDFLAGS)" -o build/sparrow-linux-amd64 ./cmd/server
	GOOS=linux GOARCH=arm64 go build -ldflags "-w -s $(LDFLAGS)" -o build/sparrow-linux-arm64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build -ldflags "-w -s $(LDFLAGS)" -o build/sparrow-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=arm64 go build -ldflags "-w -s $(LDFLAGS)" -o build/sparrow-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 go build -ldflags "-w -s $(LDFLAGS)" -o build/sparrow-windows-amd64.exe ./cmd/server

build-all: build-ui build-binaries ## Build frontend + all cross-compiled binaries



docker-build: ## Build Docker image locally
	docker build -t ghcr.io/sarathsp06/sparrow:$(VERSION) -t ghcr.io/sarathsp06/sparrow:latest --build-arg VERSION=$(VERSION) .

docker-push: ## Push Docker image to GHCR (requires: docker login ghcr.io)
	docker push ghcr.io/sarathsp06/sparrow:$(VERSION)
	docker push ghcr.io/sarathsp06/sparrow:latest

docker-dev: ## Run the development environment with Docker Compose
	docker-compose -f docker-compose.yml up -d --build

docker-purge: ## Stop and remove Docker containers, networks, volumes, and images created by Docker Compose for development
	docker-compose -f docker-compose.yml down -v

## -- Helm / Kubernetes --

CHART_DIR := charts/sparrow

helm-lint: ## Lint the Helm chart
	helm lint $(CHART_DIR)

helm-template: ## Render chart templates locally (dry-run)
	helm template sparrow $(CHART_DIR) --set secrets.databaseURL="postgresql://user:pass@db:5432/sparrow?sslmode=disable"

helm-template-pg: ## Render chart templates with bundled PostgreSQL enabled
	helm template sparrow $(CHART_DIR) --set postgresql.enabled=true

helm-package: ## Package the Helm chart into a .tgz archive
	mkdir -p build
	helm package $(CHART_DIR) -d build

example: ## Run the gRPC client example
	DATABASE_URL=$(DATABASE_URL) go run examples/grpc_client.go

run-web: ## Run the web development server
	cd web && npm run dev

test: ## Run tests
	go test -v ./...

test-integration: ## Run integration tests (requires Docker for testcontainers)
	go test -v -tags integration -timeout 120s ./internal/integration/...

run:  ## Run the gRPC server
	SPARROW_SERVE_UI=true DATABASE_URL=$(DATABASE_URL)  go run ./cmd/server

migrate: ## Run database migrations
	DATABASE_URL=$(DATABASE_URL) go run ./cmd/migrate


clean: ## Clean up all build artifacts (Go, web, docs)
	rm -rf build
	go clean -modcache
	rm -rf web/build web/node_modules/.vite
	rm -rf docs/dist docs/node_modules/.vite docs/.astro
	rm -rf internal/ui/dist
	mkdir -p internal/ui/dist
	touch	internal/ui/dist/.gitkeep
	@echo "Clean complete"

generate: ## Generate protobuf code and gRPC/ConnectRPC clients
	rm -rf client/go client/js client/python
	buf generate
	go generate ./...

generate-docs: ## Generate API reference docs from proto definitions
	go run ./cmd/gendocs

lint: ## Run golangci-lint for linting
	golangci-lint run -v --timeout 15m ./...

fmt: ## Format the code
	goimports -local github.com/sarathsp06/sparrow/  -w .

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build build-ui build-with-ui build-binaries build-all run test test-integration clean generate generate-docs docker-build docker-push docker-dev docker-purge helm-lint helm-template helm-template-pg helm-package example migrate lint fmt run-web help
