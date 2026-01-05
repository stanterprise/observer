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
