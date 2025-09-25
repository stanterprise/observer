GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
  GOBIN := $(shell go env GOPATH)/bin
endif

PROTOC_GEN_GO := $(GOBIN)/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(GOBIN)/protoc-gen-go-grpc

.PHONY: all build run run-dev run-dev-split env-print test lint proto tools clean
 .PHONY: db-up db-down db-logs db-psql

all: build

build:
	go build ./...

run: build
	go run ./server

# Defaults align with docker-compose.yml and .env.example
POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= postgres
POSTGRES_DB ?= observer
POSTGRES_PORT ?= 5432
APPLY_MIGRATIONS ?= 1

# Construct a host DSN that talks to the Compose Postgres port on localhost
DATABASE_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

# Run the API with DATABASE_URL and optional automigrate
run-dev:
	DATABASE_URL='$(DATABASE_URL)' APPLY_MIGRATIONS=$(APPLY_MIGRATIONS) go run ./server

# Run the API using split PG* environment variables (ConnectFromEnv also supports these)
run-dev-split:
	PGHOST=localhost PGPORT=$(POSTGRES_PORT) PGUSER=$(POSTGRES_USER) PGPASSWORD=$(POSTGRES_PASSWORD) PGDATABASE=$(POSTGRES_DB) PGSSLMODE=disable APPLY_MIGRATIONS=$(APPLY_MIGRATIONS) go run ./server

# Print resolved environment values for verification
env-print:
	@echo "POSTGRES_USER=$(POSTGRES_USER)"
	@echo "POSTGRES_PASSWORD=$(POSTGRES_PASSWORD)"
	@echo "POSTGRES_DB=$(POSTGRES_DB)"
	@echo "POSTGRES_PORT=$(POSTGRES_PORT)"
	@echo "APPLY_MIGRATIONS=$(APPLY_MIGRATIONS)"
	@echo "DATABASE_URL=$(DATABASE_URL)"

test:
	go test ./...

lint:
	@echo "(placeholder) integrate golangci-lint or staticcheck"

proto: $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)
	@echo "Generating protobuf stubs (proto/service.proto must exist)"
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/service.proto

$(PROTOC_GEN_GO):
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

$(PROTOC_GEN_GO_GRPC):
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

tools: $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)

clean:
	rm -rf dist

# Docker Postgres helpers
db-up:
	docker compose up -d db

db-down:
	docker compose down -v

db-logs:
	docker compose logs -f db

# Open psql in the container
db-psql:
	docker compose exec -e PGPASSWORD=$${POSTGRES_PASSWORD:-postgres} db psql -U $${POSTGRES_USER:-postgres} -d $${POSTGRES_DB:-observer}