package server

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	observer "github.com/stanterprise/proto-go/testsystem/v1/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

func TestGRPCServerReceiveMessageSize(t *testing.T) {
	const limit = 16 * 1024 * 1024

	for _, mode := range []struct {
		name string
		key  string
	}{
		{name: "local"},
		{name: "authenticated", key: testIngestionAuthKey},
	} {
		t.Run(mode.name, func(t *testing.T) {
			var server *grpc.Server
			if mode.key == "" {
				server = NewGRPCServer(nil)
			} else {
				var err error
				server, err = NewAuthenticatedGRPCServer(nil, mode.key)
				if err != nil {
					t.Fatalf("create authenticated gRPC server: %v", err)
				}
			}
			RegisterServices(server, nil, nil)

			listener := bufconn.Listen(1024 * 1024)
			go func() {
				_ = server.Serve(listener)
			}()
			t.Cleanup(func() {
				server.Stop()
				_ = listener.Close()
			})

			conn, err := grpc.NewClient(
				"passthrough:///bufnet",
				grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
					return listener.DialContext(ctx)
				}),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(limit+1)),
			)
			if err != nil {
				t.Fatalf("create gRPC client: %v", err)
			}
			t.Cleanup(func() {
				_ = conn.Close()
			})

			client := observer.NewTestEventCollectorClient(conn)
			healthClient := healthpb.NewHealthClient(conn)
			for _, tc := range []struct {
				name string
				size int
				want codes.Code
			}{
				{name: "below limit", size: limit - 1, want: codes.OK},
				{name: "at limit", size: limit, want: codes.OK},
				{name: "above limit", size: limit + 1, want: codes.ResourceExhausted},
			} {
				t.Run(tc.name, func(t *testing.T) {
					request := &events.HeartbeatEventRequest{SourceId: strings.Repeat("x", tc.size)}
					overhead := proto.Size(request) - len(request.SourceId)
					request.SourceId = request.SourceId[:tc.size-overhead]
					if size := proto.Size(request); size != tc.size {
						t.Fatalf("protobuf request size = %d, want %d", size, tc.size)
					}

					t.Run("unary", func(t *testing.T) {
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						if mode.key != "" {
							ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+mode.key)
						}
						response, err := client.Heartbeat(ctx, request)
						if got := status.Code(err); got != tc.want {
							t.Fatalf("Heartbeat status = %s, want %s (error: %v)", got, tc.want, err)
						}
						if tc.want == codes.OK && !response.GetSuccess() {
							t.Fatal("Heartbeat did not acknowledge the request")
						}
					})

					t.Run("streaming", func(t *testing.T) {
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						healthRequest := &healthpb.HealthCheckRequest{Service: request.SourceId}
						if size := proto.Size(healthRequest); size != tc.size {
							t.Fatalf("protobuf streaming request size = %d, want %d", size, tc.size)
						}
						stream, err := healthClient.Watch(ctx, healthRequest)
						if err == nil {
							var response *healthpb.HealthCheckResponse
							response, err = stream.Recv()
							if err == nil && response.GetStatus() != healthpb.HealthCheckResponse_SERVICE_UNKNOWN {
								t.Fatalf("Watch status = %s, want SERVICE_UNKNOWN", response.GetStatus())
							}
						}
						if got := status.Code(err); got != tc.want {
							t.Fatalf("Watch status = %s, want %s (error: %v)", got, tc.want, err)
						}
					})
				})
			}
		})
	}
}
