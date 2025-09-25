package main

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stanterprise/proto-go/testsystem/v1/common"
	entities "github.com/stanterprise/proto-go/testsystem/v1/entities"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	observer "github.com/stanterprise/proto-go/testsystem/v1/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

    _, err := client.ReportTestBegin(ctx, &events.TestBeginEventRequest{
        TestCase: &entities.TestCase{
            Id:      "test-id",
            Name:    "test-name",
            Metadata: map[string]string{"k": "v"},
        },
        
    })
    if err != nil { t.Fatalf("start failed: %v", err) }

    _, err = client.ReportTestEnd(ctx, &events.TestEndEventRequest{
        TestCase: &entities.TestCase{
            Id:     "test-id",
            Status: common.TestStatus_PASSED,
        },
        // EndTime is optional; server uses current time if not provided.

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

    _, err := client.ReportTestBegin(ctx, &events.TestBeginEventRequest{TestCase: &entities.TestCase{Id: ""}})
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
    _, err := client.ReportTestBegin(ctx, &events.TestBeginEventRequest{TestCase: &entities.TestCase{Id: "tid", Name: "n"}})
    if err != nil { t.Fatalf("start failed: %v", err) }

    _, err = client.ReportStepBegin(ctx, &events.StepBeginEventRequest{Step: &entities.Step{TestId: "tid"}})
    if err != nil { t.Fatalf("step failed: %v", err) }
}

func TestReportStartInvalidTable(t *testing.T) {
    cases := []struct{ name string; req *events.TestBeginEventRequest }{
        {"empty-id", &events.TestBeginEventRequest{TestCase: &entities.TestCase{Id: ""}}},
        {"nil-req", nil},
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            conn := dialBufConn(t)
            defer conn.Close()
            client := observer.NewTestEventCollectorClient(conn)
            ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
            defer cancel()
            _, err := client.ReportTestBegin(ctx, c.req)
            if err == nil { t.Fatalf("expected error for case %s", c.name) }
            st, ok := status.FromError(err)
            if !ok { t.Fatalf("expected status error") }
            if st.Code() != codes.InvalidArgument { t.Fatalf("expected InvalidArgument, got %v", st.Code()) }
        })
    }
}
