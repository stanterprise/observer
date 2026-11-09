package server

import (
	"context"
	"log/slog"
	"net"
	"os"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestReflectionIsOptIn(t *testing.T) {
	tests := []struct {
		name              string
		enableReflection  bool
		wantServiceListed bool
	}{
		{name: "disabled by default"},
		{name: "enabled explicitly", enableReflection: true, wantServiceListed: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener := bufconn.Listen(1024 * 1024)
			server := NewGRPCServer(slog.New(slog.NewTextHandler(os.Stderr, nil)))
			RegisterServices(server, nil, nil)
			if tt.enableReflection {
				RegisterReflection(server)
			}
			go func() {
				_ = server.Serve(listener)
			}()
			t.Cleanup(func() {
				server.Stop()
				_ = listener.Close()
			})

			ctx := context.Background()
			conn, err := grpc.NewClient(
				"passthrough:///bufnet",
				grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
					return listener.DialContext(ctx)
				}),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				t.Fatalf("create gRPC client: %v", err)
			}
			t.Cleanup(func() {
				_ = conn.Close()
			})

			stream, err := grpc_reflection_v1.NewServerReflectionClient(conn).ServerReflectionInfo(ctx)
			if err != nil {
				assertReflectionResult(t, tt.wantServiceListed, err)
				return
			}
			if err := stream.Send(&grpc_reflection_v1.ServerReflectionRequest{
				MessageRequest: &grpc_reflection_v1.ServerReflectionRequest_ListServices{
					ListServices: "",
				},
			}); err != nil {
				assertReflectionResult(t, tt.wantServiceListed, err)
				return
			}
			response, err := stream.Recv()
			if err != nil {
				assertReflectionResult(t, tt.wantServiceListed, err)
				return
			}
			if !tt.wantServiceListed {
				t.Fatal("reflection unexpectedly returned a response")
			}
			if len(response.GetListServicesResponse().GetService()) == 0 {
				t.Fatal("reflection response did not list any services")
			}
		})
	}
}

func assertReflectionResult(t *testing.T, wantServiceListed bool, err error) {
	t.Helper()
	if wantServiceListed {
		t.Fatalf("reflection request failed: %v", err)
	}
	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("reflection should be disabled with Unimplemented, got %v", err)
	}
}
