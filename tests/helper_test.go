package main

import (
	"context"
	"net"
	"testing"
	"time"

	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	observer "github.com/stanterprise/proto-go/testsystem/v1/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// dialBufConn returns a client connection to the in-process gRPC server.
func dialBufConn(t *testing.T) *grpc.ClientConn {
    t.Helper()
    ctx := context.Background()
    conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
        return testBufListener.Dial()
    }), grpc.WithInsecure())
    if err != nil { t.Fatalf("failed to dial bufnet: %v", err) }
    return conn
}

// TestReportLifecycle tests start and finish flow.
func TestReportLifecycle(t *testing.T) {
    conn := dialBufConn(t)
    defer conn.Close()
    client := observer.NewTestEventCollectorClient(conn)
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    _, err := client.ReportTestStart(ctx, &events.TestStartEventRequest{
        TestId:    "test-id",
        TestName:  "test-name",
        StartTime: timestamppb.New(time.Now()),
        Metadata:  map[string]string{"k": "v"},
    })
    if err != nil { t.Fatalf("start failed: %v", err) }

    _, err = client.ReportTestFinish(ctx, &events.TestFinishEventRequest{
        TestId:  "test-id",
        EndTime: timestamppb.New(time.Now()),
    })
    if err != nil { t.Fatalf("finish failed: %v", err) }
}

// TestReportStartInvalidID ensures empty test ID is rejected.
func TestReportStartInvalidID(t *testing.T) {
    conn := dialBufConn(t)
    defer conn.Close()
    client := observer.NewTestEventCollectorClient(conn)
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    _, err := client.ReportTestStart(ctx, &events.TestStartEventRequest{ TestId: "" })
    if err == nil { t.Fatalf("expected error for empty id") }
    st, ok := status.FromError(err)
    if !ok { t.Fatalf("expected status error, got %v", err) }
    if st.Code() != codes.InvalidArgument { t.Fatalf("expected InvalidArgument, got %v", st.Code()) }
}

func TestReportStep(t *testing.T) {
    conn := dialBufConn(t)
    defer conn.Close()
    client := observer.NewTestEventCollectorClient(conn)
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    // Must start test first
    _, err := client.ReportTestStart(ctx, &events.TestStartEventRequest{TestId: "tid", TestName: "n"})
    if err != nil { t.Fatalf("start failed: %v", err) }

    _, err = client.ReportTestStep(ctx, &events.TestStepEventRequest{TestId: "tid"})
    if err != nil { t.Fatalf("step failed: %v", err) }
}

func TestReportStartInvalidTable(t *testing.T) {
    cases := []struct{ name string; req *events.TestStartEventRequest }{
        {"empty-id", &events.TestStartEventRequest{TestId: ""}},
        {"nil-req", nil},
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            conn := dialBufConn(t)
            defer conn.Close()
            client := observer.NewTestEventCollectorClient(conn)
            ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
            defer cancel()
            _, err := client.ReportTestStart(ctx, c.req)
            if err == nil { t.Fatalf("expected error for case %s", c.name) }
            st, ok := status.FromError(err)
            if !ok { t.Fatalf("expected status error") }
            if st.Code() != codes.InvalidArgument { t.Fatalf("expected InvalidArgument, got %v", st.Code()) }
        })
    }
}
