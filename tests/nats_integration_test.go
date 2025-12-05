package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stanterprise/observer/pkg/publisher"
	"github.com/stanterprise/proto-go/testsystem/v1/common"
	"github.com/stanterprise/proto-go/testsystem/v1/entities"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	observer "github.com/stanterprise/proto-go/testsystem/v1/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// TestNATSIntegration tests the full NATS publishing flow
// This test only runs if NATS_TEST_URL is set (e.g., NATS_TEST_URL=nats://localhost:4222)
func TestNATSIntegration(t *testing.T) {
	natsURL := os.Getenv("NATS_TEST_URL")
	if natsURL == "" {
		t.Skip("Skipping NATS integration test - NATS_TEST_URL not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create unique stream name for this test run
	streamName := "test_events_" + time.Now().Format("20060102150405")
	subjectPrefix := "test.events.v1"

	// Initialize NATS publisher
	cfg := publisher.NATSConfig{
		URL:           natsURL,
		StreamName:    streamName,
		SubjectPrefix: subjectPrefix,
	}

	pub, err := publisher.NewNATSPublisher(cfg, logger)
	if err != nil {
		t.Fatalf("NewNATSPublisher() error = %v", err)
	}
	defer func() {
		// Cleanup: delete the test stream
		ctx := context.Background()
		if err := pub.Close(); err != nil {
			t.Logf("Warning: failed to close publisher: %v", err)
		}

		// Connect directly to NATS to delete stream
		nc, err := nats.Connect(natsURL)
		if err == nil {
			js, err := jetstream.New(nc)
			if err == nil {
				_ = js.DeleteStream(ctx, streamName)
			}
			nc.Close()
		}
	}()

	// Create a test gRPC server with NATS publisher
	testListener := bufconn.Listen(bufSize)
	grpcServer := newTestGRPCServerWithNATS(logger, pub)
	go func() {
		_ = grpcServer.Serve(&listenerWrapper{Listener: testListener})
	}()
	defer grpcServer.Stop()

	// Create a gRPC client
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return testListener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := observer.NewTestEventCollectorClient(conn)

	// Create a consumer to receive events
	nc, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("Failed to create JetStream context: %v", err)
	}

	consumer, err := js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Durable:   "test_consumer_" + time.Now().Format("20060102150405"),
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateConsumer() error = %v", err)
	}

	// Test 1: Report Test Begin
	t.Run("ReportTestBegin publishes to NATS", func(t *testing.T) {
		req := &events.TestBeginEventRequest{
			TestCase: &entities.TestCaseRun{
				Id:       "test-nats-1",
				RunId:    "run-nats-1",
				Name:    "NATS Integration Test",
				Metadata: map[string]string{"env": "test"},
			},
		}

		resp, err := client.ReportTestBegin(ctx, req)
		if err != nil {
			t.Fatalf("ReportTestBegin() error = %v", err)
		}
		if !resp.Success {
			t.Error("Expected success response")
		}

		// Verify event was published to NATS
		msgs, err := consumer.Fetch(1, jetstream.FetchMaxWait(2*time.Second))
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		msgReceived := false
		for msg := range msgs.Messages() {
			msgReceived = true
			if err := msg.Ack(); err != nil {
				t.Errorf("Failed to ack message: %v", err)
			}

			// Parse the event
			var event publisher.Event
			if err := json.Unmarshal(msg.Data(), &event); err != nil {
				t.Errorf("Failed to unmarshal event: %v", err)
			}

			if event.Type != publisher.EventTypeTestBegin {
				t.Errorf("Event type = %v, want %v", event.Type, publisher.EventTypeTestBegin)
			}

			t.Logf("Received event: type=%s, timestamp=%s", event.Type, event.Timestamp)
		}

		if !msgReceived {
			t.Error("Expected to receive test begin event")
		}
	})

	// Test 2: Report Test End
	t.Run("ReportTestEnd publishes to NATS", func(t *testing.T) {
		req := &events.TestEndEventRequest{
			TestCase: &entities.TestCaseRun{
				Id:     "test-nats-2",
				RunId:  "run-nats-2",
				Status: common.TestStatus_PASSED,
			},
		}

		resp, err := client.ReportTestEnd(ctx, req)
		if err != nil {
			t.Fatalf("ReportTestEnd() error = %v", err)
		}
		if !resp.Success {
			t.Error("Expected success response")
		}

		// Verify event was published to NATS
		msgs, err := consumer.Fetch(1, jetstream.FetchMaxWait(2*time.Second))
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		msgReceived := false
		for msg := range msgs.Messages() {
			msgReceived = true
			msg.Ack()

			var event publisher.Event
			if err := json.Unmarshal(msg.Data(), &event); err != nil {
				t.Errorf("Failed to unmarshal event: %v", err)
			}

			if event.Type != publisher.EventTypeTestEnd {
				t.Errorf("Event type = %v, want %v", event.Type, publisher.EventTypeTestEnd)
			}

			t.Logf("Received event: type=%s, timestamp=%s", event.Type, event.Timestamp)
		}

		if !msgReceived {
			t.Error("Expected to receive test end event")
		}
	})

	// Test 3: Report Step Begin
	t.Run("ReportStepBegin publishes to NATS", func(t *testing.T) {
		req := &events.StepBeginEventRequest{
			Step: &entities.StepRun{
				TestCaseRunId: "test-nats-3",
			},
		}

		resp, err := client.ReportStepBegin(ctx, req)
		if err != nil {
			t.Fatalf("ReportStepBegin() error = %v", err)
		}
		if !resp.Success {
			t.Error("Expected success response")
		}

		// Verify event was published to NATS
		msgs, err := consumer.Fetch(1, jetstream.FetchMaxWait(2*time.Second))
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		msgReceived := false
		for msg := range msgs.Messages() {
			msgReceived = true
			msg.Ack()

			var event publisher.Event
			if err := json.Unmarshal(msg.Data(), &event); err != nil {
				t.Errorf("Failed to unmarshal event: %v", err)
			}

			if event.Type != publisher.EventTypeStepBegin {
				t.Errorf("Event type = %v, want %v", event.Type, publisher.EventTypeStepBegin)
			}

			t.Logf("Received event: type=%s, timestamp=%s", event.Type, event.Timestamp)
		}

		if !msgReceived {
			t.Error("Expected to receive step begin event")
		}
	})

	// Test 4: Report Step End
	t.Run("ReportStepEnd publishes to NATS", func(t *testing.T) {
		req := &events.StepEndEventRequest{
			Step: &entities.StepRun{
				TestCaseRunId: "test-nats-4",
				Status:        common.TestStatus_PASSED,
			},
		}

		resp, err := client.ReportStepEnd(ctx, req)
		if err != nil {
			t.Fatalf("ReportStepEnd() error = %v", err)
		}
		if !resp.Success {
			t.Error("Expected success response")
		}

		// Verify event was published to NATS
		msgs, err := consumer.Fetch(1, jetstream.FetchMaxWait(2*time.Second))
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		msgReceived := false
		for msg := range msgs.Messages() {
			msgReceived = true
			msg.Ack()

			var event publisher.Event
			if err := json.Unmarshal(msg.Data(), &event); err != nil {
				t.Errorf("Failed to unmarshal event: %v", err)
			}

			if event.Type != publisher.EventTypeStepEnd {
				t.Errorf("Event type = %v, want %v", event.Type, publisher.EventTypeStepEnd)
			}

			t.Logf("Received event: type=%s, timestamp=%s", event.Type, event.Timestamp)
		}

		if !msgReceived {
			t.Error("Expected to receive step end event")
		}
	})
}
