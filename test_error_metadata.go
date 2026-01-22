package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/stanterprise/observer/internal/database"
	"github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
)

func main() {
	// Connect to MongoDB
	ctx := context.Background()
	mongoDB, err := database.ConnectMongoDBFromEnv(nil)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoDB.Close(ctx)

	repo := repository.NewMongoRepository(mongoDB.TestRunsCollection(), nil)

	// Create a test run
	runID := "test-error-metadata-run"
	testID := "test-error-metadata-test"
	stepID := "test-error-metadata-step"

	// Create run
	startTime := time.Now()
	err = repo.UpsertRunBegin(ctx, runID, &models.TestRunDocument{
		ID:        runID,
		Name:      "Test Run for Error Metadata",
		StartTime: &startTime,
	})
	if err != nil {
		log.Fatalf("Failed to create run: %v", err)
	}

	// Create test
	err = repo.UpsertTestBegin(ctx, runID, testID, "Test with Error Metadata", &startTime, 0)
	if err != nil {
		log.Fatalf("Failed to create test: %v", err)
	}

	// Create step begin
	stepBegin := &models.StepDocument{
		ID:            stepID,
		RunID:         runID,
		TestCaseRunID: testID,
		Title:         "Step with Error",
		StartTime:     &startTime,
	}
	err = repo.UpsertStepBegin(ctx, runID, stepBegin, testID, 0)
	if err != nil {
		log.Fatalf("Failed to create step: %v", err)
	}

	// Update step end with error metadata
	errorMetadata := map[string]interface{}{
		"error_stack":    "Error: Test failed\n  at test.spec.ts:45:12\n  at Worker.run:10:5",
		"error_value":    "Navigation timeout after 30000ms",
		"error_snippet":  "await page.click('#submit');\nawait page.waitForNavigation();",
		"error_location": "tests/e2e/login.spec.ts:45:12",
	}
	errorMsg := "Timeout 30000ms exceeded"
	errors := []string{"Timeout 30000ms exceeded", "Navigation failed"}
	duration := int64(30000000000) // 30 seconds in nanoseconds

	err = repo.UpsertStepEnd(ctx, runID, stepID, testID, 0, "FAILED", errorMetadata, errorMsg, errors, &duration)
	if err != nil {
		log.Fatalf("Failed to update step end: %v", err)
	}

	fmt.Println("✓ Successfully stored step with error metadata")
	fmt.Printf("  Run ID: %s\n", runID)
	fmt.Printf("  Test ID: %s\n", testID)
	fmt.Printf("  Step ID: %s\n", stepID)
	fmt.Println("\nYou can verify the data in MongoDB:")
	fmt.Printf("  mongosh --eval 'db.test_runs.findOne({_id: \"%s\"})'\n", runID)
}
