PROTOC_GEN_GO := $(shell go env GOPATH)/bin/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(shell go env GOPATH)/bin/protoc-gen-go-grpc

.PHONY: proto
proto: $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/service.proto

$(PROTOC_GEN_GO):
	go get google.golang.org/protobuf/cmd/protoc-gen-go

$(PROTOC_GEN_GO_GRPC):
	go get google.golang.org/grpc/cmd/protoc-gen-go-grpc