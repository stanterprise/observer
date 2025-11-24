GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
	GOBIN := $(shell go env GOPATH)/bin
endif

# Binaries and locations
BIN_DIR := bin
APP_BIN := $(BIN_DIR)/observer
INGESTION_BIN := $(BIN_DIR)/ingestion
PROCESSOR_BIN := $(BIN_DIR)/processor
API_BIN := $(BIN_DIR)/api

# Tooling
PROTOC_GEN_GO := $(GOBIN)/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(GOBIN)/protoc-gen-go-grpc
GOLANGCI_LINT := $(GOBIN)/golangci-lint
PROTOC ?= protoc
PROTO_DIR ?= proto

# Pinned tool versions (adjust as needed)
PROTOC_GEN_GO_VERSION ?= v1.36.6
PROTOC_GEN_GO_GRPC_VERSION ?= v1.5.1
GOLANGCI_LINT_VERSION ?= v1.60.3

.PHONY: all help build build-all build-ingestion build-processor build-api run run-dev run-dev-split env-print test test-race test-cover cover-report test-nats-integration fmt vet tidy generate lint proto tools clean clean-cache db-up db-down db-logs db-psql db-reset nats-up nats-down nats-logs docker-build docker-build-all docker-build-aio docker-build-ingestion docker-build-processor docker-build-api docker-up-aio docker-up-dist docker-down

.DEFAULT_GOAL := help

all: build-all ## Build all components

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"}; /^[a-zA-Z0-9_.-]+:.*##/ { printf "\033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

$(APP_BIN): ## Build the server binary (legacy)
	@mkdir -p $(BIN_DIR)
	go build -o $(APP_BIN) ./server

$(INGESTION_BIN): ## Build the ingestion service binary
	@mkdir -p $(BIN_DIR)
	go build -o $(INGESTION_BIN) ./cmd/ingestion

$(PROCESSOR_BIN): ## Build the processor service binary
	@mkdir -p $(BIN_DIR)
	go build -o $(PROCESSOR_BIN) ./cmd/processor

$(API_BIN): ## Build the api service binary
	@mkdir -p $(BIN_DIR)
	go build -o $(API_BIN) ./cmd/api

build: $(APP_BIN) ## Build legacy server (shortcut)

build-ingestion: $(INGESTION_BIN) ## Build ingestion service

build-processor: $(PROCESSOR_BIN) ## Build processor service

build-api: $(API_BIN) ## Build api service

build-all: $(APP_BIN) $(INGESTION_BIN) $(PROCESSOR_BIN) $(API_BIN) ## Build all components

run: build ## Run the server using the built binary
	./$(APP_BIN)

# Defaults align with docker-compose.yml and .env.example
POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= postgres
POSTGRES_DB ?= observer
POSTGRES_PORT ?= 5432
APPLY_MIGRATIONS ?= 1

# Construct a host DSN that talks to the Compose Postgres port on localhost
DATABASE_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

# Run the API with DATABASE_URL and optional automigrate
run-dev: build ## Run server with DATABASE_URL env
	DATABASE_URL='$(DATABASE_URL)' APPLY_MIGRATIONS=$(APPLY_MIGRATIONS) ./$(APP_BIN)

# Run the API using split PG* environment variables (ConnectFromEnv also supports these)
run-dev-split: build ## Run server with individual PG* env vars
	PGHOST=localhost PGPORT=$(POSTGRES_PORT) PGUSER=$(POSTGRES_USER) PGPASSWORD=$(POSTGRES_PASSWORD) PGDATABASE=$(POSTGRES_DB) PGSSLMODE=disable APPLY_MIGRATIONS=$(APPLY_MIGRATIONS) ./$(APP_BIN)

# Print resolved environment values for verification
env-print: ## Print effective DB-related environment
	@echo "POSTGRES_USER=$(POSTGRES_USER)"
	@echo "POSTGRES_PASSWORD=$(POSTGRES_PASSWORD)"
	@echo "POSTGRES_DB=$(POSTGRES_DB)"
	@echo "POSTGRES_PORT=$(POSTGRES_PORT)"
	@echo "APPLY_MIGRATIONS=$(APPLY_MIGRATIONS)"
	@echo "DATABASE_URL=$(DATABASE_URL)"

test: ## Run unit tests
	go test ./...

test-race: ## Run tests with race detector
	go test -race ./...

test-cover: ## Run tests with coverage and write coverage.out
	go test -coverprofile=coverage.out ./...

cover-report: ## Open an HTML coverage report (requires test-cover first)
	@([ -f coverage.out ] && go tool cover -html=coverage.out || (echo "coverage.out not found; run 'make test-cover' first" && false))

fmt: ## Format code
	go fmt ./...

vet: ## Vet code
	go vet ./...

tidy: ## Tidy go.mod/go.sum
	go mod tidy

generate: ## Run go generate
	go generate ./...

lint: $(GOLANGCI_LINT) ## Run golangci-lint
	$(GOLANGCI_LINT) run ./...

proto: $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) ## Generate protobuf stubs (if proto/*.proto exists)
	@FILES=$$(find "$(PROTO_DIR)" -name '*.proto' 2>/dev/null || true); \
	if [ -z "$$FILES" ]; then \
	  echo "No .proto files found in $(PROTO_DIR); skipping."; \
	else \
	  if ! command -v $(PROTOC) >/dev/null 2>&1; then \
	    echo "Error: protoc not found. Please install Protocol Buffers compiler."; exit 1; \
	  fi; \
	  echo "Generating protobuf stubs for:" $$FILES; \
	  $(PROTOC) \
	    --go_out=. \
	    --go_opt=paths=source_relative \
	    --go-grpc_out=. \
	    --go-grpc_opt=paths=source_relative \
	    $$FILES; \
	fi

$(PROTOC_GEN_GO): ## Install protoc-gen-go (pinned)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)

$(PROTOC_GEN_GO_GRPC): ## Install protoc-gen-go-grpc (pinned)
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)

$(GOLANGCI_LINT): ## Install golangci-lint (pinned)
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

tools: $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) $(GOLANGCI_LINT) ## Install dev tools

clean: ## Remove build and coverage artifacts
	rm -rf "$(BIN_DIR)" coverage.out

clean-cache: ## Clean Go build and test caches
	go clean -cache -testcache

# Docker Postgres helpers
db-up: ## Start Postgres container
	docker compose up -d db

db-down: ## Stop containers and remove volumes
	docker compose down -v

db-logs: ## Tail Postgres logs
	docker compose logs -f db

# Open psql in the container
db-psql: ## Open psql against the db container
	docker compose exec -e PGPASSWORD=$${POSTGRES_PASSWORD:-postgres} db psql -U $${POSTGRES_USER:-postgres} -d $${POSTGRES_DB:-observer}

db-reset: ## Reset database by recreating container and volume
	docker compose down -v && docker compose up -d db

db-clear: ## Clear all data from the database (drops and recreates tables)
	docker compose exec -e PGPASSWORD=$${POSTGRES_PASSWORD:-postgres} db psql -U $${POSTGRES_USER:-postgres} -d $${POSTGRES_DB:-observer} -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO $${POSTGRES_USER:-postgres}; GRANT ALL ON SCHEMA public TO public;"

db-migrate: ## Run database migrations
	@APPLY_MIGRATIONS=1 DATABASE_URL='$(DATABASE_URL)' go run ./scripts/migrate.go

# NATS helpers
nats-up: ## Start NATS container
	docker compose up -d nats

nats-down: ## Stop NATS container
	docker compose stop nats

nats-logs: ## Tail NATS logs
	docker compose logs -f nats

# Integration tests
test-nats-integration: ## Run NATS integration tests (requires NATS running)
	NATS_TEST_URL=nats://localhost:4222 go test ./tests/... -v -run TestNATSIntegration

# Web UI targets
web-install: ## Install Web UI dependencies
	cd web && npm install

web-dev: ## Run Web UI in development mode
	cd web && npm run dev

web-build: ## Build Web UI for production
	cd web && npm run build

web-clean: ## Clean Web UI build artifacts
	rm -rf web/dist web/node_modules

# Docker image management
IMAGE_NAME ?= observer
IMAGE_TAG ?= latest

docker-build-all: build-all ## Build all Docker images
	docker build -f Dockerfile.aio -t $(IMAGE_NAME):aio .
	docker build -f Dockerfile.ingestion -t $(IMAGE_NAME):ingestion .
	docker build -f Dockerfile.processor -t $(IMAGE_NAME):processor .
	docker build -f Dockerfile.api -t $(IMAGE_NAME):api .
	docker build -f Dockerfile.web -t $(IMAGE_NAME):web .

docker-build-aio: build-all ## Build AIO Docker image
	docker build -f Dockerfile.aio -t $(IMAGE_NAME):aio .

docker-build-ingestion: build-all ## Build ingestion Docker image
	docker build -f Dockerfile.ingestion -t $(IMAGE_NAME):ingestion .

docker-build-processor: build-all ## Build processor Docker image
	docker build -f Dockerfile.processor -t $(IMAGE_NAME):processor .

docker-build-api: build-all ## Build API Docker image
	docker build -f Dockerfile.api -t $(IMAGE_NAME):api .

docker-build-web: ## Build Web UI Docker image
	docker build -f Dockerfile.web -t $(IMAGE_NAME):web .


# Backward compatibility
docker-build: docker-build-all ## Build all Docker images (alias)

# Docker Compose helpers
docker-up-aio: docker-build-aio ## Start AIO profile with docker compose
	docker compose --profile aio up -d

docker-dev-web: ## Start all containers except for Web UI in dev mode
	docker compose --profile web-dev up -d
	@echo "Waiting for database to be ready..."
	@sleep 2
	@echo "Running database migrations..."
	@$(MAKE) db-migrate

docker-up-dist: docker-build-ingestion docker-build-processor docker-build-api docker-build-web ## Start distributed profile with docker compose
	docker compose --profile dist up -d

docker-down: ## Stop all docker compose services
	docker compose down
