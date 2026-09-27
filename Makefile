
# Generic build target for any OS/arch

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
OUTPUT ?= build/server-$(GOOS)-$(GOARCH)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
SEMVER  ?= $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0-dev")
LDFLAGS := -s -w -X github.com/sarathsp06/sparrow.Version=$(VERSION)
IMAGE_E2E ?= sparrow:e2e


build: ## Build the server binary for current OS/arch
	mkdir -p build
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUTPUT) ./cmd/server


build-cli: ## Build the sparrow CLI binary for current OS/arch
	mkdir -p build
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o build/sparrow-$(GOOS)-$(GOARCH) ./satellites/sparrow

build-sources: ## Build the sparrow-sources binary for current OS/arch
	mkdir -p build
	go build -trimpath -ldflags "-s -w" -o build/sparrow-sources-$(GOOS)-$(GOARCH) ./satellites/sparrow-sources

build-sinks: ## Build the sparrow-sinks binary for current OS/arch
	mkdir -p build
	go build -trimpath -ldflags "-s -w" -o build/sparrow-sinks-$(GOOS)-$(GOARCH) ./satellites/sparrow-sinks

build-all: build build-cli build-sources build-sinks ## Build server, CLI, sources, and sinks binaries

build-ui: ## Build the frontend for embedding in the Go binary
	cd web && VITE_APP_VERSION=$(SEMVER) npm ci && VITE_APP_VERSION=$(SEMVER) PUBLIC_API_URL=/ npm run build

build-with-ui: build-ui build ## Build frontend + server binary with embedded UI

release-dry-run: build-ui ## Test GoReleaser locally (no publish)
	goreleaser release --snapshot --clean

docker-dev: ## Run the development environment with Docker Compose (builds from source)
	docker compose -f docker-compose.dev.yml up -d --build
	@echo ""
	@echo "✅ Sparrow is deployed:"
	@echo "   Web UI:     http://localhost:8080/"
	@echo "   API docs:   http://localhost:8080/docs"
	@echo "   OpenAPI:    http://localhost:8080/openapi.yaml"
	@echo "   Health:     http://localhost:8080/health"
	@echo "   Logs:       docker compose -f docker-compose.dev.yml logs -f sparrow"

docker-purge: ## Stop and remove Docker containers, networks, volumes, and images created by Docker Compose for development
	docker compose -f docker-compose.dev.yml down -v

run-web: ## Run the web development server
	cd web && npm run dev

web-test: ## Run frontend unit tests
	cd web && npm run test:unit

# The CLI (satellites/sparrow) and pkg/{signature,template} are separate Go
# modules joined by the committed go.work (see docs/adr/0002-cli-module-split.md).
# Root ./... only covers the root module, so the workspace modules are listed
# explicitly; go.work makes the cross-module import paths resolve.
MODULE_TEST_PATHS := ./... \
	github.com/sarathsp06/sparrow/satellites/sparrow/... \
	github.com/sarathsp06/sparrow/satellites/recipes/... \
	github.com/sarathsp06/sparrow/pkg/access/... \
	github.com/sarathsp06/sparrow/pkg/signature/... \
	github.com/sarathsp06/sparrow/pkg/template/...

test: ## Run tests (all modules)
	go test -v $(MODULE_TEST_PATHS)

verify-conformance: ## Run the shared signature vectors against every client/verify helper (Docker for missing toolchains)
	scripts/verify-conformance.sh

test-integration: ## Run integration tests (requires Docker for testcontainers)
	go test -v -tags integration -timeout 120s ./internal/integration/...

client-python: ## Regenerate the Python client from the committed OpenAPI spec
	rm -rf client/python
	uvx openapi-python-client generate --path api/openapi.yaml --output-path client/python --overwrite

docker-build-e2e: ## Build the Docker image the e2e suite runs (tag: $(IMAGE_E2E))
	docker build -t $(IMAGE_E2E) .

test-e2e: client-python docker-build-e2e ## Run end-to-end tests (Gauge + Python, requires Docker)
	cd e2e && uv run gauge run specs/

test-ui: ## Run Playwright UI/portal browser tests (boots Postgres + embedded-UI server; requires Docker)
	./scripts/test-ui.sh

test-e2e-spec: client-python docker-build-e2e ## Run a single e2e spec (usage: make test-e2e-spec SPEC=00_hello_world)
	cd e2e && uv run gauge run specs/$(SPEC).spec

test-e2e-tag: client-python docker-build-e2e ## Run e2e tests by tag (usage: make test-e2e-tag TAG=retry)
	cd e2e && uv run gauge run --tags "$(TAG)" specs/

test-e2e-parallel: client-python docker-build-e2e ## Run e2e tests in parallel
	cd e2e && uv run gauge run --parallel specs/

test-e2e-report: test-e2e ## Run e2e tests and open HTML report
	open e2e/reports/html-report/index.html

test-e2e-setup: ## Install Gauge and Python dependencies (one-time)
	brew install gauge || true
	gauge install python || true
	cd e2e && uv sync

# run/migrate: DATABASE_URL and the encryption keyring come from the shell or
# .env; scripts/dev-env.sh fills dev defaults (the `make dev-db` Postgres and an
# all-zeros keyring) only for what neither sets.
run:  ## Run the server with the embedded UI (dev defaults; Postgres from `make dev-db`)
	SPARROW_SERVE_UI=true ./scripts/dev-env.sh go run ./cmd/server

dev-db: ## Start a throwaway Postgres on localhost:5432 matching the default DATABASE_URL
	docker run -d --name sparrow-dev-pg -p 5432:5432 \
		-e POSTGRES_USER=sparrow -e POSTGRES_PASSWORD=sparrow -e POSTGRES_DB=sparrow \
		postgres:15-alpine

migrate: ## Run database migrations
	./scripts/dev-env.sh go run ./cmd/migrate


clean: ## Clean up all build artifacts (Go, web)
	rm -rf build dist
	go clean -modcache
	rm -rf web/build web/node_modules/.vite
	rm -rf internal/ui/dist
	mkdir -p internal/ui/dist && touch internal/ui/dist/.gitkeep
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

.PHONY: build build-cli build-sources build-sinks build-all build-ui client-python docker-build-e2e build-with-ui release-dry-run run dev-db test test-integration verify-conformance test-ui test-e2e test-e2e-spec test-e2e-tag test-e2e-parallel test-e2e-report test-e2e-setup clean generate docker-dev docker-purge helm-lint helm-template helm-template-pg helm-package migrate lint fmt run-web book help
