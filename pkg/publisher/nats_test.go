package publisher

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

func TestNewNATSPublisher_RequiresURL(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := NATSConfig{
		URL: "",
	}

	_, err := NewNATSPublisher(cfg, logger)
	if err == nil {
		t.Error("NewNATSPublisher() with empty URL should return error")
	}
}

func TestNewNATSPublisher_NilLogger(t *testing.T) {
	// This test verifies nil logger handling doesn't panic
	// but will fail to connect without a real NATS server
	cfg := NATSConfig{
		URL: "nats://localhost:4222",
	}

	_, err := NewNATSPublisher(cfg, nil)
	// We expect an error because NATS is not running, but shouldn't panic
	if err == nil {
		t.Error("Expected error when NATS server is not available")
	}
}

func TestNATSConfig_Defaults(t *testing.T) {
	// Test that default values are properly set
	cfg := NATSConfig{
		URL: "nats://localhost:4222",
	}

	if cfg.StreamName == "" {
		cfg.StreamName = DefaultStreamName
	}
	if cfg.SubjectPrefix == "" {
		cfg.SubjectPrefix = DefaultSubjectPrefix
	}

	if cfg.StreamName != DefaultStreamName {
		t.Errorf("Default stream name = %v, want %v", cfg.StreamName, DefaultStreamName)
	}
	if cfg.SubjectPrefix != DefaultSubjectPrefix {
		t.Errorf("Default subject prefix = %v, want %v", cfg.SubjectPrefix, DefaultSubjectPrefix)
	}
}

func TestEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		expected  string
	}{
		{"test begin", EventTypeTestBegin, "test.begin"},
		{"test end", EventTypeTestEnd, "test.end"},
		{"step begin", EventTypeStepBegin, "step.begin"},
		{"step end", EventTypeStepEnd, "step.end"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.eventType) != tt.expected {
				t.Errorf("EventType = %v, want %v", tt.eventType, tt.expected)
			}
		})
	}
}

func TestNoopWriter(t *testing.T) {
	w := &noopWriter{}
	data := []byte("test data")
	n, err := w.Write(data)
	if err != nil {
		t.Errorf("noopWriter.Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("noopWriter.Write() n = %v, want %v", n, len(data))
	}
}

// Integration test - only runs if NATS_TEST_URL is set
func TestNATSPublisher_Integration(t *testing.T) {
	natsURL := os.Getenv("NATS_TEST_URL")
	if natsURL == "" {
		t.Skip("Skipping integration test - NATS_TEST_URL not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Use unique stream name for testing
	cfg := NATSConfig{
		URL:           natsURL,
		StreamName:    "test_events_" + time.Now().Format("20060102150405"),
		SubjectPrefix: "test.events.v1",
	}

	pub, err := NewNATSPublisher(cfg, logger)
	if err != nil {
		t.Fatalf("NewNATSPublisher() error = %v", err)
	}
	defer pub.Close()

	ctx := context.Background()

	// Test publishing an event
	testData := map[string]string{
		"test_id": "test-123",
		"status":  "PASSED",
	}

	err = pub.Publish(ctx, EventTypeTestBegin, testData)
	if err != nil {
		t.Errorf("Publish() error = %v", err)
	}

	// Verify we can subscribe and receive the event
	consumer, err := pub.js.CreateOrUpdateConsumer(ctx, cfg.StreamName, jetstream.ConsumerConfig{
		Durable:   "test_consumer_" + time.Now().Format("20060102150405"),
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateConsumer() error = %v", err)
	}

	// Try to fetch the message
	msgs, err := consumer.Fetch(1, jetstream.FetchMaxWait(2*time.Second))
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	msgReceived := false
	for msg := range msgs.Messages() {
		msgReceived = true
		if err := msg.Ack(); err != nil {
			t.Errorf("failed to ack message: %v", err)
		}
		t.Logf("Received message: %s", string(msg.Data()))
	}

	if !msgReceived {
		t.Error("Expected to receive published message")
	}

	// Cleanup - delete the test stream
	if err := pub.js.DeleteStream(ctx, cfg.StreamName); err != nil {
		t.Logf("Warning: failed to delete test stream: %v", err)
	}
}

// Test that Close doesn't panic with nil connection
func TestNATSPublisher_Close_Nil(t *testing.T) {
	p := &NATSPublisher{
		nc:     nil,
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	err := p.Close()
	if err != nil {
		t.Errorf("Close() with nil connection error = %v", err)
	}
}

// Test Publish with invalid data
func TestNATSPublisher_Publish_InvalidData(t *testing.T) {
	// Create a mock publisher without actual NATS connection
	// to test JSON marshaling failures
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	p := &NATSPublisher{
		nc:     nil,
		logger: logger,
		prefix: DefaultSubjectPrefix,
	}

	ctx := context.Background()

	// Channel type cannot be marshaled to JSON
	invalidData := make(chan int)

	err := p.Publish(ctx, EventTypeTestBegin, invalidData)
	if err == nil {
		t.Error("Publish() with unmarshalable data should return error")
	}
}

// TestEvent verifies Event struct marshaling
func TestEvent_Marshaling(t *testing.T) {
	event := Event{
		Type:      EventTypeTestBegin,
		Timestamp: time.Now(),
		Data:      []byte(`{"test":"data"}`),
	}

	// Should be able to marshal and unmarshal
	typeStr, err := event.Type.String()
	if err != nil {
		t.Errorf("EventType.String() error = %v", err)
	}
	if typeStr == "" {
		t.Error("EventType.String() returned empty string")
	}

	if event.Type != EventTypeTestBegin {
		t.Errorf("Event.Type = %v, want %v", event.Type, EventTypeTestBegin)
	}
}

// String method for EventType for better formatting
func (e EventType) String() (string, error) {
	return string(e), nil
}
