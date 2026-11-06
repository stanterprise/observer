package server

import (
	"context"
	"net"
	"testing"

	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	observer "github.com/stanterprise/proto-go/testsystem/v1/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const testIngestionAuthKey = "test-ingestion-auth-key-with-32-bytes-minimum"

func TestAuthenticatedGRPCServerRequiresBearerKey(t *testing.T) {
	server, err := NewAuthenticatedGRPCServer(nil, testIngestionAuthKey)
	if err != nil {
		t.Fatalf("create authenticated gRPC server: %v", err)
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
	)
	if err != nil {
		t.Fatalf("create gRPC client: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	client := observer.NewTestEventCollectorClient(conn)
	for _, tc := range []struct {
		name   string
		values []string
		want   codes.Code
	}{
		{name: "missing token", want: codes.Unauthenticated},
		{name: "invalid token", values: []string{"Bearer wrong"}, want: codes.Unauthenticated},
		{name: "malformed token", values: []string{"Basic " + testIngestionAuthKey}, want: codes.Unauthenticated},
		{name: "duplicate token", values: []string{"Bearer " + testIngestionAuthKey, "Bearer " + testIngestionAuthKey}, want: codes.Unauthenticated},
		{name: "valid token", values: []string{"Bearer " + testIngestionAuthKey}, want: codes.OK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if len(tc.values) != 0 {
				ctx = metadata.NewOutgoingContext(ctx, metadata.MD{"authorization": tc.values})
			}
			_, err := client.Heartbeat(ctx, &events.HeartbeatEventRequest{SourceId: "test-runner"})
			if got := status.Code(err); got != tc.want {
				t.Fatalf("Heartbeat status = %s, want %s (error: %v)", got, tc.want, err)
			}
		})
	}

	if _, err := healthpb.NewHealthClient(conn).Check(context.Background(), &healthpb.HealthCheckRequest{}); err != nil {
		t.Fatalf("health check should remain available without an ingestion key: %v", err)
	}
}

func TestNewAuthenticatedGRPCServerRejectsInvalidKeys(t *testing.T) {
	for _, key := range []string{"", "short", "has surrounding whitespace                                ", "key with spaces of sufficient length"} {
		if _, err := NewAuthenticatedGRPCServer(nil, key); err == nil {
			t.Errorf("NewAuthenticatedGRPCServer accepted invalid key %q", key)
		}
	}
}
