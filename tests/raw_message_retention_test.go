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

// TestRawMessageRetention validates that the consumer stores raw messages in the
// raw_messages collection when retention is enabled and leaves the collection empty
// when retention is disabled.
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

	// Publish a test.begin event
	runID := "run-retention-test-1"
	suiteID := "suite-retention-1"

	// Use the publisher to emit events that the consumer will process
	if err := pub.Publish(ctx, publisher.EventTypeSuiteBegin, map[string]interface{}{
		"suite": map[string]interface{}{
			"id":     suiteID,
			"run_id": runID,
			"name":   "Retention Test Suite",
			"status": "RUNNING",
		},
	}); err != nil {
		t.Fatalf("Publish suite begin: %v", err)
	}

	cfg := consumer.MongoNATSConsumerConfig{
		URL:            natsURL,
		StreamName:     streamName,
		ConsumerName:   "retention-test-consumer",
		BatchSize:      10,
		MaxWait:        1 * time.Second,
		RetainMessages: true,
	}

	natsConsumer, err := consumer.NewMongoNATSConsumer(cfg, logger, repo, rawMsgRepo)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer natsConsumer.Close()

	// Drain one batch of messages
	consumerCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	go func() {
		_ = natsConsumer.Start(consumerCtx, cfg)
	}()

	// Wait for the consumer to process the published event
	time.Sleep(3 * time.Second)
	cancel()

	// Verify that a raw message document was stored
	count, err := rawMsgCol.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("CountDocuments() error = %v", err)
	}
	if count == 0 {
		t.Fatal("Expected at least one raw message document when retention is enabled")
	}

	// Verify structure of the stored document
	var rawDoc m.RawMessageDocument
	if err := rawMsgCol.FindOne(ctx, bson.M{}).Decode(&rawDoc); err != nil {
		t.Fatalf("FindOne() error = %v", err)
	}
	if rawDoc.Subject == "" {
		t.Error("RawMessageDocument.Subject should not be empty")
	}
	if rawDoc.EventType == "" {
		t.Error("RawMessageDocument.EventType should not be empty")
	}
	if len(rawDoc.Payload) == 0 {
		t.Error("RawMessageDocument.Payload should not be empty")
	}
	if rawDoc.ReceivedAt.IsZero() {
		t.Error("RawMessageDocument.ReceivedAt should not be zero")
	}
	if rawDoc.Stream != streamName {
		t.Errorf("RawMessageDocument.Stream = %q, want %q", rawDoc.Stream, streamName)
	}

	// Payload should be valid JSON containing an event envelope
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(rawDoc.Payload, &envelope); err != nil {
		t.Errorf("Payload is not valid JSON: %v", err)
	}
	if _, ok := envelope["type"]; !ok {
		t.Error("Payload JSON should contain 'type' field")
	}

	t.Logf("✅ Raw message retention: %d document(s) stored in raw_messages collection", count)

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
	natsConsumer, err := consumer.NewMongoNATSConsumer(cfg, logger, repo, nil)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer natsConsumer.Close()

	consumerCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	go func() {
		_ = natsConsumer.Start(consumerCtx, cfg)
	}()

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
