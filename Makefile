
# Generic build target for any OS/arch

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
OUTPUT ?= build/server-$(GOOS)-$(GOARCH)
DATABASE_URL ?= 'postgres://riveruser:riverpass@localhost:5432/riverqueue?sslmode=disable'
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
SEMVER  ?= $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0-dev")
LDFLAGS := -X github.com/sarathsp06/sparrow.Version=$(VERSION)


build: ## Build the server binary for current OS/arch
	mkdir -p build
	go build -ldflags "$(LDFLAGS)" -o $(OUTPUT) ./cmd/server

build-ui: ## Build the frontend for embedding in the Go binary
	cd web && VITE_APP_VERSION=$(SEMVER) npm ci && VITE_APP_VERSION=$(SEMVER) npm run build

build-with-ui: build-ui build ## Build frontend + server binary with embedded UI

release-dry-run: build-ui ## Test GoReleaser locally (no publish)
	goreleaser release --snapshot --clean

docker-dev: ## Run the development environment with Docker Compose (builds from source)
	docker compose -f docker-compose.dev.yml up -d --build

docker-purge: ## Stop and remove Docker containers, networks, volumes, and images created by Docker Compose for development
	docker compose -f docker-compose.dev.yml down -v

## -- Helm / Kubernetes --

CHART_DIR := charts/sparrow
HELM_FAKE_ENCRYPTION_KEY := 0000000000000000000000000000000000000000000000000000000000000000
HELM_FAKE_DATABASE_URL := postgresql://user:pass@db:5432/sparrow?sslmode=disable

helm-lint: ## Lint the Helm chart
	helm lint $(CHART_DIR) \
		--set secrets.encryptionKey="$(HELM_FAKE_ENCRYPTION_KEY)" \
		--set secrets.databaseURL="$(HELM_FAKE_DATABASE_URL)"

helm-template: ## Render chart templates locally (dry-run)
	helm template sparrow $(CHART_DIR) \
		--set secrets.encryptionKey="$(HELM_FAKE_ENCRYPTION_KEY)" \
		--set secrets.databaseURL="$(HELM_FAKE_DATABASE_URL)"

helm-template-pg: ## Render chart templates with bundled PostgreSQL enabled
	helm template sparrow $(CHART_DIR) \
		--set postgresql.enabled=true \
		--set secrets.encryptionKey="$(HELM_FAKE_ENCRYPTION_KEY)"

helm-package: ## Package the Helm chart into a .tgz archive
	mkdir -p build
	helm package $(CHART_DIR) -d build

run-web: ## Run the web development server
	cd web && npm run dev

test: ## Run tests
	go test -v ./...

test-integration: ## Run integration tests (requires Docker for testcontainers)
	go test -v -tags integration -timeout 120s ./internal/integration/...

client-python: ## Regenerate the Python client from the committed OpenAPI spec
	rm -rf client/python
	uvx openapi-python-client generate --path api/openapi.yaml --output-path client/python --overwrite

test-e2e: client-python ## Run end-to-end tests (Gauge + Python, requires Docker)
	cd e2e && uv run gauge run specs/

test-e2e-spec: client-python ## Run a single e2e spec (usage: make test-e2e-spec SPEC=00_hello_world)
	cd e2e && uv run gauge run specs/$(SPEC).spec

test-e2e-tag: client-python ## Run e2e tests by tag (usage: make test-e2e-tag TAG=retry)
	cd e2e && uv run gauge run --tags "$(TAG)" specs/

test-e2e-parallel: client-python ## Run e2e tests in parallel
	cd e2e && uv run gauge run --parallel specs/

test-e2e-report: test-e2e ## Run e2e tests and open HTML report
	open e2e/reports/html-report/index.html

test-e2e-setup: ## Install Gauge and Python dependencies (one-time)
	brew install gauge || true
	gauge install python || true
	cd e2e && uv sync

run:  ## Run the server
	SPARROW_SERVE_UI=true DATABASE_URL=$(DATABASE_URL)  go run ./cmd/server

migrate: ## Run database migrations
	DATABASE_URL=$(DATABASE_URL) go run ./cmd/migrate


clean: ## Clean up all build artifacts (Go, web)
	rm -rf build dist
	go clean -modcache
	rm -rf web/build web/node_modules/.vite
	rm -rf internal/ui/dist
	@echo "Clean complete"

generate: ## Export the OpenAPI spec from Go and regenerate client SDKs
	go run ./cmd/openapi-export api
	rm -rf client/python
	uvx openapi-python-client generate --path api/openapi.yaml --output-path client/python --overwrite
	go generate ./...

book: ## Build the Svelte 5 tutorial PDF with Typst
	typst compile --font-path book/fonts book/tutorial.typ book/svelte5-tutorial.pdf

lint: ## Run golangci-lint for linting
	golangci-lint run -v --timeout 15m ./...

fmt: ## Format the code
	goimports -local github.com/sarathsp06/sparrow/  -w .

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build build-ui client-python build-with-ui release-dry-run run test test-integration test-e2e test-e2e-spec test-e2e-tag test-e2e-parallel test-e2e-report test-e2e-setup clean generate docker-dev docker-purge helm-lint helm-template helm-template-pg helm-package migrate lint fmt run-web book help
