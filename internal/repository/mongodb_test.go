package repository

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	testMongoContainer *mongodb.MongoDBContainer
	testMongoClient    *mongo.Client
	testCollection     *mongo.Collection
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start MongoDB container
	container, err := mongodb.RunContainer(ctx,
		testcontainers.WithImage("mongo:7.0"),
	)
	if err != nil {
		panic(err)
	}
	testMongoContainer = container

	// Get connection string
	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		panic(err)
	}

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connStr))
	if err != nil {
		panic(err)
	}
	testMongoClient = client
	testCollection = client.Database("observer_test").Collection("test_runs")

	// Run tests
	code := m.Run()

	// Cleanup
	if err := client.Disconnect(ctx); err != nil {
		panic(err)
	}
	if err := container.Terminate(ctx); err != nil {
		panic(err)
	}

	os.Exit(code)
}

func setupTest(t *testing.T) *MongoRepository {
	// Clear collection before each test
	ctx := context.Background()
	if err := testCollection.Drop(ctx); err != nil {
		t.Fatalf("Failed to drop collection: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Reduce noise in tests
	}))

	return NewMongoRepository(testCollection, logger)
}

func TestUpsertSuiteBegin_RootSuite(t *testing.T) {
	repo := setupTest(t)
	ctx := context.Background()

	now := time.Now()
	suite := &m.SuiteDocument{
		ID:              "suite-root-1",
		Name:            "Root Suite",
		Description:     "Root test suite",
		Status:          "RUNNING",
		Metadata:        map[string]interface{}{"env": "test"},
		TestSuiteSpecID: "spec-123",
		InitiatedBy:     "user@example.com",
		ProjectName:     "test-project",
		StartTime:       &now,
	}

	err := repo.UpsertSuiteBegin(ctx, suite, "")
	if err != nil {
		t.Fatalf("UpsertSuiteBegin failed: %v", err)
	}

	// Verify document was created
	var doc m.TestRunDocument
	err = testCollection.FindOne(ctx, bson.M{"_id": "suite-root-1"}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	if doc.ID != "suite-root-1" {
		t.Errorf("ID = %v, want suite-root-1", doc.ID)
	}
	if doc.Name != "Root Suite" {
		t.Errorf("Name = %v, want Root Suite", doc.Name)
	}
	if doc.Status != "RUNNING" {
		t.Errorf("Status = %v, want RUNNING", doc.Status)
	}
	if doc.ProjectName != "test-project" {
		t.Errorf("ProjectName = %v, want test-project", doc.ProjectName)
	}
}

func TestUpsertSuiteBegin_NestedSuite(t *testing.T) {
	repo := setupTest(t)
	ctx := context.Background()

	// First create root suite
	rootSuite := &m.SuiteDocument{
		ID:     "suite-root-2",
		Name:   "Root Suite",
		Status: "RUNNING",
	}
	err := repo.UpsertSuiteBegin(ctx, rootSuite, "")
	if err != nil {
		t.Fatalf("Failed to create root suite: %v", err)
	}

	// Create nested suite
	nestedSuite := &m.SuiteDocument{
		ID:     "suite-nested-1",
		Name:   "Nested Suite",
		Status: "RUNNING",
	}

	err = repo.UpsertSuiteBegin(ctx, nestedSuite, "suite-root-2")
	if err != nil {
		t.Fatalf("UpsertSuiteBegin for nested suite failed: %v", err)
	}

	// Verify nested suite was appended
	var doc m.TestRunDocument
	err = testCollection.FindOne(ctx, bson.M{"_id": "suite-root-2"}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	if len(doc.Suites) != 1 {
		t.Fatalf("Suites length = %v, want 1", len(doc.Suites))
	}
	if doc.Suites[0].ID != "suite-nested-1" {
		t.Errorf("Nested suite ID = %v, want suite-nested-1", doc.Suites[0].ID)
	}
}

func TestUpsertTestBegin_RootLevel(t *testing.T) {
	repo := setupTest(t)
	ctx := context.Background()

	// Create root suite
	suite := &m.SuiteDocument{
		ID:     "suite-root-3",
		Name:   "Root Suite",
		Status: "RUNNING",
	}
	err := repo.UpsertSuiteBegin(ctx, suite, "")
	if err != nil {
		t.Fatalf("Failed to create root suite: %v", err)
	}

	// Add test to root suite
	test := &m.TestDocument{
		ID:     "test-1",
		Title:  "Test Case 1",
		Status: "RUNNING",
	}

	err = repo.UpsertTestBegin(ctx, test, "suite-root-3")
	if err != nil {
		t.Fatalf("UpsertTestBegin failed: %v", err)
	}

	// Verify test was appended
	var doc m.TestRunDocument
	err = testCollection.FindOne(ctx, bson.M{"_id": "suite-root-3"}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	if len(doc.Tests) != 1 {
		t.Fatalf("Tests length = %v, want 1", len(doc.Tests))
	}
	if doc.Tests[0].ID != "test-1" {
		t.Errorf("Test ID = %v, want test-1", doc.Tests[0].ID)
	}
}

func TestUpsertStepBegin_DirectChild(t *testing.T) {
	repo := setupTest(t)
	ctx := context.Background()

	// Create root suite and test
	suite := &m.SuiteDocument{
		ID:     "suite-root-4",
		Name:   "Root Suite",
		Status: "RUNNING",
	}
	err := repo.UpsertSuiteBegin(ctx, suite, "")
	if err != nil {
		t.Fatalf("Failed to create root suite: %v", err)
	}

	test := &m.TestDocument{
		ID:     "test-2",
		Title:  "Test with steps",
		Status: "RUNNING",
	}
	err = repo.UpsertTestBegin(ctx, test, "suite-root-4")
	if err != nil {
		t.Fatalf("Failed to create test: %v", err)
	}

	// Add step to test
	step := &m.StepDocument{
		ID:       "step-1",
		Status:   "RUNNING",
		Category: "click",
		Title:    "Click button",
	}

	err = repo.UpsertStepBegin(ctx, step, "test-2", "")
	if err != nil {
		t.Fatalf("UpsertStepBegin failed: %v", err)
	}

	// Verify step was appended
	var doc m.TestRunDocument
	err = testCollection.FindOne(ctx, bson.M{"_id": "suite-root-4"}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	if len(doc.Tests) != 1 || len(doc.Tests[0].Steps) != 1 {
		t.Fatal("Steps not as expected")
	}
	if doc.Tests[0].Steps[0].ID != "step-1" {
		t.Errorf("Step ID = %v, want step-1", doc.Tests[0].Steps[0].ID)
	}
}

func TestGetTestRun(t *testing.T) {
	repo := setupTest(t)
	ctx := context.Background()

	// Create a test run
	suite := &m.SuiteDocument{
		ID:     "suite-root-5",
		Name:   "Test Run",
		Status: "PASSED",
	}
	err := repo.UpsertSuiteBegin(ctx, suite, "")
	if err != nil {
		t.Fatalf("Failed to create suite: %v", err)
	}

	// Retrieve it
	doc, err := repo.GetTestRun(ctx, "suite-root-5")
	if err != nil {
		t.Fatalf("GetTestRun failed: %v", err)
	}

	if doc == nil {
		t.Fatal("GetTestRun returned nil")
	}
	if doc.ID != "suite-root-5" {
		t.Errorf("ID = %v, want suite-root-5", doc.ID)
	}
}

func TestListTestRuns(t *testing.T) {
	repo := setupTest(t)
	ctx := context.Background()

	// Create multiple test runs
	for i := 0; i < 5; i++ {
		suite := &m.SuiteDocument{
			ID:     fmt.Sprintf("suite-%d", i),
			Name:   fmt.Sprintf("Suite %d", i),
			Status: "PASSED",
		}
		err := repo.UpsertSuiteBegin(ctx, suite, "")
		if err != nil {
			t.Fatalf("Failed to create suite %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// List with pagination
	docs, count, err := repo.ListTestRuns(ctx, bson.M{}, 3, 0)
	if err != nil {
		t.Fatalf("ListTestRuns failed: %v", err)
	}

	if count != 5 {
		t.Errorf("Count = %v, want 5", count)
	}
	if len(docs) != 3 {
		t.Errorf("Docs length = %v, want 3 (limit)", len(docs))
	}
}

func TestConcurrentUpserts(t *testing.T) {
	repo := setupTest(t)
	ctx := context.Background()

	// Create root suite
	suite := &m.SuiteDocument{
		ID:     "suite-concurrent-1",
		Name:   "Concurrent Suite",
		Status: "RUNNING",
	}
	err := repo.UpsertSuiteBegin(ctx, suite, "")
	if err != nil {
		t.Fatalf("Failed to create root suite: %v", err)
	}

	// Concurrently add tests
	const numTests = 10
	errChan := make(chan error, numTests)

	for i := 0; i < numTests; i++ {
		go func(index int) {
			test := &m.TestDocument{
				ID:     fmt.Sprintf("test-concurrent-%d", index),
				Title:  fmt.Sprintf("Test %d", index),
				Status: "RUNNING",
			}
			errChan <- repo.UpsertTestBegin(ctx, test, "suite-concurrent-1")
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numTests; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent upsert failed: %v", err)
		}
	}

	// Verify all tests were added
	var doc m.TestRunDocument
	err = testCollection.FindOne(ctx, bson.M{"_id": "suite-concurrent-1"}).Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	if len(doc.Tests) != numTests {
		t.Errorf("Tests length = %v, want %d", len(doc.Tests), numTests)
	}
}
