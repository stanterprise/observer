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
MIGRATE_BIN := $(BIN_DIR)/migrate

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

.PHONY: all help build build-all build-ingestion build-processor build-api build-migrate run run-dev run-dev-split env-print test test-race test-cover cover-report test-nats-integration fmt vet tidy generate lint proto tools clean clean-cache mongodb-up mongodb-down mongodb-logs mongodb-shell mongodb-reset nats-up nats-down nats-logs docker-build docker-build-all docker-build-aio docker-build-ingestion docker-build-processor docker-build-api docker-up-aio docker-up-dist docker-down docker-web-dev-down web-dev-mode helm-deps helm-lint helm-template helm-template-aio helm-template-prod helm-dry-run helm-dry-run-aio helm-dry-run-prod helm-test helm-validate FORCE

.DEFAULT_GOAL := help

all: build-all ## Build all components

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"}; /^[a-zA-Z0-9_.-]+:.*##/ { printf "\033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

$(INGESTION_BIN): FORCE ## Build the ingestion service binary
	@mkdir -p $(BIN_DIR)
	go build -o $(INGESTION_BIN) ./cmd/ingestion

$(PROCESSOR_BIN): FORCE ## Build the processor service binary
	@mkdir -p $(BIN_DIR)
	go build -o $(PROCESSOR_BIN) ./cmd/processor

$(API_BIN): FORCE ## Build the api service binary
	@mkdir -p $(BIN_DIR)
	go build -o $(API_BIN) ./cmd/api

$(MIGRATE_BIN): FORCE ## Build the postgres migration binary
	@mkdir -p $(BIN_DIR)
	go build -o $(MIGRATE_BIN) ./cmd/migrate

FORCE:

build-ingestion: $(INGESTION_BIN) ## Build ingestion service

build-processor: $(PROCESSOR_BIN) ## Build processor service

build-api: $(API_BIN) ## Build api service

build-migrate: $(MIGRATE_BIN) ## Build migration service

build-all: $(INGESTION_BIN) $(PROCESSOR_BIN) $(API_BIN) $(MIGRATE_BIN) ## Build all components

run: build ## Run the server using the built binary
	./$(APP_BIN)

# Defaults align with docker-compose.yml and .env.example
MONGO_USER ?= root
MONGO_PASSWORD ?= password
MONGO_DATABASE ?= observer
MONGO_PORT ?= 27017

# Construct a MongoDB URI that talks to the Compose MongoDB port on localhost
MONGODB_URI := mongodb://$(MONGO_USER):$(MONGO_PASSWORD)@localhost:$(MONGO_PORT)/$(MONGO_DATABASE)?authSource=admin

# Run the API with MONGODB_URI
run-dev: build ## Run server with MONGODB_URI env
	MONGODB_URI='$(MONGODB_URI)' ./$(APP_BIN)

# Run the API using split MONGO_* environment variables (ConnectMongoDBFromEnv also supports these)
run-dev-split: build ## Run server with individual MONGO_* env vars
	MONGO_HOST=localhost MONGO_PORT=$(MONGO_PORT) MONGO_USER=$(MONGO_USER) MONGO_PASSWORD=$(MONGO_PASSWORD) MONGO_DATABASE=$(MONGO_DATABASE) MONGO_AUTH_SOURCE=admin ./$(APP_BIN)

# Print resolved environment values for verification
env-print: ## Print effective DB-related environment
	@echo "MONGO_USER=$(MONGO_USER)"
	@echo "MONGO_PASSWORD=$(MONGO_PASSWORD)"
	@echo "MONGO_DATABASE=$(MONGO_DATABASE)"
	@echo "MONGO_PORT=$(MONGO_PORT)"
	@echo "MONGODB_URI=$(MONGODB_URI)"

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

# Docker MongoDB helpers
mongodb-up: ## Start MongoDB container
	docker compose up -d mongodb

mongodb-down: ## Stop containers and remove volumes
	docker compose down -v

mongodb-logs: ## Tail MongoDB logs
	docker compose logs -f mongodb

# Open mongosh in the container
mongodb-shell: ## Open mongosh against the mongodb container
	docker compose exec mongodb mongosh -u $${MONGO_USER:-root} -p $${MONGO_PASSWORD:-password} --authenticationDatabase admin $${MONGO_DATABASE:-observer}

mongodb-reset: ## Reset database by recreating container and volume
	docker compose down -v && docker compose up -d mongodb

# MongoDB helpers
MONGO_USER ?= root
MONGO_PASSWORD ?= password
MONGO_DATABASE ?= observer
MONGO_PORT ?= 27017
MONGODB_URI := mongodb://$(MONGO_USER):$(MONGO_PASSWORD)@localhost:$(MONGO_PORT)/$(MONGO_DATABASE)?authSource=admin

mongo-up: ## Start MongoDB container
	docker compose up -d mongodb

mongo-down: ## Stop MongoDB container
	docker compose stop mongodb

mongo-logs: ## Tail MongoDB logs
	docker compose logs -f mongodb

mongo-shell: ## Open mongosh against the MongoDB container
	docker compose exec mongodb mongosh --username $(MONGO_USER) --password $(MONGO_PASSWORD) --authenticationDatabase admin $(MONGO_DATABASE)

mongo-reset: ## Reset MongoDB by recreating container and volume
	docker compose down mongodb -v && docker compose up -d mongodb

mongo-clear: ## Clear all data from MongoDB (drops database)
	docker compose exec mongodb mongosh --username $(MONGO_USER) --password $(MONGO_PASSWORD) --authenticationDatabase admin --eval "db.getSiblingDB('$(MONGO_DATABASE)').dropDatabase()"

mongo-env-print: ## Print effective MongoDB-related environment
	@echo "MONGO_USER=$(MONGO_USER)"
	@echo "MONGO_PASSWORD=$(MONGO_PASSWORD)"
	@echo "MONGO_DATABASE=$(MONGO_DATABASE)"
	@echo "MONGO_PORT=$(MONGO_PORT)"
	@echo "MONGODB_URI=$(MONGODB_URI)"

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

test-mongo-integration: ## Run MongoDB integration tests (requires MongoDB running)
	MONGODB_TEST_URI='$(MONGODB_URI)' go test ./tests/... -v -run TestMongoDB

test-all-integration: ## Run all integration tests (requires NATS and MongoDB running)
	NATS_TEST_URL=nats://localhost:4222 MONGODB_TEST_URI='$(MONGODB_URI)' go test ./tests/... -v

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

docker-build-all: ## Build all Docker images
	docker build -f Dockerfile.aio -t $(IMAGE_NAME):aio .
	docker build -f Dockerfile.ingestion -t $(IMAGE_NAME):ingestion .
	docker build -f Dockerfile.processor -t $(IMAGE_NAME):processor .
	docker build -f Dockerfile.api -t $(IMAGE_NAME):api .
	docker build -f Dockerfile.web -t $(IMAGE_NAME):web .

docker-build-aio: ## Build AIO Docker image
	docker build -f Dockerfile.aio -t $(IMAGE_NAME):aio .

docker-buildx-aio: ## Build multi-platform AIO Docker image with BuildKit (optimized)
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--cache-from type=local,src=/tmp/.buildx-cache \
		--cache-to type=local,dest=/tmp/.buildx-cache-new \
		-f Dockerfile.aio \
		-t $(IMAGE_NAME):aio \
		--load \
		. && \
	rm -rf /tmp/.buildx-cache && \
	mv /tmp/.buildx-cache-new /tmp/.buildx-cache

docker-buildx-setup: ## Setup buildx multi-platform builder (one-time setup)
	docker buildx create --name multiarch --driver docker-container --use || true
	docker buildx inspect --bootstrap

docker-buildx-clean: ## Clean buildx cache
	rm -rf /tmp/.buildx-cache
	docker buildx prune -af

docker-build-ingestion: ## Build ingestion Docker image
	docker build -f Dockerfile.ingestion -t $(IMAGE_NAME):ingestion .

docker-build-processor: ## Build processor Docker image
	docker build -f Dockerfile.processor -t $(IMAGE_NAME):processor .

docker-build-api: ## Build API Docker image
	docker build -f Dockerfile.api -t $(IMAGE_NAME):api .

docker-build-web: ## Build Web UI Docker image
	docker build -f Dockerfile.web -t $(IMAGE_NAME):web .


# Backward compatibility
docker-build: docker-build-all ## Build all Docker images (alias)

docker-clean-images: ## Remove all built Docker images
	docker rmi -f $(IMAGE_NAME):aio $(IMAGE_NAME):ingestion $(IMAGE_NAME):processor $(IMAGE_NAME):api $(IMAGE_NAME):web 2>/dev/null || true

# Docker Compose helpers
docker-up-aio: docker-build-aio ## Start AIO profile with docker compose
	docker compose --profile aio up -d

docker-dev-web: ## Start all containers except for Web UI in dev mode
	docker compose --profile web-dev up -d
	@echo "Web development services started. MongoDB and API are ready."

docker-dev-mongo: ## Start all containers with MongoDB backend (except Web UI)
	docker compose --profile mongo up -d
	@echo "Waiting for MongoDB to be ready..."
	@sleep 3
	@echo "MongoDB is ready for connections"

docker-up-dist: docker-build-ingestion docker-build-processor docker-build-api docker-build-web ## Start distributed profile with docker compose
	docker compose --profile dist up -d

docker-down: ## Stop all docker compose services
	docker compose down

docker-web-dev-down: ## Stop web development profile services
	docker compose --profile web-dev down

web-dev-mode: docker-web-dev-down docker-clean-images docker-build-all docker-dev-web web-dev ## Rebuild and restart all services in Docker and start web UI in development mode

# Helm chart management
helm-deps: ## Update Helm chart dependencies
	helm dependency update charts/observer/

helm-lint: ## Lint the Helm chart
	helm lint charts/observer/

helm-template: ## Render Helm templates with default values
	helm template observer charts/observer/

helm-template-aio: ## Render Helm templates with AIO mode
	helm template observer charts/observer/ --values charts/observer/values-aio.yaml

helm-template-prod: ## Render Helm templates with production values
	helm template observer charts/observer/ --values charts/observer/values-production.yaml

helm-dry-run: ## Dry-run Helm install with default values
	helm install observer-test charts/observer/ --dry-run --debug

helm-dry-run-aio: ## Dry-run Helm install with AIO mode
	helm install observer-test charts/observer/ --values charts/observer/values-aio.yaml --dry-run --debug

helm-dry-run-prod: ## Dry-run Helm install with production values
	helm install observer-test charts/observer/ --values charts/observer/values-production.yaml --dry-run --debug

helm-test: ## Run comprehensive Helm chart tests
	./scripts/test-helm-chart.sh

helm-validate: helm-deps helm-lint helm-test ## Run all Helm validation steps
