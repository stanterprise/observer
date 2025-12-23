package main

import (
	"context"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/stanterprise/observer/pkg/publisher"
	obsrv "github.com/stanterprise/observer/pkg/server"
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
	if err != nil {
		t.Fatalf("failed to dial bufnet: %v", err)
	}
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
		TestCase: &entities.TestCaseRun{
			Id:       "test-id",
			RunId:    "test-id",
			Name:     "test-name",
			Metadata: map[string]string{"k": "v"},
		},
	})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}

	_, err = client.ReportTestEnd(ctx, &events.TestEndEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:     "test-id",
			RunId:  "test-id",
			Status: common.TestStatus_PASSED,
		},
		// EndTime is optional; server uses current time if not provided.

	})
	if err != nil {
		t.Fatalf("finish failed: %v", err)
	}
}

// TestReportStartInvalidID ensures empty test ID is rejected.
func TestReportStartInvalidID(t *testing.T) {
	conn := dialBufConn(t)
	defer conn.Close()
	client := observer.NewTestEventCollectorClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.ReportTestBegin(ctx, &events.TestBeginEventRequest{TestCase: &entities.TestCaseRun{Id: ""}})
	if err == nil {
		t.Fatalf("expected error for empty id")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestReportStep(t *testing.T) {
	conn := dialBufConn(t)
	defer conn.Close()
	client := observer.NewTestEventCollectorClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Must start test first
	_, err := client.ReportTestBegin(ctx, &events.TestBeginEventRequest{TestCase: &entities.TestCaseRun{Id: "test-id", RunId: "tid", Name: "n"}})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}

	_, err = client.ReportStepBegin(ctx, &events.StepBeginEventRequest{Step: &entities.StepRun{Id: "step-id", RunId: "tid", TestCaseRunId: "test-id"}})
	if err != nil {
		t.Fatalf("step failed: %v", err)
	}
}

func TestReportStartInvalidTable(t *testing.T) {
	cases := []struct {
		name string
		req  *events.TestBeginEventRequest
	}{
		{"empty-id", &events.TestBeginEventRequest{TestCase: &entities.TestCaseRun{RunId: ""}}},
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
			if err == nil {
				t.Fatalf("expected error for case %s", c.name)
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected status error")
			}
			if st.Code() != codes.InvalidArgument {
				t.Fatalf("expected InvalidArgument, got %v", st.Code())
			}
		})
	}
}

// newTestGRPCServerWithNATS creates a test gRPC server with NATS publisher
func newTestGRPCServerWithNATS(logger *slog.Logger, pub *publisher.NATSPublisher) *grpc.Server {
	grpcServer := obsrv.NewGRPCServer(logger)
	obsrv.RegisterServicesWithPublisher(grpcServer, logger, nil, pub)
	return grpcServer
}

// TestMapTestRun tests the MapTestRun endpoint
func TestMapTestRun(t *testing.T) {
	conn := dialBufConn(t)
	defer conn.Close()
	client := observer.NewTestEventCollectorClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Valid request with run_id and test suites
	_, err := client.MapTestRun(ctx, &events.MapTestRunEventRequest{
		RunId: "run-123",
		TestSuites: []*entities.TestSuiteRun{
			{
				Id:       "suite-1",
				Name:     "Test Suite 1",
				Metadata: map[string]string{"key": "value"},
			},
			{
				Id:       "suite-2",
				Name:     "Test Suite 2",
				Metadata: map[string]string{"env": "prod"},
			},
		},
	})
	if err != nil {
		t.Fatalf("MapTestRun failed: %v", err)
	}
}

// TestMapTestRunInvalidInput tests validation of MapTestRun endpoint
func TestMapTestRunInvalidInput(t *testing.T) {
	cases := []struct {
		name string
		req  *events.MapTestRunEventRequest
	}{
		{"nil-req", nil},
		{"empty-run-id", &events.MapTestRunEventRequest{
			RunId:      "",
			TestSuites: []*entities.TestSuiteRun{{Id: "suite-1"}},
		}},
		{"empty-test-suites", &events.MapTestRunEventRequest{
			RunId:      "run-123",
			TestSuites: []*entities.TestSuiteRun{},
		}},
		{"nil-test-suites", &events.MapTestRunEventRequest{
			RunId:      "run-123",
			TestSuites: nil,
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			conn := dialBufConn(t)
			defer conn.Close()
			client := observer.NewTestEventCollectorClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, err := client.MapTestRun(ctx, c.req)
			if err == nil {
				t.Fatalf("expected error for case %s", c.name)
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected status error")
			}
			if st.Code() != codes.InvalidArgument {
				t.Fatalf("expected InvalidArgument, got %v", st.Code())
			}
		})
	}
}
