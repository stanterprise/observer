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
	"github.com/stanterprise/observer/internal/database"
	"github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/pkg/consumer"
	"github.com/stanterprise/observer/pkg/publisher"
	"github.com/stanterprise/proto-go/testsystem/v1/common"
	"github.com/stanterprise/proto-go/testsystem/v1/entities"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	observer "github.com/stanterprise/proto-go/testsystem/v1/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// TestEndToEndIntegration tests the complete flow: gRPC → NATS → Consumer → Database
// This test validates the distributed architecture with actual NATS and database
func TestEndToEndIntegration(t *testing.T) {
	natsURL := os.Getenv("NATS_TEST_URL")
	if natsURL == "" {
		t.Skip("Skipping E2E integration test - NATS_TEST_URL not set. Run: NATS_TEST_URL=nats://localhost:4222 go test")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create unique stream name for this test
	streamName := "e2e_test_" + time.Now().Format("20060102150405")
	subjectPrefix := "e2e.events.v1"

	// Initialize NATS publisher
	pubCfg := publisher.NATSConfig{
		URL:           natsURL,
		StreamName:    streamName,
		SubjectPrefix: subjectPrefix,
	}

	pub, err := publisher.NewNATSPublisher(pubCfg, logger)
	if err != nil {
		t.Fatalf("NewNATSPublisher() error = %v", err)
	}

	defer func() {
		pub.Close()

		// Cleanup: delete the test stream
		nc, err := nats.Connect(natsURL)
		if err == nil {
			js, err := jetstream.New(nc)
			if err == nil {
				_ = js.DeleteStream(context.Background(), streamName)
			}
			nc.Close()
		}
	}()

	// Setup temporary file-based SQLite database for testing
	// Note: Using file-based DB instead of :memory: to ensure consumer sees the same database
	tmpFile := "test_" + time.Now().Format("20060102150405") + ".db"
	defer os.Remove(tmpFile)

	db, err := database.Connect(tmpFile, logger)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.AutoMigrate(&models.TestCaseRun{}, &models.StepRun{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Setup gRPC server with NATS publisher
	testListener := bufconn.Listen(bufSize)
	grpcServer := newTestGRPCServerWithNATS(logger, pub)
	go func() {
		_ = grpcServer.Serve(&listenerWrapper{Listener: testListener})
	}()
	defer grpcServer.Stop()

	// Create gRPC client
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

	// Setup NATS consumer
	consumerCfg := consumer.NATSConsumerConfig{
		URL:          natsURL,
		StreamName:   streamName,
		ConsumerName: "e2e_consumer_" + time.Now().Format("20060102150405"),
		BatchSize:    5,
		MaxWait:      2 * time.Second,
	}

	natsConsumer, err := consumer.NewNATSConsumer(consumerCfg, logger, db)
	if err != nil {
		t.Fatalf("NewNATSConsumer() error = %v", err)
	}

	// Start consumer in background
	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	defer cancelConsumer()

	go func() {
		_ = natsConsumer.Start(consumerCtx, consumerCfg)
	}()

	// Give consumer time to start
	time.Sleep(500 * time.Millisecond)

	// Execute test scenario
	testID := "e2e-test-1"
	runID := "e2e-run-1"
	stepID := "e2e-step-1"

	t.Run("Full E2E Flow", func(t *testing.T) {
		// 1. Send TestBegin event via gRPC
		beginReq := &events.TestBeginEventRequest{
			TestCase: &entities.TestCaseRun{
				Id:    testID,
				RunId: runID,
				Name: "E2E Integration Test",
				Metadata: map[string]string{
					"environment": "test",
					"framework":   "playwright",
				},
			},
		}

		resp, err := client.ReportTestBegin(ctx, beginReq)
		if err != nil {
			t.Fatalf("ReportTestBegin failed: %v", err)
		}
		if !resp.Success {
			t.Error("Expected success response")
		}

		// 2. Send StepBegin event
		stepBeginReq := &events.StepBeginEventRequest{
			Step: &entities.StepRun{
				Id:            stepID,
				RunId:         runID,
				TestCaseRunId: testID,
				Title:         "Navigate to page",
			},
		}

		resp, err = client.ReportStepBegin(ctx, stepBeginReq)
		if err != nil {
			t.Fatalf("ReportStepBegin failed: %v", err)
		}
		if !resp.Success {
			t.Error("Expected success response")
		}

		// 3. Send StepEnd event
		stepEndReq := &events.StepEndEventRequest{
			Step: &entities.StepRun{
				Id:            stepID,
				RunId:         runID,
				TestCaseRunId: testID,
				Status:        common.TestStatus_PASSED,
			},
		}

		resp, err = client.ReportStepEnd(ctx, stepEndReq)
		if err != nil {
			t.Fatalf("ReportStepEnd failed: %v", err)
		}
		if !resp.Success {
			t.Error("Expected success response")
		}

		// 4. Send TestEnd event
		endReq := &events.TestEndEventRequest{
			TestCase: &entities.TestCaseRun{
				Id:     testID,
				RunId:  runID,
				Status: common.TestStatus_PASSED,
			},
		}

		resp, err = client.ReportTestEnd(ctx, endReq)
		if err != nil {
			t.Fatalf("ReportTestEnd failed: %v", err)
		}
		if !resp.Success {
			t.Error("Expected success response")
		}

		// 5. Wait for consumer to process events
		// Poll database until data appears or timeout
		deadline := time.Now().Add(10 * time.Second)
		testFound := false
		stepFound := false
		statusUpdated := false

		for time.Now().Before(deadline) && (!testFound || !stepFound || !statusUpdated) {
			var testCase models.TestCaseRun
			result := db.Where("id = ?", testID).First(&testCase)
			if result.Error == nil {
				testFound = true

				// Check if status was updated (status is set on TestEnd)
				if testCase.Status == common.TestStatus_PASSED.String() {
					statusUpdated = true

					// Verify test case data
					if testCase.Title != "E2E Integration Test" {
						t.Errorf("Expected title 'E2E Integration Test', got '%s'", testCase.Title)
					}
					if testCase.Metadata["environment"] != "test" {
						t.Errorf("Expected metadata environment=test, got %v", testCase.Metadata["environment"])
					}
				}
			}

			var step models.StepRun
			result = db.Where("test_case_run_id = ?", testID).First(&step)
			if result.Error == nil {
				stepFound = true

				// Verify step data
				if step.Status != "" && step.Status != "RUNNING" {
					// Step status has been updated to final state
					if step.Status != common.TestStatus_PASSED.String() {
						t.Errorf("Expected step status PASSED, got %v", step.Status)
					}
				}
				if step.TestCaseRunID != testID {
					t.Errorf("Expected test_case_run_id '%s', got '%s'", testID, step.TestCaseRunID)
				}
			}

			if !testFound || !stepFound || !statusUpdated {
				time.Sleep(500 * time.Millisecond)
			}
		}

		if !testFound {
			t.Error("Test case not found in database after timeout")
		}
		if !stepFound {
			t.Error("Step not found in database after timeout")
		}
		if !statusUpdated {
			t.Error("Test status not updated to PASSED after timeout")
		}
	})

	// Test event order and consistency
	t.Run("Event Order Consistency", func(t *testing.T) {
		// Query all events for the test
		var testCase models.TestCaseRun
		result := db.Where("id = ?", testID).First(&testCase)
		if result.Error != nil {
			t.Fatalf("Failed to query test case: %v", result.Error)
		}

		// Verify timestamps
		if testCase.CreatedAt.After(testCase.UpdatedAt) {
			t.Error("Test case created_at should be before or equal to updated_at")
		}

		// Query step
		var step models.StepRun
		result = db.Where("test_case_run_id = ?", testID).First(&step)
		if result.Error != nil {
			t.Fatalf("Failed to query step: %v", result.Error)
		}

		// Step should be created after or at same time as test
		if step.CreatedAt.Before(testCase.CreatedAt) {
			t.Error("Step should be created after test case")
		}
	})
}

// TestNATSEventFormat validates the event format in NATS
func TestNATSEventFormat(t *testing.T) {
	natsURL := os.Getenv("NATS_TEST_URL")
	if natsURL == "" {
		t.Skip("Skipping NATS event format test - NATS_TEST_URL not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	streamName := "format_test_" + time.Now().Format("20060102150405")
	subjectPrefix := "format.events.v1"

	cfg := publisher.NATSConfig{
		URL:           natsURL,
		StreamName:    streamName,
		SubjectPrefix: subjectPrefix,
	}

	pub, err := publisher.NewNATSPublisher(cfg, logger)
	if err != nil {
		t.Fatalf("NewNATSPublisher() error = %v", err)
	}
	defer pub.Close()

	// Connect to NATS and create consumer
	nc, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("Failed to create JetStream context: %v", err)
	}

	defer func() {
		_ = js.DeleteStream(context.Background(), streamName)
	}()

	cons, err := js.CreateOrUpdateConsumer(context.Background(), streamName, jetstream.ConsumerConfig{
		Durable:   "format_consumer",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateConsumer() error = %v", err)
	}

	// Publish a test event
	testReq := &events.TestBeginEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:    "format-test-1",
			RunId: "format-run-1",
			Name: "Format Validation Test",
		},
	}

	ctx := context.Background()
	if err := pub.Publish(ctx, publisher.EventTypeTestBegin, testReq); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	// Fetch and validate event format
	msgs, err := cons.Fetch(1, jetstream.FetchMaxWait(2*time.Second))
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	msgReceived := false
	for msg := range msgs.Messages() {
		msgReceived = true
		if err := msg.Ack(); err != nil {
			t.Errorf("Failed to ack message: %v", err)
		}

		// Parse event envelope
		var event publisher.Event
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			t.Fatalf("Failed to unmarshal event envelope: %v", err)
		}

		// Validate envelope fields
		if event.Type != publisher.EventTypeTestBegin {
			t.Errorf("Expected type %s, got %s", publisher.EventTypeTestBegin, event.Type)
		}

		if event.Timestamp.IsZero() {
			t.Error("Event timestamp is zero")
		}

		// Validate event data can be unmarshaled
		var req events.TestBeginEventRequest
		if err := json.Unmarshal(event.Data, &req); err != nil {
			t.Fatalf("Failed to unmarshal event data: %v", err)
		}

		if req.TestCase.Id != "format-test-1" {
			t.Errorf("Expected test ID 'format-test-1', got '%s'", req.TestCase.Id)
		}

		t.Logf("Successfully validated event format: type=%s, timestamp=%s", event.Type, event.Timestamp)
	}

	if !msgReceived {
		t.Error("No message received from NATS")
	}
}
