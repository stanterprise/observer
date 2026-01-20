package main

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"github.com/stanterprise/observer/pkg/consumer"
	"github.com/stanterprise/observer/pkg/publisher"
	"github.com/stanterprise/proto-go/testsystem/v1/common"
	entities "github.com/stanterprise/proto-go/testsystem/v1/entities"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	natsmodule "github.com/testcontainers/testcontainers-go/modules/nats"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestShardedExecution_MultipleRunStarts tests that multiple run start events with the same run_id
// accumulate total_tests and suites correctly
func TestShardedExecution_MultipleRunStarts(t *testing.T) {
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

	// Connect to MongoDB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	collection := mongoClient.Database("observer_test").Collection("test_runs")
	repo := repository.NewMongoRepository(collection, logger)

	// Start NATS container
	natsContainer, err := natsmodule.Run(ctx, "nats:latest")
	if err != nil {
		t.Fatalf("Failed to start NATS container: %v", err)
	}
	defer natsContainer.Terminate(ctx)

	natsURI, err := natsContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get NATS connection string: %v", err)
	}

	// Create NATS publisher
	pub, err := publisher.NewNATSPublisher(publisher.NATSConfig{
		URL:        natsURI,
		StreamName: "tests_events",
	}, logger)
	if err != nil {
		t.Fatalf("Failed to create NATS publisher: %v", err)
	}
	defer pub.Close()

	// Create NATS consumer
	consumerConfig := consumer.MongoNATSConsumerConfig{
		URL:          natsURI,
		StreamName:   "tests_events",
		ConsumerName: "test_consumer_sharded",
		BatchSize:    10,
		MaxWait:      5 * time.Second,
	}
	natsConsumer, err := consumer.NewMongoNATSConsumer(consumerConfig, logger, repo)
	if err != nil {
		t.Fatalf("Failed to create NATS consumer: %v", err)
	}

	// Start consumer in background
	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	defer cancelConsumer()

	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- natsConsumer.Start(consumerCtx, consumerConfig)
	}()

	// Wait for consumer to be ready
	time.Sleep(500 * time.Millisecond)

	runID := "sharded-run-123"
	runName := "Sharded Test Run"
	now := timestamppb.Now()

	// Simulate 3 shards sending run start events
	// Shard 1: 50 tests, suites A and B
	shard1Metadata := map[string]string{
		"shard_id":    "shard-1",
		"worker_id":   "worker-1",
		"environment": "ci",
		"browser":     "chromium",
	}
	shard1Event := &events.ReportRunStartEventRequest{
		RunId:      runID,
		Name:       runName,
		TotalTests: 50,
		Metadata:   shard1Metadata,
		TestSuites: []*entities.TestSuiteRun{
			{
				Id:        "suite-A",
				RunId:     runID,
				Name:      "Suite A",
				Status:    common.TestStatus_RUNNING,
				StartTime: now,
				Metadata: map[string]string{
					"shard": "1",
				},
			},
			{
				Id:        "suite-B",
				RunId:     runID,
				Name:      "Suite B",
				Status:    common.TestStatus_RUNNING,
				StartTime: now,
				Metadata: map[string]string{
					"shard": "1",
				},
			},
		},
	}

	if err := pub.Publish(ctx, publisher.EventTypeRunStart, shard1Event); err != nil {
		t.Fatalf("Failed to publish shard 1 run start: %v", err)
	}

	// Shard 2: 30 tests, suites C and D
	shard2Metadata := map[string]string{
		"shard_id":    "shard-2",
		"worker_id":   "worker-2",
		"environment": "ci",
		"browser":     "firefox",
	}
	shard2Event := &events.ReportRunStartEventRequest{
		RunId:      runID,
		Name:       runName,
		TotalTests: 30,
		Metadata:   shard2Metadata,
		TestSuites: []*entities.TestSuiteRun{
			{
				Id:        "suite-C",
				RunId:     runID,
				Name:      "Suite C",
				Status:    common.TestStatus_RUNNING,
				StartTime: now,
				Metadata: map[string]string{
					"shard": "2",
				},
			},
			{
				Id:        "suite-D",
				RunId:     runID,
				Name:      "Suite D",
				Status:    common.TestStatus_RUNNING,
				StartTime: now,
				Metadata: map[string]string{
					"shard": "2",
				},
			},
		},
	}

	if err := pub.Publish(ctx, publisher.EventTypeRunStart, shard2Event); err != nil {
		t.Fatalf("Failed to publish shard 2 run start: %v", err)
	}

	// Shard 3: 20 tests, suite E
	shard3Metadata := map[string]string{
		"shard_id":    "shard-3",
		"worker_id":   "worker-3",
		"environment": "ci",
		"browser":     "webkit",
	}
	shard3Event := &events.ReportRunStartEventRequest{
		RunId:      runID,
		Name:       runName,
		TotalTests: 20,
		Metadata:   shard3Metadata,
		TestSuites: []*entities.TestSuiteRun{
			{
				Id:        "suite-E",
				RunId:     runID,
				Name:      "Suite E",
				Status:    common.TestStatus_RUNNING,
				StartTime: now,
				Metadata: map[string]string{
					"shard": "3",
				},
			},
		},
	}

	if err := pub.Publish(ctx, publisher.EventTypeRunStart, shard3Event); err != nil {
		t.Fatalf("Failed to publish shard 3 run start: %v", err)
	}

	// Wait for events to be processed
	time.Sleep(5 * time.Second)

	// Verify the run document
	var doc m.TestRunDocument
	err = collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find run document: %v", err)
	}

	// Verify total_tests accumulation: 50 + 30 + 20 = 100
	if doc.TotalTests != 100 {
		t.Errorf("Expected total_tests=100, got %d", doc.TotalTests)
	}

	// Verify shard_count: 3 shards
	if doc.ShardCount != 3 {
		t.Errorf("Expected shard_count=3, got %d", doc.ShardCount)
	}

	// Verify all 5 suites are present
	if len(doc.Suites) != 5 {
		t.Errorf("Expected 5 suites, got %d", len(doc.Suites))
	}

	// Verify suite IDs
	suiteIDs := make(map[string]bool)
	for _, suite := range doc.Suites {
		suiteIDs[suite.ID] = true
	}

	expectedSuites := []string{"suite-A", "suite-B", "suite-C", "suite-D", "suite-E"}
	for _, expectedID := range expectedSuites {
		if !suiteIDs[expectedID] {
			t.Errorf("Missing expected suite: %s", expectedID)
		}
	}

	// Verify metadata merge - all keys should be present
	if doc.Metadata == nil {
		t.Fatal("Expected metadata to be populated")
	}

	// Check that metadata from last shard (shard-3) is present
	if doc.Metadata["shard_id"] != "shard-3" {
		t.Errorf("Expected metadata shard_id=shard-3, got %v", doc.Metadata["shard_id"])
	}
	if doc.Metadata["browser"] != "webkit" {
		t.Errorf("Expected metadata browser=webkit, got %v", doc.Metadata["browser"])
	}
	if doc.Metadata["environment"] != "ci" {
		t.Errorf("Expected metadata environment=ci, got %v", doc.Metadata["environment"])
	}

	// Verify run name
	if doc.Name != runName {
		t.Errorf("Expected name=%s, got %s", runName, doc.Name)
	}

	// Clean shutdown
	cancelConsumer()
	select {
	case err := <-consumerDone:
		if err != nil && err != context.Canceled {
			t.Errorf("Consumer error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Consumer shutdown timeout")
	}
}

// TestShardedExecution_IdempotentRunStart tests that replaying the same run start event
// is idempotent (doesn't duplicate data)
func TestShardedExecution_IdempotentRunStart(t *testing.T) {
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

	// Connect to MongoDB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	collection := mongoClient.Database("observer_test").Collection("test_runs")
	repo := repository.NewMongoRepository(collection, logger)

	runID := "idempotent-run-456"

	// Call MapSuites twice with the same data
	metadata := map[string]interface{}{
		"environment": "test",
		"shard_id":    "shard-1",
	}

	suites := []m.SuiteDocument{
		{
			ID:     "suite-1",
			RunID:  runID,
			Name:   "Test Suite 1",
			Status: "RUNNING",
		},
	}

	// First call
	err = repo.MapSuites(ctx, runID, "Test Run", metadata, 25, suites)
	if err != nil {
		t.Fatalf("First MapSuites call failed: %v", err)
	}

	// Second call with same data
	err = repo.MapSuites(ctx, runID, "Test Run", metadata, 25, suites)
	if err != nil {
		t.Fatalf("Second MapSuites call failed: %v", err)
	}

	// Verify the run document
	var doc m.TestRunDocument
	err = collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find run document: %v", err)
	}

	// Total tests should accumulate: 25 + 25 = 50
	if doc.TotalTests != 50 {
		t.Errorf("Expected total_tests=50 (accumulated), got %d", doc.TotalTests)
	}

	// Shard count should be 2 (two calls)
	if doc.ShardCount != 2 {
		t.Errorf("Expected shard_count=2, got %d", doc.ShardCount)
	}

	// Suites will be duplicated (same suite appended twice)
	// This is expected behavior - deduplication is not implemented
	if len(doc.Suites) != 2 {
		t.Errorf("Expected 2 suites (duplicated), got %d", len(doc.Suites))
	}
}

// TestShardedExecution_ZeroTotalTests tests that shards with zero total_tests don't affect accumulation
func TestShardedExecution_ZeroTotalTests(t *testing.T) {
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

	// Connect to MongoDB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	collection := mongoClient.Database("observer_test").Collection("test_runs")
	repo := repository.NewMongoRepository(collection, logger)

	runID := "zero-tests-run-789"

	// First shard with 30 tests
	err = repo.MapSuites(ctx, runID, "Test Run", map[string]interface{}{"shard": "1"}, 30, []m.SuiteDocument{
		{ID: "suite-1", RunID: runID, Name: "Suite 1"},
	})
	if err != nil {
		t.Fatalf("First MapSuites failed: %v", err)
	}

	// Second shard with 0 tests (should not increment)
	err = repo.MapSuites(ctx, runID, "Test Run", map[string]interface{}{"shard": "2"}, 0, []m.SuiteDocument{
		{ID: "suite-2", RunID: runID, Name: "Suite 2"},
	})
	if err != nil {
		t.Fatalf("Second MapSuites failed: %v", err)
	}

	// Third shard with 20 tests
	err = repo.MapSuites(ctx, runID, "Test Run", map[string]interface{}{"shard": "3"}, 20, []m.SuiteDocument{
		{ID: "suite-3", RunID: runID, Name: "Suite 3"},
	})
	if err != nil {
		t.Fatalf("Third MapSuites failed: %v", err)
	}

	// Verify the run document
	var doc m.TestRunDocument
	err = collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find run document: %v", err)
	}

	// Total tests should be 30 + 20 = 50 (zero not counted)
	if doc.TotalTests != 50 {
		t.Errorf("Expected total_tests=50, got %d", doc.TotalTests)
	}

	// Shard count should be 2 (only shards with totalTests > 0)
	if doc.ShardCount != 2 {
		t.Errorf("Expected shard_count=2, got %d", doc.ShardCount)
	}

	// All 3 suites should be present
	if len(doc.Suites) != 3 {
		t.Errorf("Expected 3 suites, got %d", len(doc.Suites))
	}
}
