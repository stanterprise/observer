package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stanterprise/observer/pkg/publisher"
	"github.com/stanterprise/proto-go/testsystem/v1/common"
	"github.com/stanterprise/proto-go/testsystem/v1/entities"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// TestAllEventTypes verifies that all 11 event types are properly published to NATS
func TestAllEventTypes(t *testing.T) {
	natsURL := os.Getenv("NATS_TEST_URL")
	if natsURL == "" {
		t.Skip("Skipping all events test - NATS_TEST_URL not set")
	}

	ctx := context.Background()

	// Connect to NATS
	nc, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("Failed to create JetStream context: %v", err)
	}

	// Create publisher
	pub, err := publisher.NewNATSPublisher(publisher.NATSConfig{
		URL:        natsURL,
		StreamName: "test_all_events",
	}, nil)
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}
	defer pub.Close()

	// Create a consumer to verify messages
	consumerName := "test-all-events-consumer"
	cons, err := js.CreateOrUpdateConsumer(ctx, "test_all_events", jetstream.ConsumerConfig{
		Durable:       consumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}

	// Map of event types to verify
	expectedEvents := map[publisher.EventType]bool{
		publisher.EventTypeTestBegin:   false,
		publisher.EventTypeTestEnd:     false,
		publisher.EventTypeStepBegin:   false,
		publisher.EventTypeStepEnd:     false,
		publisher.EventTypeSuiteBegin:  false,
		publisher.EventTypeSuiteEnd:    false,
		publisher.EventTypeTestFailure: false,
		publisher.EventTypeTestError:   false,
		publisher.EventTypeStdOutput:   false,
		publisher.EventTypeStdError:    false,
		publisher.EventTypeHeartbeat:   false,
	}

	// Publish all event types
	t.Log("Publishing all event types...")

	// 1. SuiteBegin
	if err := pub.Publish(ctx, publisher.EventTypeSuiteBegin, &events.SuiteBeginEventRequest{
		Suite: &entities.TestSuiteRun{
			Id:   "suite-1",
			Name: "Test Suite",
		},
	}); err != nil {
		t.Errorf("Failed to publish SuiteBegin: %v", err)
	}

	// 2. TestBegin
	if err := pub.Publish(ctx, publisher.EventTypeTestBegin, &events.TestBeginEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:    "test-1",
			RunId: "run-1",
			Title: "Test Case 1",
		},
	}); err != nil {
		t.Errorf("Failed to publish TestBegin: %v", err)
	}

	// 3. StdOutput
	if err := pub.Publish(ctx, publisher.EventTypeStdOutput, &events.StdOutputEventRequest{
		TestId:    "test-1",
		Message:   "Console output",
		Timestamp: timestamppb.Now(),
	}); err != nil {
		t.Errorf("Failed to publish StdOutput: %v", err)
	}

	// 4. StepBegin
	if err := pub.Publish(ctx, publisher.EventTypeStepBegin, &events.StepBeginEventRequest{
		Step: &entities.StepRun{
			TestCaseRunId: "test-1",
		},
	}); err != nil {
		t.Errorf("Failed to publish StepBegin: %v", err)
	}

	// 5. StepEnd
	if err := pub.Publish(ctx, publisher.EventTypeStepEnd, &events.StepEndEventRequest{
		Step: &entities.StepRun{
			TestCaseRunId: "test-1",
			Status:        common.TestStatus_PASSED,
		},
	}); err != nil {
		t.Errorf("Failed to publish StepEnd: %v", err)
	}

	// 6. TestFailure
	if err := pub.Publish(ctx, publisher.EventTypeTestFailure, &events.TestFailureEventRequest{
		TestId:         "test-1",
		FailureMessage: "Test failed",
		StackTrace:     "at line 1",
		Timestamp:      timestamppb.Now(),
	}); err != nil {
		t.Errorf("Failed to publish TestFailure: %v", err)
	}

	// 7. TestError
	if err := pub.Publish(ctx, publisher.EventTypeTestError, &events.TestErrorEventRequest{
		TestId:       "test-1",
		ErrorMessage: "Test error",
		StackTrace:   "at line 2",
		Timestamp:    timestamppb.Now(),
	}); err != nil {
		t.Errorf("Failed to publish TestError: %v", err)
	}

	// 8. StdError
	if err := pub.Publish(ctx, publisher.EventTypeStdError, &events.StdErrorEventRequest{
		TestId:    "test-1",
		Message:   "Error output",
		Timestamp: timestamppb.Now(),
	}); err != nil {
		t.Errorf("Failed to publish StdError: %v", err)
	}

	// 9. TestEnd
	if err := pub.Publish(ctx, publisher.EventTypeTestEnd, &events.TestEndEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:     "test-1",
			RunId:  "run-1",
			Status: common.TestStatus_FAILED,
		},
	}); err != nil {
		t.Errorf("Failed to publish TestEnd: %v", err)
	}

	// 10. SuiteEnd
	if err := pub.Publish(ctx, publisher.EventTypeSuiteEnd, &events.SuiteEndEventRequest{
		Suite: &entities.TestSuiteRun{
			Id:     "suite-1",
			Status: common.TestStatus_FAILED,
		},
	}); err != nil {
		t.Errorf("Failed to publish SuiteEnd: %v", err)
	}

	// 11. Heartbeat
	if err := pub.Publish(ctx, publisher.EventTypeHeartbeat, &events.HeartbeatEventRequest{
		SourceId:  "test-runner",
		Timestamp: timestamppb.Now(),
	}); err != nil {
		t.Errorf("Failed to publish Heartbeat: %v", err)
	}

	t.Log("All events published, fetching from NATS...")

	// Wait a bit for messages to propagate
	time.Sleep(500 * time.Millisecond)

	// Fetch and verify messages
	timeout := time.After(5 * time.Second)
	receivedCount := 0

fetchLoop:
	for receivedCount < len(expectedEvents) {
		select {
		case <-timeout:
			t.Logf("Timeout waiting for all events. Received %d/%d", receivedCount, len(expectedEvents))
			break fetchLoop
		default:
			msgs, err := cons.Fetch(10, jetstream.FetchMaxWait(1*time.Second))
			if err != nil {
				if err == nats.ErrTimeout || err == jetstream.ErrNoMessages {
					continue
				}
				t.Errorf("Failed to fetch messages: %v", err)
				break fetchLoop
			}

			for msg := range msgs.Messages() {
				var event publisher.Event
				if err := json.Unmarshal(msg.Data(), &event); err != nil {
					t.Errorf("Failed to unmarshal event: %v", err)
					msg.Nak()
					continue
				}

				t.Logf("Received event type: %s", event.Type)

				if _, expected := expectedEvents[event.Type]; expected {
					if !expectedEvents[event.Type] {
						expectedEvents[event.Type] = true
						receivedCount++
						t.Logf("✓ Verified event type: %s", event.Type)
					}
				} else {
					t.Logf("Received unexpected event type: %s", event.Type)
				}

				msg.Ack()
			}
		}
	}

	// Check that we received all expected events
	for eventType, received := range expectedEvents {
		if !received {
			t.Errorf("Missing event type: %s", eventType)
		}
	}

	if receivedCount == len(expectedEvents) {
		t.Logf("✓ Successfully verified all %d event types!", receivedCount)
	} else {
		t.Errorf("Expected %d event types, received %d", len(expectedEvents), receivedCount)
	}
}
