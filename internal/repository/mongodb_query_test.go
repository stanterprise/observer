package repository

import (
	"context"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// setupTestRepo creates a test repository with a MongoDB testcontainer
func setupTestRepo(t *testing.T) (*MongoRepository, func()) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := mongodb.RunContainer(ctx, testcontainers.WithImage("mongo:7.0"))
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %v", err)
	}

	mongoURI, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		mongoContainer.Terminate(ctx)
		t.Fatalf("Failed to get MongoDB connection string: %v", err)
	}

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		mongoContainer.Terminate(ctx)
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	dbName := "observer_test_" + time.Now().Format("20060102150405")
	collection := client.Database(dbName).Collection("test_runs")
	repo := NewMongoRepository(collection, nil)

	cleanup := func() {
		client.Database(dbName).Drop(context.Background())
		client.Disconnect(context.Background())
		mongoContainer.Terminate(context.Background())
	}

	return repo, cleanup
}

func TestMongoRepository_SuiteExists_RootLevel(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a root suite document
	rootSuiteID := "run-123-suite-root"
	doc := &m.TestRunDocument{
		ID:        rootSuiteID,
		Name:      "Root Suite",
		Status:    "running",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Tests:     []*m.TestDocument{},
		Suites:    []*m.SuiteDocument{},
	}

	_, err := repo.collection.InsertOne(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to insert test document: %v", err)
	}

	// Test: Root suite should exist
	exists, err := repo.SuiteExists(ctx, rootSuiteID)
	if err != nil {
		t.Fatalf("SuiteExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected root suite to exist")
	}

	// Test: Non-existent suite should not exist
	exists, err = repo.SuiteExists(ctx, "nonexistent-suite")
	if err != nil {
		t.Fatalf("SuiteExists failed: %v", err)
	}
	if exists {
		t.Error("Expected non-existent suite to not exist")
	}
}

func TestMongoRepository_SuiteExists_Nested(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a document with nested suite
	rootSuiteID := "run-456-suite-root"
	nestedSuiteID := "run-456-suite-/path/to/nested"
	doc := &m.TestRunDocument{
		ID:        rootSuiteID,
		Name:      "Root Suite",
		Status:    "running",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Tests:     []*m.TestDocument{},
		Suites: []*m.SuiteDocument{
			{
				ID:        nestedSuiteID,
				Name:      "Nested Suite",
				Status:    "running",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Tests:     []*m.TestDocument{},
			},
		},
	}

	_, err := repo.collection.InsertOne(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to insert test document: %v", err)
	}

	// Test: Nested suite should exist
	exists, err := repo.SuiteExists(ctx, nestedSuiteID)
	if err != nil {
		t.Fatalf("SuiteExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected nested suite to exist")
	}
}

func TestMongoRepository_TestExists_RootLevel(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a document with root-level test
	rootSuiteID := "run-789-suite-root"
	testID := "run-789-test-1"
	doc := &m.TestRunDocument{
		ID:        rootSuiteID,
		Name:      "Root Suite",
		Status:    "running",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Tests: []*m.TestDocument{
			{
				ID:        testID,
				Title:     "Test 1",
				Status:    "running",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Steps:     []*m.StepDocument{},
			},
		},
		Suites: []*m.SuiteDocument{},
	}

	_, err := repo.collection.InsertOne(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to insert test document: %v", err)
	}

	// Test: Root-level test should exist
	exists, err := repo.TestExists(ctx, testID)
	if err != nil {
		t.Fatalf("TestExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected root-level test to exist")
	}

	// Test: Non-existent test should not exist
	exists, err = repo.TestExists(ctx, "nonexistent-test")
	if err != nil {
		t.Fatalf("TestExists failed: %v", err)
	}
	if exists {
		t.Error("Expected non-existent test to not exist")
	}
}

func TestMongoRepository_TestExists_InSuite(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a document with test in nested suite
	rootSuiteID := "run-abc-suite-root"
	testID := "run-abc-test-nested"
	doc := &m.TestRunDocument{
		ID:        rootSuiteID,
		Name:      "Root Suite",
		Status:    "running",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Tests:     []*m.TestDocument{},
		Suites: []*m.SuiteDocument{
			{
				ID:        "run-abc-suite-/nested",
				Name:      "Nested Suite",
				Status:    "running",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Tests: []*m.TestDocument{
					{
						ID:        testID,
						Title:     "Nested Test",
						Status:    "running",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Steps:     []*m.StepDocument{},
					},
				},
			},
		},
	}

	_, err := repo.collection.InsertOne(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to insert test document: %v", err)
	}

	// Test: Test in nested suite should exist
	exists, err := repo.TestExists(ctx, testID)
	if err != nil {
		t.Fatalf("TestExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected nested test to exist")
	}
}

func TestMongoRepository_StepExists(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a document with test containing steps
	rootSuiteID := "run-def-suite-root"
	testID := "run-def-test-1"
	stepID := "run-def-step-1"
	doc := &m.TestRunDocument{
		ID:        rootSuiteID,
		Name:      "Root Suite",
		Status:    "running",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Tests: []*m.TestDocument{
			{
				ID:        testID,
				Title:     "Test 1",
				Status:    "running",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Steps: []*m.StepDocument{
					{
						ID:        stepID,
						Title:     "Step 1",
						Status:    "running",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			},
		},
		Suites: []*m.SuiteDocument{},
	}

	_, err := repo.collection.InsertOne(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to insert test document: %v", err)
	}

	// Test: Step should exist
	exists, err := repo.StepExists(ctx, stepID)
	if err != nil {
		t.Fatalf("StepExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected step to exist")
	}

	// Test: Non-existent step should not exist
	exists, err = repo.StepExists(ctx, "nonexistent-step")
	if err != nil {
		t.Fatalf("StepExists failed: %v", err)
	}
	if exists {
		t.Error("Expected non-existent step to not exist")
	}
}

func TestMongoRepository_StepExists_InNestedSuite(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a document with step in nested suite test
	rootSuiteID := "run-ghi-suite-root"
	stepID := "run-ghi-step-nested"
	doc := &m.TestRunDocument{
		ID:        rootSuiteID,
		Name:      "Root Suite",
		Status:    "running",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Tests:     []*m.TestDocument{},
		Suites: []*m.SuiteDocument{
			{
				ID:        "run-ghi-suite-/nested",
				Name:      "Nested Suite",
				Status:    "running",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Tests: []*m.TestDocument{
					{
						ID:        "run-ghi-test-nested",
						Title:     "Nested Test",
						Status:    "running",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Steps: []*m.StepDocument{
							{
								ID:        stepID,
								Title:     "Nested Step",
								Status:    "running",
								CreatedAt: time.Now(),
								UpdatedAt: time.Now(),
							},
						},
					},
				},
			},
		},
	}

	_, err := repo.collection.InsertOne(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to insert test document: %v", err)
	}

	// Test: Step in nested suite test should exist
	exists, err := repo.StepExists(ctx, stepID)
	if err != nil {
		t.Fatalf("StepExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected nested step to exist")
	}
}

func TestMongoRepository_GetTestTrends(t *testing.T) {
repo, cleanup := setupTestRepo(t)
defer cleanup()

ctx := context.Background()

// Create multiple test runs with the same test ID to simulate historical data
testID := "test-auth-login"
baseTime := time.Now().Add(-24 * time.Hour)

// Run 1: Test in root level, PASSED
doc1 := &m.TestRunDocument{
ID:        "run-001",
Name:      "Run 1",
Status:    "completed",
CreatedAt: baseTime,
UpdatedAt: baseTime,
Tests: []*m.TestDocument{
{
ID:        testID,
Title:     "Login Test",
Status:    "PASSED",
Duration:  int64Ptr(2000000000), // 2s in nanoseconds
StartTime: timePtr(baseTime),
EndTime:   timePtr(baseTime.Add(2 * time.Second)),
CreatedAt: baseTime,
UpdatedAt: baseTime,
Steps:     []*m.StepDocument{},
},
},
Suites: []*m.SuiteDocument{},
}

// Run 2: Test in nested suite, FAILED
run2Time := baseTime.Add(1 * time.Hour)
doc2 := &m.TestRunDocument{
ID:        "run-002",
Name:      "Run 2",
Status:    "completed",
CreatedAt: run2Time,
UpdatedAt: run2Time,
Tests:     []*m.TestDocument{},
Suites: []*m.SuiteDocument{
{
ID:        "suite-auth",
Name:      "Auth Suite",
Status:    "completed",
CreatedAt: run2Time,
UpdatedAt: run2Time,
Tests: []*m.TestDocument{
{
ID:        testID,
Title:     "Login Test",
Status:    "FAILED",
Duration:  int64Ptr(3500000000), // 3.5s in nanoseconds
StartTime: timePtr(run2Time),
EndTime:   timePtr(run2Time.Add(3500 * time.Millisecond)),
CreatedAt: run2Time,
UpdatedAt: run2Time,
Steps:     []*m.StepDocument{},
},
},
},
},
}

// Run 3: Test in root level, PASSED
run3Time := baseTime.Add(2 * time.Hour)
doc3 := &m.TestRunDocument{
ID:        "run-003",
Name:      "Run 3",
Status:    "completed",
CreatedAt: run3Time,
UpdatedAt: run3Time,
Tests: []*m.TestDocument{
{
ID:        testID,
Title:     "Login Test",
Status:    "PASSED",
Duration:  int64Ptr(1800000000), // 1.8s in nanoseconds
StartTime: timePtr(run3Time),
EndTime:   timePtr(run3Time.Add(1800 * time.Millisecond)),
CreatedAt: run3Time,
UpdatedAt: run3Time,
Steps:     []*m.StepDocument{},
},
},
Suites: []*m.SuiteDocument{},
}

// Insert all test runs
_, err := repo.collection.InsertOne(ctx, doc1)
if err != nil {
t.Fatalf("Failed to insert doc1: %v", err)
}
_, err = repo.collection.InsertOne(ctx, doc2)
if err != nil {
t.Fatalf("Failed to insert doc2: %v", err)
}
_, err = repo.collection.InsertOne(ctx, doc3)
if err != nil {
t.Fatalf("Failed to insert doc3: %v", err)
}

// Test: Get trends for the test ID
trends, err := repo.GetTestTrends(ctx, testID, 50)
if err != nil {
t.Fatalf("GetTestTrends failed: %v", err)
}

// Verify we got 3 trends (one per run)
if len(trends) != 3 {
t.Errorf("Expected 3 trends, got %d", len(trends))
}

// Verify trends are sorted by createdAt descending (newest first)
if len(trends) >= 2 {
if trends[0].CreatedAt.Before(trends[1].CreatedAt) {
t.Error("Expected trends to be sorted by createdAt descending")
}
}

// Verify the most recent trend (run-003)
if len(trends) > 0 {
latest := trends[0]
if latest.TestID != testID {
t.Errorf("Expected testId %s, got %s", testID, latest.TestID)
}
if latest.RunID != "run-003" {
t.Errorf("Expected runId run-003, got %s", latest.RunID)
}
if latest.Status != "PASSED" {
t.Errorf("Expected status PASSED, got %s", latest.Status)
}
if latest.Duration == nil || *latest.Duration != 1800000000 {
t.Errorf("Expected duration 1800000000, got %v", latest.Duration)
}
}

// Verify the middle trend (run-002, nested suite)
if len(trends) > 1 {
middle := trends[1]
if middle.RunID != "run-002" {
t.Errorf("Expected runId run-002, got %s", middle.RunID)
}
if middle.Status != "FAILED" {
t.Errorf("Expected status FAILED, got %s", middle.Status)
}
}

// Verify the oldest trend (run-001)
if len(trends) > 2 {
oldest := trends[2]
if oldest.RunID != "run-001" {
t.Errorf("Expected runId run-001, got %s", oldest.RunID)
}
if oldest.Status != "PASSED" {
t.Errorf("Expected status PASSED, got %s", oldest.Status)
}
}

// Test: Limit parameter works
limitedTrends, err := repo.GetTestTrends(ctx, testID, 2)
if err != nil {
t.Fatalf("GetTestTrends with limit failed: %v", err)
}
if len(limitedTrends) != 2 {
t.Errorf("Expected 2 trends with limit, got %d", len(limitedTrends))
}

// Test: Non-existent test ID returns empty array
emptyTrends, err := repo.GetTestTrends(ctx, "nonexistent-test", 50)
if err != nil {
t.Fatalf("GetTestTrends for nonexistent test failed: %v", err)
}
if len(emptyTrends) != 0 {
t.Errorf("Expected 0 trends for nonexistent test, got %d", len(emptyTrends))
}
}

// Helper functions for test data
func int64Ptr(v int64) *int64 {
return &v
}

func timePtr(t time.Time) *time.Time {
return &t
}
