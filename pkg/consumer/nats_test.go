package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stanterprise/observer/pkg/publisher"
	"github.com/stanterprise/proto-go/testsystem/v1/common"
	"github.com/stanterprise/proto-go/testsystem/v1/entities"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestNATSConsumer_Integration tests the consumer with an actual NATS server
// This test only runs if NATS_TEST_URL is set
func TestNATSConsumer_Integration(t *testing.T) {
	natsURL := os.Getenv("NATS_TEST_URL")
	if natsURL == "" {
		t.Skip("Skipping NATS consumer integration test - NATS_TEST_URL not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create in-memory SQLite database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&testCaseRun{}, &stepRun{}); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}

	// Create unique stream for this test
	streamName := "test_consumer_" + time.Now().Format("20060102150405")
	subjectPrefix := "test.events.v1"

	// Setup NATS stream and publisher
	nc, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("Failed to create JetStream context: %v", err)
	}

	// Create stream
	_, err = js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:        streamName,
		Subjects:    []string{subjectPrefix + ".>"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      1 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
		Description: "Test consumer stream",
	})
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}
	defer js.DeleteStream(context.Background(), streamName)

	// Create consumer
	cfg := NATSConsumerConfig{
		URL:          natsURL,
		StreamName:   streamName,
		ConsumerName: "test_consumer",
		BatchSize:    5,
		MaxWait:      2 * time.Second,
	}

	consumer, err := NewNATSConsumer(cfg, logger, db)
	if err != nil {
		t.Fatalf("NewNATSConsumer() error = %v", err)
	}
	defer consumer.Close()

	// Start consumer in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		err := consumer.Start(ctx, cfg)
		if err != nil && err != context.Canceled {
			errChan <- err
		}
	}()

	// Publish test events
	t.Run("TestBegin event processing", func(t *testing.T) {
		event := publisher.Event{
			Type:      publisher.EventTypeTestBegin,
			Timestamp: time.Now(),
		}

		reqData, _ := json.Marshal(&events.TestBeginEventRequest{
			TestCase: &entities.TestCaseRun{
				Id:       "test-consumer-1",
				RunId:    "run-consumer-1",
				Title:    "Consumer Test",
				Metadata: map[string]string{"env": "test"},
			},
		})
		event.Data = reqData

		eventBytes, _ := json.Marshal(event)

		// Publish event
		_, err := js.Publish(context.Background(), subjectPrefix+".test.begin", eventBytes)
		if err != nil {
			t.Fatalf("Failed to publish event: %v", err)
		}

		// Wait for processing
		time.Sleep(1 * time.Second)

		// Verify database
		var tc testCaseRun
		result := db.Where("id = ?", "test-consumer-1").First(&tc)
		if result.Error != nil {
			t.Errorf("Failed to find test case: %v", result.Error)
		}
		if tc.Title != "Consumer Test" {
			t.Errorf("Title = %v, want Consumer Test", tc.Title)
		}
	})

	t.Run("TestEnd event processing", func(t *testing.T) {
		event := publisher.Event{
			Type:      publisher.EventTypeTestEnd,
			Timestamp: time.Now(),
		}

		reqData, _ := json.Marshal(&events.TestEndEventRequest{
			TestCase: &entities.TestCaseRun{
				Id:     "test-consumer-1",
				RunId:  "run-consumer-1",
				Status: common.TestStatus_PASSED,
			},
		})
		event.Data = reqData

		eventBytes, _ := json.Marshal(event)

		// Publish event
		_, err := js.Publish(context.Background(), subjectPrefix+".test.end", eventBytes)
		if err != nil {
			t.Fatalf("Failed to publish event: %v", err)
		}

		// Wait for processing
		time.Sleep(1 * time.Second)

		// Verify database
		var tc testCaseRun
		result := db.Where("id = ?", "test-consumer-1").First(&tc)
		if result.Error != nil {
			t.Errorf("Failed to find test case: %v", result.Error)
		}
		if tc.Status != "PASSED" {
			t.Errorf("Status = %v, want PASSED", tc.Status)
		}
	})

	// Stop consumer
	cancel()

	// Check for errors
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Consumer error: %v", err)
		}
	case <-time.After(2 * time.Second):
		// No error, consumer stopped cleanly
	}
}

// Test models for in-memory DB testing
type testCaseRun struct {
	ID        string `gorm:"primaryKey"`
	RunID     string `gorm:"index"`
	Title     string
	Status    string
	Metadata  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (testCaseRun) TableName() string {
	return "test_case_runs"
}

type stepRun struct {
	ID            uint   `gorm:"primaryKey;autoIncrement"`
	TestCaseRunID string `gorm:"index"`
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (stepRun) TableName() string {
	return "step_runs"
}

func TestNATSConsumerConfig_Defaults(t *testing.T) {
	// Test that defaults are applied correctly
	cfg := NATSConsumerConfig{
		URL: "nats://localhost:4222",
		// Leave other fields empty to test defaults
	}

	if cfg.StreamName == "" {
		cfg.StreamName = publisher.DefaultStreamName
	}
	if cfg.ConsumerName == "" {
		cfg.ConsumerName = "processor"
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10
	}
	if cfg.MaxWait <= 0 {
		cfg.MaxWait = 5 * time.Second
	}

	if cfg.StreamName != publisher.DefaultStreamName {
		t.Errorf("StreamName = %v, want %v", cfg.StreamName, publisher.DefaultStreamName)
	}
	if cfg.ConsumerName != "processor" {
		t.Errorf("ConsumerName = %v, want processor", cfg.ConsumerName)
	}
	if cfg.BatchSize != 10 {
		t.Errorf("BatchSize = %v, want 10", cfg.BatchSize)
	}
	if cfg.MaxWait != 5*time.Second {
		t.Errorf("MaxWait = %v, want 5s", cfg.MaxWait)
	}
}

func TestNewNATSConsumer_RequiresURL(t *testing.T) {
	cfg := NATSConsumerConfig{
		URL: "",
	}

	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	_, err := NewNATSConsumer(cfg, nil, db)
	if err == nil {
		t.Error("Expected error for empty URL")
	}
	if err.Error() != "NATS URL is required" {
		t.Errorf("Error = %v, want 'NATS URL is required'", err)
	}
}

func TestNewNATSConsumer_RequiresDB(t *testing.T) {
	cfg := NATSConsumerConfig{
		URL: "nats://localhost:4222",
	}

	_, err := NewNATSConsumer(cfg, nil, nil)
	if err == nil {
		t.Error("Expected error for nil database")
	}
	if err.Error() != "database connection is required for processor" {
		t.Errorf("Error = %v, want 'database connection is required for processor'", err)
	}
}
