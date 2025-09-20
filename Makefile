GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
  GOBIN := $(shell go env GOPATH)/bin
endif

PROTOC_GEN_GO := $(GOBIN)/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(GOBIN)/protoc-gen-go-grpc

.PHONY: all build run test lint proto tools clean

all: build

build:
	go build ./...

run: build
	go run ./server

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