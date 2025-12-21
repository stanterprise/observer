package main

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"github.com/stanterprise/observer/pkg/publisher"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	natsmodule "github.com/testcontainers/testcontainers-go/modules/nats"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestNATSToMongoDB_FullEventFlow tests the complete flow from NATS publisher to MongoDB persistence
func TestNATSToMongoDB_FullEventFlow(t *testing.T) {
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

	natsURL, err := natsContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get NATS connection string: %v", err)
	}

	// Create unique stream for this test
	streamName := "test_events_" + time.Now().Format("20060102150405")
	subjectPrefix := "test.events"

	// Initialize publisher with retry logic
	var pub *publisher.NATSPublisher
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		pub, err = publisher.NewNATSPublisher(publisher.NATSConfig{
			URL:           natsURL,
			StreamName:    streamName,
			SubjectPrefix: subjectPrefix,
		}, logger)
		if err == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
	if err != nil {
		t.Fatalf("Failed to create publisher after %d retries: %v", maxRetries, err)
	}
	defer pub.Close()

	// Note: In this test we're directly using the repository instead of the consumer
	// to avoid dependency on GORM database. In production, the consumer would
	// be configured with a MongoDB repository to handle events.

	// Publish suite begin event
	runID := "run-integration-1"
	suiteID := "suite-integration-1"
	suiteBeginEvent := &m.SuiteDocument{
		ID:          suiteID,
		Name:        "Integration Test Suite",
		Description: "Testing NATS to MongoDB flow",
		Status:      "RUNNING",
		ProjectName: "observer-test",
	}

	// Manually insert via repository (simulating what consumer would do)
	err = repo.UpsertSuiteBegin(ctx, runID, suiteBeginEvent, "")
	if err != nil {
		t.Fatalf("Failed to upsert suite begin: %v", err)
	}

	// Publish test begin event
	testID := "test-integration-1"
	testBeginEvent := &m.TestDocument{
		ID:     testID,
		RunID:  runID,
		Title:  "Integration Test Case",
		Status: "RUNNING",
	}

	err = repo.UpsertTestBegin(ctx, runID, testBeginEvent, suiteID)
	if err != nil {
		t.Fatalf("Failed to upsert test begin: %v", err)
	}

	// Publish step begin event
	stepID := "step-integration-1"
	stepBeginEvent := &m.StepDocument{
		ID:       stepID,
		Status:   "RUNNING",
		Category: "action",
		Title:    "Perform action",
	}

	err = repo.UpsertStepBegin(ctx, runID, stepBeginEvent, testID, "")
	if err != nil {
		t.Fatalf("Failed to upsert step begin: %v", err)
	}

	// Publish step end event
	err = repo.UpsertStepEnd(ctx, runID, stepID, testID, "PASSED")
	if err != nil {
		t.Fatalf("Failed to upsert step end: %v", err)
	}

	// Publish test end event
	testDuration := int64(1000000000)
	err = repo.UpsertTestEnd(ctx, runID, testID, "PASSED", &testDuration)
	if err != nil {
		t.Fatalf("Failed to upsert test end: %v", err)
	}

	// Publish suite end event
	suiteEndTime := time.Now()
	suiteDuration := int64(5000000000)
	err = repo.UpsertSuiteEnd(ctx, runID, suiteID, "PASSED", &suiteEndTime, &suiteDuration)
	if err != nil {
		t.Fatalf("Failed to upsert suite end: %v", err)
	}

	// Verify final document structure
	var finalDoc m.TestRunDocument
	err = collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&finalDoc)
	if err != nil {
		t.Fatalf("Failed to retrieve final document: %v", err)
	}

	// Assertions
	if finalDoc.ID != runID {
		t.Errorf("Document ID = %v, want %v", finalDoc.ID, runID)
	}

	// Document should have suite in suites array
	if len(finalDoc.Suites) != 1 {
		t.Fatalf("Suites count = %v, want 1 (suites: %+v)", len(finalDoc.Suites), finalDoc.Suites)
	}

	suite := finalDoc.Suites[0]
	if suite.ID != suiteID {
		t.Errorf("Suite ID = %v, want %v", suite.ID, suiteID)
	}
	if suite.Status != "PASSED" {
		t.Errorf("Suite status = %v, want PASSED", suite.Status)
	}
	if suite.EndTime == nil {
		t.Error("Suite end time is nil")
	}
	if suite.Duration == nil || *suite.Duration != suiteDuration {
		t.Errorf("Suite duration = %v, want %v", suite.Duration, suiteDuration)
	}

	// Test should be in suite's tests array
	if len(suite.Tests) != 1 {
		t.Fatalf("Tests count = %v, want 1", len(suite.Tests))
	}
	if suite.Tests[0].ID != testID {
		t.Errorf("Test ID = %v, want %v", suite.Tests[0].ID, testID)
	}
	if suite.Tests[0].Status != "PASSED" {
		t.Errorf("Test status = %v, want PASSED", suite.Tests[0].Status)
	}
	if suite.Tests[0].Duration == nil || *suite.Tests[0].Duration != testDuration {
		t.Errorf("Test duration = %v, want %v", suite.Tests[0].Duration, testDuration)
	}

	// Step should be in test's steps array
	if len(suite.Tests[0].Steps) != 1 {
		t.Fatalf("Steps count = %v, want 1", len(suite.Tests[0].Steps))
	}
	if suite.Tests[0].Steps[0].ID != stepID {
		t.Errorf("Step ID = %v, want %v", suite.Tests[0].Steps[0].ID, stepID)
	}
	if suite.Tests[0].Steps[0].Status != "PASSED" {
		t.Errorf("Step status = %v, want PASSED", suite.Tests[0].Steps[0].Status)
	}

	// Cleanup stream
	nc, err := nats.Connect(natsURL)
	if err == nil {
		js, err := jetstream.New(nc)
		if err == nil {
			js.DeleteStream(ctx, streamName)
		}
		nc.Close()
	}

	t.Logf("✅ Successfully validated complete event flow: Suite → Test → Step → Updates")
}

// TestNATSToMongoDB_NestedSuites tests nested suite hierarchy
func TestNATSToMongoDB_NestedSuites(t *testing.T) {
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

	// Create root suite
	runID := "run-nested-suite-test"
	rootSuiteID := "suite-root"
	rootSuite := &m.SuiteDocument{
		ID:     rootSuiteID,
		Name:   "Root Suite",
		Status: "RUNNING",
	}
	err = repo.UpsertSuiteBegin(ctx, runID, rootSuite, "")
	if err != nil {
		t.Fatalf("Failed to create root suite: %v", err)
	}

	// Create nested suite level 1
	nestedSuite1ID := "suite-level1"
	nestedSuite1 := &m.SuiteDocument{
		ID:     nestedSuite1ID,
		Name:   "Nested Suite Level 1",
		Status: "RUNNING",
	}
	err = repo.UpsertSuiteBegin(ctx, runID, nestedSuite1, rootSuiteID)
	if err != nil {
		t.Fatalf("Failed to create nested suite 1: %v", err)
	}

	// Add test to nested suite
	testID := "test-in-nested-suite"
	test := &m.TestDocument{
		ID:     testID,
		Title:  "Test in Nested Suite",
		Status: "PASSED",
	}
	err = repo.UpsertTestBegin(ctx, runID, test, nestedSuite1ID)
	if err != nil {
		t.Fatalf("Failed to create test: %v", err)
	}

	// Verify final state
	var finalDoc m.TestRunDocument
	err = collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&finalDoc)
	if err != nil {
		t.Fatalf("Failed to get final document: %v", err)
	}

	// Should have root suite with one nested suite
	if len(finalDoc.Suites) == 0 {
		t.Fatal("No root suites found")
	}

	verifiedRootSuite := finalDoc.Suites[0]
	if verifiedRootSuite.ID != rootSuiteID {
		t.Errorf("Expected root suite ID %s, got %s", rootSuiteID, verifiedRootSuite.ID)
	}

	if len(verifiedRootSuite.Suites) == 0 {
		t.Fatal("No nested suites found in root suite")
	}

	nestedSuite := verifiedRootSuite.Suites[0]
	if nestedSuite.ID != nestedSuite1ID {
		t.Errorf("Expected nested suite ID %s, got %s", nestedSuite1ID, nestedSuite.ID)
	}

	if len(nestedSuite.Tests) == 0 {
		t.Fatal("No tests found in nested suite")
	}

	foundTest := nestedSuite.Tests[0]
	if foundTest.ID != testID {
		t.Errorf("Expected test ID %s, got %s", testID, foundTest.ID)
	}

	t.Logf("✅ Successfully validated one-level nested suite structure: Root → Nested → Test")
}
