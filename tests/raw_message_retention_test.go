package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"github.com/stanterprise/observer/pkg/consumer"
	"github.com/stanterprise/observer/pkg/publisher"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	natsmodule "github.com/testcontainers/testcontainers-go/modules/nats"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestRawMessageRetention validates that the consumer stores all messages for a
// test run in a single document identified by the run_id when retention is enabled.
func TestRawMessageRetention(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Start MongoDB container
	mongoContainer, err := mongodb.RunContainer(ctx, testcontainers.WithImage("mongo:7.0"))
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %v", err)
	}
	defer mongoContainer.Terminate(ctx)

	mongoURI, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get MongoDB connection string: %v", err)
	}

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	db := mongoClient.Database("observer_retention_test")
	testRunsCol := db.Collection("test_runs")
	rawMsgCol := db.Collection("raw_messages")

	repo := repository.NewMongoRepository(testRunsCol, logger)
	rawMsgRepo := repository.NewRawMessageRepository(rawMsgCol, logger)

	// Start NATS container
	natsContainer, err := natsmodule.Run(ctx, "nats:latest")
	if err != nil {
		t.Fatalf("Failed to start NATS container: %v", err)
	}
	defer natsContainer.Terminate(ctx)

	natsURL, err := natsContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get NATS connection string: %v", err)
	}

	streamName := "retention_test_events_" + time.Now().Format("20060102150405")
	subjectPrefix := "retention.test.events"

	// Create publisher
	var pub *publisher.NATSPublisher
	for i := 0; i < 5; i++ {
		pub, err = publisher.NewNATSPublisher(publisher.NATSConfig{
			URL:           natsURL,
			StreamName:    streamName,
			SubjectPrefix: subjectPrefix,
		}, logger)
		if err == nil {
			break
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}
	defer pub.Close()

	// Publish several events – all for the same run_id.
	runID := "run-retention-test-1"
	suiteID := "suite-retention-1"

	events := []struct {
		typ  publisher.EventType
		data interface{}
	}{
		{
			publisher.EventTypeSuiteBegin, map[string]interface{}{
				"suite": map[string]interface{}{
					"id": suiteID, "run_id": runID,
					"name": "Retention Test Suite", "status": "RUNNING",
				},
			},
		},
		{
			publisher.EventTypeTestBegin, map[string]interface{}{
				"test_case": map[string]interface{}{
					"id": "test-1", "run_id": runID,
					"test_suite_id": suiteID, "name": "test one", "status": "RUNNING",
				},
			},
		},
		{
			publisher.EventTypeTestEnd, map[string]interface{}{
				"test_case": map[string]interface{}{
					"id": "test-1", "run_id": runID,
					"test_suite_id": suiteID, "status": "PASSED",
				},
			},
		},
		{
			publisher.EventTypeSuiteEnd, map[string]interface{}{
				"suite": map[string]interface{}{
					"id": suiteID, "run_id": runID, "status": "PASSED",
				},
			},
		},
	}

	for _, ev := range events {
		if err := pub.Publish(ctx, ev.typ, ev.data); err != nil {
			t.Fatalf("Publish %s: %v", ev.typ, err)
		}
	}

	cfg := consumer.MongoNATSConsumerConfig{
		URL:            natsURL,
		StreamName:     streamName,
		ConsumerName:   "retention-test-consumer",
		BatchSize:      10,
		MaxWait:        1 * time.Second,
		RetainMessages: true,
	}

	natsConsumer, err := consumer.NewNATSConsumer(cfg, logger, repo, rawMsgRepo)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer natsConsumer.Close()

	// Run the consumer until messages are drained.
	consumerCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	go func() { _ = natsConsumer.Start(consumerCtx, cfg) }()

	// Wait for the consumer to process all published events.
	time.Sleep(5 * time.Second)
	cancel()

	// There should be exactly ONE document in raw_messages (all events share the same run_id).
	docCount, err := rawMsgCol.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("CountDocuments() error = %v", err)
	}
	if docCount != 1 {
		t.Errorf("Expected 1 raw_messages document (all events for run grouped), got %d", docCount)
	}

	// Verify the document structure.
	var runDoc m.RawMessagesRunDocument
	if err := rawMsgCol.FindOne(ctx, bson.M{"_id": runID}).Decode(&runDoc); err != nil {
		t.Fatalf("FindOne(_id=%q) error = %v", runID, err)
	}

	if runDoc.RunID != runID {
		t.Errorf("RunID = %q, want %q", runDoc.RunID, runID)
	}
	if len(runDoc.Messages) != len(events) {
		t.Errorf("Messages count = %d, want %d", len(runDoc.Messages), len(events))
	}

	for i, msg := range runDoc.Messages {
		if msg.Subject == "" {
			t.Errorf("Messages[%d].Subject should not be empty", i)
		}
		if msg.EventType == "" {
			t.Errorf("Messages[%d].EventType should not be empty", i)
		}
		if msg.Payload == nil {
			t.Errorf("Messages[%d].Payload should not be nil", i)
		}
		if msg.ReceivedAt.IsZero() {
			t.Errorf("Messages[%d].ReceivedAt should not be zero", i)
		}
		if msg.Stream != streamName {
			t.Errorf("Messages[%d].Stream = %q, want %q", i, msg.Stream, streamName)
		}
		// Payload is stored as a parsed JSON map.  Re-encode to JSON to verify
		// it contains the full event envelope (type, timestamp, data).
		payloadJSON, err := json.Marshal(msg.Payload)
		if err != nil {
			t.Errorf("Messages[%d].Payload cannot be JSON-marshaled: %v", i, err)
			continue
		}
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(payloadJSON, &envelope); err != nil {
			t.Errorf("Messages[%d].Payload is not a valid JSON object: %v", i, err)
			continue
		}
		if _, ok := envelope["type"]; !ok {
			t.Errorf("Messages[%d].Payload JSON should contain 'type' field", i)
		}
	}

	if runDoc.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if runDoc.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}

	t.Logf("✅ Raw message retention: %d message(s) stored in ONE document (run_id=%q)",
		len(runDoc.Messages), runID)

	// Cleanup NATS stream
	nc, err := nats.Connect(natsURL)
	if err == nil {
		js, err := jetstream.New(nc)
		if err == nil {
			js.DeleteStream(ctx, streamName)
		}
		nc.Close()
	}
}

// TestRawMessageRetention_Disabled verifies that no raw messages are stored when
// retention is disabled (rawMessageRepo is nil).
func TestRawMessageRetention_Disabled(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Start MongoDB container
	mongoContainer, err := mongodb.RunContainer(ctx, testcontainers.WithImage("mongo:7.0"))
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %v", err)
	}
	defer mongoContainer.Terminate(ctx)

	mongoURI, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get MongoDB connection string: %v", err)
	}

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	db := mongoClient.Database("observer_retention_disabled_test")
	testRunsCol := db.Collection("test_runs")
	rawMsgCol := db.Collection("raw_messages")

	repo := repository.NewMongoRepository(testRunsCol, logger)
	// No raw message repo – retention disabled

	// Start NATS container
	natsContainer, err := natsmodule.Run(ctx, "nats:latest")
	if err != nil {
		t.Fatalf("Failed to start NATS container: %v", err)
	}
	defer natsContainer.Terminate(ctx)

	natsURL, err := natsContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get NATS connection string: %v", err)
	}

	streamName := "retention_disabled_test_" + time.Now().Format("20060102150405")
	subjectPrefix := "retention.disabled.test"

	var pub *publisher.NATSPublisher
	for i := 0; i < 5; i++ {
		pub, err = publisher.NewNATSPublisher(publisher.NATSConfig{
			URL:           natsURL,
			StreamName:    streamName,
			SubjectPrefix: subjectPrefix,
		}, logger)
		if err == nil {
			break
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}
	defer pub.Close()

	if err := pub.Publish(ctx, publisher.EventTypeSuiteBegin, map[string]interface{}{
		"suite": map[string]interface{}{"id": "s1", "run_id": "r1", "name": "Suite", "status": "RUNNING"},
	}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	cfg := consumer.MongoNATSConsumerConfig{
		URL:            natsURL,
		StreamName:     streamName,
		ConsumerName:   "retention-disabled-consumer",
		BatchSize:      10,
		MaxWait:        1 * time.Second,
		RetainMessages: false,
	}

	// nil rawMessageRepo → retention disabled
	natsConsumer, err := consumer.NewNATSConsumer(cfg, logger, repo, nil)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer natsConsumer.Close()

	consumerCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	go func() { _ = natsConsumer.Start(consumerCtx, cfg) }()

	time.Sleep(3 * time.Second)
	cancel()

	// raw_messages collection should be empty
	count, err := rawMsgCol.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("CountDocuments() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 raw message documents when retention is disabled, got %d", count)
	}

	t.Logf("✅ Raw message retention disabled: 0 documents in raw_messages collection")

	// Cleanup
	nc, err := nats.Connect(natsURL)
	if err == nil {
		js, err := jetstream.New(nc)
		if err == nil {
			js.DeleteStream(ctx, streamName)
		}
		nc.Close()
	}
}

// TestRawMessageRetention_MultipleRuns verifies that events from different runs
// land in separate documents.
func TestRawMessageRetention_MultipleRuns(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	mongoContainer, err := mongodb.RunContainer(ctx, testcontainers.WithImage("mongo:7.0"))
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %v", err)
	}
	defer mongoContainer.Terminate(ctx)

	mongoURI, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get MongoDB connection string: %v", err)
	}

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	db := mongoClient.Database("observer_retention_multi_test")
	testRunsCol := db.Collection("test_runs")
	rawMsgCol := db.Collection("raw_messages")

	repo := repository.NewMongoRepository(testRunsCol, logger)
	rawMsgRepo := repository.NewRawMessageRepository(rawMsgCol, logger)

	natsContainer, err := natsmodule.Run(ctx, "nats:latest")
	if err != nil {
		t.Fatalf("Failed to start NATS container: %v", err)
	}
	defer natsContainer.Terminate(ctx)

	natsURL, err := natsContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get NATS connection string: %v", err)
	}

	streamName := "retention_multi_test_" + time.Now().Format("20060102150405")
	subjectPrefix := "retention.multi.test"

	var pub *publisher.NATSPublisher
	for i := 0; i < 5; i++ {
		pub, err = publisher.NewNATSPublisher(publisher.NATSConfig{
			URL:           natsURL,
			StreamName:    streamName,
			SubjectPrefix: subjectPrefix,
		}, logger)
		if err == nil {
			break
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}
	defer pub.Close()

	// Publish events for two different runs.
	runIDs := []string{"run-multi-A", "run-multi-B"}
	for _, runID := range runIDs {
		if err := pub.Publish(ctx, publisher.EventTypeSuiteBegin, map[string]interface{}{
			"suite": map[string]interface{}{
				"id": "suite-" + runID, "run_id": runID,
				"name": "Suite " + runID, "status": "RUNNING",
			},
		}); err != nil {
			t.Fatalf("Publish for run %s: %v", runID, err)
		}
	}

	cfg := consumer.MongoNATSConsumerConfig{
		URL:            natsURL,
		StreamName:     streamName,
		ConsumerName:   "retention-multi-consumer",
		BatchSize:      10,
		MaxWait:        1 * time.Second,
		RetainMessages: true,
	}

	natsConsumer, err := consumer.NewNATSConsumer(cfg, logger, repo, rawMsgRepo)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer natsConsumer.Close()

	consumerCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	go func() { _ = natsConsumer.Start(consumerCtx, cfg) }()

	time.Sleep(5 * time.Second)
	cancel()

	// Expect one document per run_id.
	docCount, err := rawMsgCol.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("CountDocuments() error = %v", err)
	}
	if int(docCount) != len(runIDs) {
		t.Errorf("document count = %d, want %d (one per run)", docCount, len(runIDs))
	}

	for _, runID := range runIDs {
		var runDoc m.RawMessagesRunDocument
		if err := rawMsgCol.FindOne(ctx, bson.M{"_id": runID}).Decode(&runDoc); err != nil {
			t.Errorf("FindOne(_id=%q) error = %v", runID, err)
			continue
		}
		if len(runDoc.Messages) == 0 {
			t.Errorf("run %q: expected at least one message", runID)
		}
	}

	t.Logf("✅ Multiple runs: %d documents in raw_messages (one per run)", docCount)

	nc, err := nats.Connect(natsURL)
	if err == nil {
		js, err := jetstream.New(nc)
		if err == nil {
			js.DeleteStream(ctx, streamName)
		}
		nc.Close()
	}
}
